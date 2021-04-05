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
	CommitHash  string
	Command     string
	Description *string
}

type CreateSweepRequest struct {
	Method      string
	Parameters  map[string][]json.RawMessage
	Description *string
}

type LogDocumentRequest struct {
	RunID    int64
	Document json.RawMessage
}
type IterateHyperparametersRequest struct{ SweepID int64 }

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

		var sweepID int64
		if err := tx.Get(&sweepID, `
		INSERT INTO sweep (
			Method, 
			Description
		) VALUES ($1, $2) returning id
		`, request.Method, request.Description); err != nil {
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
		var json CreateRunRequest
		if err := c.BindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var runID int64
		if err := db.Get(&runID, `
		INSERT INTO run (
			CommitHash,
			Command, 
			Description
		) VALUES ($1, $2, $3) returning id
		`, json.CommitHash, json.Command, json.Description); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(200, gin.H{
			"RunID": runID,
		})
	})
	r.POST("/log-document", func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)
		var request LogDocumentRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var logID int64
		if err := db.Get(&logID, `
		INSERT INTO run_log (
			RunID,
			Document
		) VALUES ($1, $2) RETURNING id
		`, request.RunID, request.Document); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		c.JSON(200, gin.H{
			"LogID": logID,
		})
	})
	r.POST("/iterate-hyperparameters", func(c *gin.Context) {
		db := c.MustGet("db").(*sqlx.DB)
		var request IterateHyperparametersRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer tx.Rollback()

		var sweep struct {
			Method         string
			GridIndex      pq.Int32Array
			ParametersJSON string
		}
		if err := tx.Get(&sweep, `
			SELECT
				Method,
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

		var parameterNames []string
		for name := range parameters {
			parameterNames = append(parameterNames, name)
		}
		sort.Strings(parameterNames)

		chosenParameters := make(map[string]json.RawMessage)

		if sweep.Method == "grid" {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("not implemented"))
			return
		} else if sweep.Method == "random" {
			for key, values := range parameters {
				chosenParameters[key] = values[rand.Intn(len(values))]
			}
		} else {
			c.AbortWithError(http.StatusInternalServerError, fmt.Errorf("invalid method %q", sweep.Method))
			return
		}

		if err := tx.Commit(); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.JSON(200, gin.H{
			"Parameters": chosenParameters,
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
