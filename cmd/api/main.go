package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var data = make(map[string]string)

func shorten(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if !govalidator.IsURL(req.URL) {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(req.URL))
	alias := base64.URLEncoding.EncodeToString(hash[:])[:7]
	data[alias] = req.URL

	resp := map[string]interface{}{
		"data": URL{
			OriginalURL: req.URL,
			Alias:       alias,
		},
	}

	payload, err := json.Marshal(&resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(payload)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	url, ok := data[alias]
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/shorten", shorten)
	r.Get("/{alias}", redirect)

	return r
}

func main() {
	http.ListenAndServe(":9000", router())
}
