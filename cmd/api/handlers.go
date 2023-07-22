package main

import (
	"encoding/json"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi/v5"
)

func (app *application) shorten(w http.ResponseWriter, r *http.Request) {
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

	alias, err := app.service.createAlias(req.URL)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	resp := map[string]interface{}{
		"data": model{
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

func (app *application) redirect(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	url, err := app.service.getURL(alias)
	if err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	if url == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, url, http.StatusSeeOther)
}
