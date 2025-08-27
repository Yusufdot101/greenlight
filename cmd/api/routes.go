package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowedResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheck)

	router.HandlerFunc(http.MethodGet, "/v1/movies", app.ListMovies)

	router.HandlerFunc(http.MethodPost, "/v1/movies", app.createMovie)

	router.HandlerFunc(http.MethodGet, "/v1/movies/:id", app.GetMovieByID)

	router.HandlerFunc(http.MethodDelete, "/v1/movies/:id", app.DeleteMovieByID)

	router.HandlerFunc(http.MethodPatch, "/v1/movies/:id", app.updateMovie)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	return app.recoverPanic(app.rateLimiter(router))
}
