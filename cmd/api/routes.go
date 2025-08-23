package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	// show application information
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	// show specific movie
	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.showMovieHandler)
	// show specific movie
	router.HandlerFunc(http.MethodGet, "/v1/movies", app.listMoviesHandler)

	// create a new movie
	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovieHandler)
	// update the details of a specific movie
	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.updateMovieHandler)
	// delete a specific movie
	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.deleteMovieHandler)

	// wrap the router with the panic recovery middleware
	return app.recoverPanic(app.rateLimit(router))
}
