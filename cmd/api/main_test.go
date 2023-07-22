package main

// func newTestServer(t *testing.T, h http.Handler) *httptest.Server {
// 	ts := httptest.NewServer(h)
// 	ts.Client().CheckRedirect = func(req *http.Request, via []*http.Request) error {
// 		return http.ErrUseLastResponse
// 	}
// 	return ts
// }

// func TestValidInput(t *testing.T) {
// 	ts := newTestServer(t, router())
// 	defer ts.Close()

// 	url := "https://reddit.com"
// 	payload := strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, url))
// 	resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", payload)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	require.Equal(t, http.StatusCreated, resp.StatusCode)
// 	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
// 	body, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)
// 	var envelope struct {
// 		Data struct {
// 			OriginalURL string `json:"original_url"`
// 			Alias       string `json:"alias"`
// 		} `json:"data"`
// 	}

// 	err = json.Unmarshal(body, &envelope)
// 	require.NoError(t, err)
// 	require.Equal(t, url, envelope.Data.OriginalURL)

// 	resp, err = ts.Client().Get(ts.URL + "/" + envelope.Data.Alias)
// 	require.NoError(t, err)
// 	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
// 	require.Equal(t, envelope.Data.OriginalURL, resp.Header.Get("Location"))
// }

// func TestInvalidInput(t *testing.T) {
// 	ts := newTestServer(t, app.routes())
// 	defer ts.Close()

// 	t.Run("Invalid JSON", func(t *testing.T) {
// 		payload := strings.NewReader(`{"url": "https://}`)
// 		resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", payload)
// 		require.NoError(t, err)
// 		defer resp.Body.Close()
// 		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
// 	})

// 	t.Run("Empty URL", func(t *testing.T) {
// 		payload := strings.NewReader(`{"url": ""}`)
// 		resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", payload)
// 		require.NoError(t, err)
// 		defer resp.Body.Close()
// 		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
// 	})

// 	t.Run("URL is not supplied", func(t *testing.T) {
// 		payload := strings.NewReader(`{}`)
// 		resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", payload)
// 		require.NoError(t, err)
// 		defer resp.Body.Close()
// 		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
// 	})

// 	t.Run("Invalid URL", func(t *testing.T) {
// 		payload := strings.NewReader(`{"url": "https://"}`)
// 		resp, err := ts.Client().Post(ts.URL+"/api/shorten", "application/json", payload)
// 		require.NoError(t, err)
// 		defer resp.Body.Close()
// 		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
// 	})

// 	t.Run("Invalid alias", func(t *testing.T) {
// 		alias := "non-existing-alias"
// 		resp, err := ts.Client().Get(ts.URL + "/" + alias)
// 		require.NoError(t, err)
// 		defer resp.Body.Close()
// 		require.Equal(t, http.StatusNotFound, resp.StatusCode)
// 	})
// }
