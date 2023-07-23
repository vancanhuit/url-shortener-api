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
	"time"

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

func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
	ts := httptest.NewServer(h)
	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return ts
}

func TestAPI(t *testing.T) {
	resource, err := startPostgreSQL()
	require.NoError(t, err)
	t.Cleanup(func() {
		if err := pool.Purge(resource); err != nil {
			log.Printf("failed to purge resource: %s", err)
		}
	})
	dsn := fmt.Sprintf("postgres://test:test@localhost:%s/test?sslmode=disable", resource.GetPort("5432/tcp"))
	log.Printf("connecting to database: %s", dsn)

	resource.Expire(120)
	pool.MaxWait = 120 * time.Second
	var db *sql.DB
	err = pool.Retry(func() error {
		db, err = openDB(dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	})
	require.NoError(t, err)

	err = migrateDB(db)
	require.NoError(t, err)

	app := &application{service: service{db: db}}
	ts := newTestServer(t, app.routes())
	defer ts.Close()

	resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", strings.NewReader(`{"url": "https://reddit.com"}`))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var envelope struct {
		Data struct {
			model
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = json.Unmarshal(body, &envelope)
	require.NoError(t, err)

	require.Equal(t, "https://reddit.com", envelope.Data.OriginalURL)

	resp, err = ts.Client().Get(ts.URL + "/" + envelope.Data.Alias)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusFound, resp.StatusCode)
	require.Equal(t, envelope.Data.OriginalURL, resp.Header.Get("Location"))
}
