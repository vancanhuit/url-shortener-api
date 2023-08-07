package main

import (
	"fmt"
	"log"
	"net/http"
)

func errorResponse(w http.ResponseWriter, status int, message interface{}) {
	data := map[string]interface{}{"error": message}
	err := writeJSON(w, status, data)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func serverErrorResponse(w http.ResponseWriter, err error) {
	log.Println(err)
	message := "the server encountered a problem and could not process your request"
	errorResponse(w, http.StatusInternalServerError, message)
}

func notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	errorResponse(w, http.StatusNotFound, message)
}

func methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	errorResponse(w, http.StatusMethodNotAllowed, message)
}

func badRequestResponse(w http.ResponseWriter, err error) {
	errorResponse(w, http.StatusBadRequest, err.Error())
}

func failedValidationResponse(w http.ResponseWriter, errors map[string]string) {
	errorResponse(w, http.StatusUnprocessableEntity, errors)
}
