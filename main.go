package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type CreateRunRequest struct {
	SweepID  *int64
	Metadata json.RawMessage
}

type CreateSweepRequest struct {
	Method     string
	Parameters map[string][]json.RawMessage
	Metadata   json.RawMessage
}

type CreateLogRequest struct {
	RunID    int64
	log json.RawMessage
}

func addRoutes(r *gin.Engine) {
	r.POST("/create-sweep", func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)

		var request CreateSweepRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		var gridIndex *int
		if request.Method == "grid" {
			gridIndex = new(int)
		}

		if request.Metadata == nil {
			request.Metadata = json.RawMessage("{}")
		}

		var sweepID int64
		if err := tx.Get(&sweepID, `
		INSERT INTO sweep (
			gridIndex,
			Metadata
		) VALUES ($1, $2) returning id
		`, gridIndex, request.Metadata); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for key, values := range request.Parameters {
			var serializedValues []string
			for _, value := range values {
				serializedValue, err := json.Marshal(value)
				if err != nil {
					c.AbortWithError(http.StatusBadRequest, err)
					return
				}
				serializedValues = append(serializedValues, string(serializedValue))
			}
			if _, err := tx.Exec(`
			INSERT INTO sweep_parameter (
				SweepID, 
				"Key", 
				"Values"
			) VALUES ($1, $2, $3)
			`, sweepID, key, pq.Array(serializedValues)); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(200, gin.H{
			"SweepID": sweepID,
		})
	})
	r.POST("/create-run", func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)
		var request CreateRunRequest
		if err := c.BindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		if request.Metadata == nil {
			request.Metadata = json.RawMessage("{}")
		}

		var runID int64
		if err := tx.Get(&runID, `
		INSERT INTO run (
			Metadata,
      SweepID
		) VALUES ($1, $2) returning id
		`, request.Metadata, request.SweepID); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if request.SweepID == nil {
			c.JSON(200, gin.H{
				"RunID": runID,
			})
		} else {
			var sweep struct {
				GridIndex      *int64
				ParametersJSON string
			}
			if err := tx.Get(&sweep, `
			SELECT
				GridIndex,
				JSON_OBJECT_AGG("Key", "Values") AS ParametersJSON
			FROM sweep
			JOIN sweep_parameter ON SweepID = sweep.ID
			WHERE sweep.ID = $1
			GROUP BY sweep.ID
			`, request.SweepID); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			var parameters map[string][]json.RawMessage
			if err := json.Unmarshal([]byte(sweep.ParametersJSON), &parameters); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}

			choices := make(map[string]int)
			if sweep.GridIndex == nil {
				for key, values := range parameters {
					choices[key] = rand.Intn(len(values))
				}
			} else {
				var parameterNames []string
				for name := range parameters {
					parameterNames = append(parameterNames, name)
				}
				sort.Strings(parameterNames)
				limits := make([]int, len(parameters))
				for i, key := range parameterNames {
					limits[i] = len(parameters[key])
				}
				parameterIndices := chooseNth(int(*sweep.GridIndex), limits)
				for i, key := range parameterNames {
					choices[key] = parameterIndices[i]
				}

				if _, err := tx.Exec(`
				UPDATE sweep SET gridIndex = $1 WHERE ID = $2
					`, *sweep.GridIndex+1, request.SweepID); err != nil {
					c.AbortWithError(http.StatusInternalServerError, err)
					return
				}

			}
			chosenParameters := make(map[string]json.RawMessage)
			for key, choice := range choices {
				chosenParameters[key] = parameters[key][choice]
			}

			c.JSON(200, gin.H{
				"RunID":      runID,
				"Parameters": chosenParameters,
			})

		}

		if err := tx.Commit(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

	})
	r.POST("/create-log", func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)
		var request CreateLogRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var logID int64
		if err := db.Get(&logID, `
		INSERT INTO run_log (
			RunID,
			log
		) VALUES ($1, $2) RETURNING id
		`, request.RunID, request.log); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(200, gin.H{
			"LogID": logID,
		})
	})
}

func main() {
	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("user=postgres dbname=postgres password=%s port=5432 host=localhost sslmode=disable", os.Getenv("PGPASSWORD")))
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	addRoutes(r)
	log.Fatal(r.Run())
}
