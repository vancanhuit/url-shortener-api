package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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

type testServer struct {
	*httptest.Server
}

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

	err = pool.Retry(func() error {
		db, err = openDB(dsn)
		return err
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func newTestServer(h http.Handler) *testServer {
	ts := httptest.NewServer(h)
	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &testServer{ts}
}

func (ts *testServer) do(t *testing.T, method, urlPath string, reqBody io.Reader) (int, http.Header, []byte) {
	req, err := http.NewRequest(method, ts.URL+urlPath, reqBody)
	require.NoError(t, err)
	if req.Method == http.MethodPost {
		req.Header.Set(contentTypeHeader, contentType)
	}
	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bytes.TrimSpace(body)

	return resp.StatusCode, resp.Header, body
}

func TestAPIWithValidInput(t *testing.T) {
	db, err := connectToTestDB(t)
	require.NoError(t, err)
	err = migrateDB(db)
	require.NoError(t, err)

	app := &application{service: service{db: db}}
	ts := newTestServer(app.routes())
	defer ts.Close()

	url := "https://reddit.com"
	alias := ""

	var envelope struct {
		Data struct {
			model
		} `json:"data"`
	}

	reqBody := strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, url))
	statusCode, header, body := ts.do(t, http.MethodPost, "/api/shorten", reqBody)
	require.Equal(t, http.StatusCreated, statusCode)
	require.Equal(t, contentType, header.Get(contentTypeHeader))
	err = json.Unmarshal(body, &envelope)
	require.NoError(t, err)
	alias = envelope.Data.Alias
	require.Equal(t, url, envelope.Data.OriginalURL)

	reqBody = strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, url))
	statusCode, header, body = ts.do(t, http.MethodPost, "/api/shorten", reqBody)
	require.Equal(t, http.StatusCreated, statusCode)
	require.Equal(t, contentType, header.Get(contentTypeHeader))
	err = json.Unmarshal(body, &envelope)
	require.NoError(t, err)
	require.Equal(t, url, envelope.Data.OriginalURL)
	require.Equal(t, alias, envelope.Data.Alias)

	statusCode, header, _ = ts.do(t, http.MethodGet, "/"+envelope.Data.Alias, nil)
	require.Equal(t, http.StatusFound, statusCode)
	require.Equal(t, envelope.Data.OriginalURL, header.Get("Location"))

	statusCode, _, _ = ts.do(t, http.MethodDelete, "/"+envelope.Data.Alias, nil)
	require.Equal(t, http.StatusNoContent, statusCode)

	statusCode, header, _ = ts.do(t, http.MethodGet, "/"+envelope.Data.Alias, nil)
	require.Equal(t, http.StatusNotFound, statusCode)
	require.Equal(t, contentType, header.Get(contentTypeHeader))

	statusCode, header, _ = ts.do(t, http.MethodDelete, "/"+envelope.Data.Alias, nil)
	require.Equal(t, http.StatusNotFound, statusCode)
	require.Equal(t, contentType, header.Get(contentTypeHeader))
}

func TestAPIWithInvalidInput(t *testing.T) {
	db, err := connectToTestDB(t)
	require.NoError(t, err)
	err = migrateDB(db)
	require.NoError(t, err)

	app := &application{service: service{db: db}}
	ts := newTestServer(app.routes())
	defer ts.Close()

	largeData := make([]byte, maxBytes)
	_, err = rand.Read(largeData)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		payload    string
		statusCode int
	}{
		{
			name:       "Empty request body",
			payload:    ``,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Syntax error",
			payload:    `{"url": https://example}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Incorrect value type",
			payload:    `{"url": 123213}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Unexpected EOF",
			payload:    `{"url": "https://example}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Unknown key",
			payload:    `{"urls": "https://example.com"}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       `Multiple JSON values`,
			payload:    `{"url": "https://example.com"}{}{}`,
			statusCode: http.StatusBadRequest,
		},
		{
			name:       `Request too large`,
			payload:    fmt.Sprintf(`{"url": "https://%s"}`, base64.URLEncoding.EncodeToString(largeData)),
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Missing input",
			payload:    `{}`,
			statusCode: http.StatusUnprocessableEntity,
		},
		{
			name:       "Invalid URL",
			payload:    `{"url": "https://"}`,
			statusCode: http.StatusUnprocessableEntity,
		},
		{
			name:       "Length of URL is more than 500 bytes",
			payload:    fmt.Sprintf(`{"url": "https://%s"}`, base64.URLEncoding.EncodeToString(largeData[:500])),
			statusCode: http.StatusUnprocessableEntity,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := strings.NewReader(tc.payload)
			statusCode, header, _ := ts.do(t, http.MethodPost, "/api/shorten", reqBody)
			require.Equal(t, tc.statusCode, statusCode)
			require.Equal(t, contentType, header.Get(contentTypeHeader))
		})
	}
}
