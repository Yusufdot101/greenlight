package main

import (
	"fmt"
	"net/http"
)

func (app *application) logError(r *http.Request, err error) {
	app.logger.Println(err)
}

func (app *application) errorResponse(
	w http.ResponseWriter, r *http.Request,
	statusCode int, message any,
) {
	env := envelope{"error": message}

	err := app.writeJSON(w, statusCode, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (app *application) serverErrorResponse(
	w http.ResponseWriter, r *http.Request,
	err error,
) {
	app.logError(r, err)
	message := "the server encountered a problem and could not resolve your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) methodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not allowed for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {

	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}
