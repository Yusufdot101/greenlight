package main

import (
	"fmt"
	"net/http"
)

func (app *application) logError(err error, properties map[string]string) {
	app.logger.PrintError(err, properties)
}

func (app *application) errorResponse(w http.ResponseWriter, statusCode int, message any) {
	err := app.writeJSON(w, statusCode, envelope{"error": message})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(err, map[string]string{"method": r.Method})

	message := "the server encountered and error and could not resolve your request"
	app.errorResponse(w, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the resource you requested for could not be found"
	app.errorResponse(w, http.StatusNotFound, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not allowed for this resource", r.Method)
	app.errorResponse(w, http.StatusMethodNotAllowed, message)
}

func (app *application) badRequestResponse(w http.ResponseWriter, err error) {
	app.errorResponse(w, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(w http.ResponseWriter, err map[string]string) {
	app.errorResponse(w, http.StatusBadRequest, err)
}

func (app *application) editConflictResponse(w http.ResponseWriter) {
	message := "an error occcured and your edit did not go through, please try again"
	app.errorResponse(w, http.StatusConflict, message)
}

func (app *application) rateLimitExceededResponse(w http.ResponseWriter) {
	message := "rate limit exceeded"
	app.errorResponse(w, http.StatusTooManyRequests, message)
}
