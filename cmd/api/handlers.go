package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (app *application) shorten(w http.ResponseWriter, r *http.Request) {
	var input struct {
		URL string `json:"url"`
	}
	if err := readJSON(w, r, &input); err != nil {
		badRequestResponse(w, err)
		return
	}

	v := &validator{errors: make(map[string]string)}
	if validateURL(v, input.URL); !v.valid() {
		failedValidationResponse(w, v.errors)
		return
	}

	reqID := middleware.GetReqID(r.Context())

	alias, err := app.service.createAlias(input.URL, reqID)
	if err != nil {
		serverErrorResponse(w, err)
		return
	}

	data := map[string]interface{}{
		"data": model{
			OriginalURL: input.URL,
			Alias:       alias,
		},
	}

	if err := writeJSON(w, http.StatusCreated, data); err != nil {
		serverErrorResponse(w, err)
	}
}

func (app *application) delete(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	err := app.service.deleteURL(alias)
	if err != nil {
		switch {
		case errors.Is(err, errRecordNotFound):
			notFoundResponse(w, r)
		default:
			serverErrorResponse(w, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) redirect(w http.ResponseWriter, r *http.Request) {
	alias := chi.URLParam(r, "alias")
	url, err := app.service.getURL(alias)
	if err != nil {
		switch {
		case errors.Is(err, errRecordNotFound):
			notFoundResponse(w, r)
		default:
			serverErrorResponse(w, err)
		}
		return
	}

	http.Redirect(w, r, url, http.StatusFound)
}
