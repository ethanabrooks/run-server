package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestMain(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := sqlx.Connect("postgres",
		fmt.Sprintf("user=postgres dbname=postgres password=%s port=5432 host=localhost sslmode=disable", os.Getenv("PGPASSWORD")))
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	engine := gin.Default()
	engine.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})
	addRoutes(engine)

	server := httptest.NewServer(engine)
	client := server.Client()

	var sweepID int64
	{
		data, err := json.Marshal(CreateSweepRequest{
			Method: "random",
			Parameters: map[string][]json.RawMessage{
				"param1": []json.RawMessage{
					json.RawMessage([]byte(`{"foo1.1": "bar1.1"}`)),
					json.RawMessage([]byte(`{"foo1.2": "bar1.2"}`)),
					json.RawMessage([]byte(`{"foo1.3": "bar1.3"}`)),
				},
				"param2": []json.RawMessage{
					json.RawMessage([]byte(`{"foo2.1": "bar2.1"}`)),
					json.RawMessage([]byte(`{"foo2.2": "bar2.2"}`)),
					json.RawMessage([]byte(`{"foo2.3": "bar2.3"}`)),
				},
				"param3": []json.RawMessage{
					json.RawMessage([]byte(`{"foo3.1": "bar3.1"}`)),
					json.RawMessage([]byte(`{"foo3.2": "bar3.2"}`)),
					json.RawMessage([]byte(`{"foo3.3": "bar3.3"}`)),
				},
			},
		})
		require.NoError(t, err)
		res, err := client.Post(server.URL+"/create-sweep", "application/json", bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)

		var response struct {
			SweepID int64
		}
		json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
		require.GreaterOrEqual(t, response.SweepID, int64(0))
		sweepID = response.SweepID
	}

	var runID int64
	{
		data, err := json.Marshal(CreateRunRequest{
			CommitHash: "asdf",
			Command:    "thud",
		})
		require.NoError(t, err)
		res, err := client.Post(server.URL+"/create-run", "application/json", bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)

		var response struct {
			RunID int64
		}
		json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
		require.Greater(t, response.RunID, int64(0))
		runID = response.RunID
	}

	{
		data, err := json.Marshal(LogDocumentRequest{
			RunID:    runID,
			Document: json.RawMessage([]byte(`{"key": "value"}`)),
		})
		require.NoError(t, err)
		res, err := client.Post(server.URL+"/log-document", "application/json", bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)

		var response struct {
			LogID int
		}
		json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
		require.Greater(t, response.LogID, 0)
	}

	{
		data, err := json.Marshal(IterateHyperparametersRequest{
			SweepID: sweepID,
		})
		require.NoError(t, err)
		res, err := client.Post(server.URL+"/iterate-hyperparameters", "application/json", bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, res.StatusCode)

		var response struct {
			Parameters map[string]json.RawMessage
		}
		json.NewDecoder(res.Body).Decode(&response)
		require.NoError(t, err)
		log.Printf("%q", response)
	}
}