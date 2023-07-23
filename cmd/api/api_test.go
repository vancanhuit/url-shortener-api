package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
)

var pool *dockertest.Pool

func getPool() (*dockertest.Pool, error) {
	if pool != nil {
		return pool, nil
	}
	var err error
	pool, err = dockertest.NewPool("")
	return pool, err
}

func startPostgreSQL() (*dockertest.Resource, error) {
	pool, err := getPool()

	if err != nil {
		return nil, err
	}

	if err := pool.Client.Ping(); err != nil {
		return nil, err
	}

	return pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "15.3",
		Env: []string{
			"POSTGRES_USER=test",
			"POSTGRES_PASSWORD=test",
			"POSTGRES_DB=test",
		},
	})
}

func connectToTestDB(t *testing.T) (*sql.DB, error) {
	resource, err := startPostgreSQL()
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if err := pool.Purge(resource); err != nil {
			log.Printf("Failed to purge resource: %s", err)
		}
	})

	var db *sql.DB
	dsn := fmt.Sprintf("postgres://test:test@localhost:%s/test?sslmode=disable", resource.GetPort("5432/tcp"))
	log.Printf("Connecting to database: %s", dsn)

	resource.Expire(120)

	err = pool.Retry(func() error {
		db, err = openDB(dsn)
		return err
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
	ts := httptest.NewServer(h)
	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return ts
}

func TestAPI(t *testing.T) {
	db, err := connectToTestDB(t)
	require.NoError(t, err)
	err = migrateDB(db)
	require.NoError(t, err)

	app := &application{service: service{db: db}}
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	url := "https://reddit.com"
	reqBody := strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, url))

	resp1, err := ts.Client().Post(ts.URL+"/api/shorten", contentType, reqBody)
	require.NoError(t, err)
	defer resp1.Body.Close()
	// resp2, err := ts.Client().Post(ts.URL+"/api/shorten", contentType, reqBody)
	// require.NoError(t, err)
	// defer resp2.Body.Close()

	require.Equal(t, http.StatusCreated, resp1.StatusCode)
	require.Equal(t, contentType, resp1.Header.Get("Content-Type"))
	//require.Equal(t, http.StatusCreated, resp2.StatusCode)
	//require.Equal(t, contentType, resp2.Header.Get("Content-Type"))

	var envelope struct {
		Data struct {
			model
		} `json:"data"`
	}
	body1, err := io.ReadAll(resp1.Body)
	require.NoError(t, err)
	//body2, err := io.ReadAll(resp2.Body)
	//require.NoError(t, err)

	//require.Equal(t, body1, body2)

	err = json.Unmarshal(body1, &envelope)
	require.NoError(t, err)

	require.Equal(t, url, envelope.Data.OriginalURL)

	resp3, err := ts.Client().Get(ts.URL + "/" + envelope.Data.Alias)
	require.NoError(t, err)
	defer resp3.Body.Close()

	require.Equal(t, http.StatusFound, resp3.StatusCode)
	require.Equal(t, envelope.Data.OriginalURL, resp3.Header.Get("Location"))

	req, err := http.NewRequest(http.MethodDelete, ts.URL+"/"+envelope.Data.Alias, nil)
	require.NoError(t, err)
	resp4, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusNoContent, resp4.StatusCode)

	resp5, err := ts.Client().Get(ts.URL + "/" + envelope.Data.Alias)
	require.NoError(t, err)
	defer resp5.Body.Close()
	require.Equal(t, http.StatusNotFound, resp5.StatusCode)

	resp6, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp6.Body.Close()
	require.Equal(t, http.StatusNotFound, resp6.StatusCode)
}
