package main

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testServer struct {
	*httptest.Server
}

func randomDBName() string {
	var buf [20]byte
	_, _ = rand.Read(buf[:])
	return fmt.Sprintf("test_db_%x", buf)
}

func setupTestDB(t *testing.T, dbName string) string {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	db, err := openDB(dsn)
	require.NoError(t, err)
	parsedURL, err := url.Parse(dsn)
	require.NoError(t, err)
	parsedURL.Path = dbName

	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := db.Exec(fmt.Sprintf("DROP DATABASE %s", dbName))
		if err != nil {
			t.Logf("Failed to drop database %s: %v", dbName, err)
		}
		db.Close() //nolint:errcheck
	})
	return parsedURL.String()
}

func connectToTestDB(t *testing.T) (*sql.DB, error) {
	dbName := randomDBName()
	dsn := setupTestDB(t, dbName)
	db, err := openDB(dsn)
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
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	bytes.TrimSpace(body)

	return resp.StatusCode, resp.Header, body
}

func TestAPIWithValidInput(t *testing.T) {
	db, err := connectToTestDB(t)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close() //nolint:errcheck
	})
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
	t.Cleanup(func() {
		db.Close() //nolint:errcheck
	})
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
