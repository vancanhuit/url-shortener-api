package main

import (
	"fmt"
	"log"
	"net/http"
)

func errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	data := map[string]interface{}{"error": message}

	err := writeJSON(w, status, data)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Println(err)

	message := "the server encountered a problem and could not process your request"
	errorResponse(w, r, http.StatusInternalServerError, message)
}

func notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	errorResponse(w, r, http.StatusNotFound, message)
}

func methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}
