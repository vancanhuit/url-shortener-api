package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) shorten(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := readJSON(w, r, &req); err != nil {
		badRequestResponse(w, r, err)
		return
	}

	v := &validator{errors: make(map[string]string)}
	if validateURL(v, req.URL); !v.valid() {
		failedValidationResponse(w, r, v.errors)
		return
	}

	alias, err := app.service.createAlias(req.URL)
	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}

	data := map[string]interface{}{
		"data": model{
			OriginalURL: req.URL,
			Alias:       alias,
		},
	}

	if err := writeJSON(w, http.StatusCreated, data); err != nil {
		serverErrorResponse(w, r, err)
	}

}

func (app *application) redirect(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	v := &validator{errors: make(map[string]string)}
	if validateAlias(v, alias); !v.valid() {
		failedValidationResponse(w, r, v.errors)
		return
	}
	url, err := app.service.getURL(alias)
	if err != nil {
		serverErrorResponse(w, r, err)
		return
	}
	if url == "" {
		notFoundResponse(w, r)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}
