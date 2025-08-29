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

	router.HandlerFunc(
		http.MethodGet, "/v1/movies", app.requirePermission("movies:read", app.ListMovies),
	)

	router.HandlerFunc(
		http.MethodGet, "/v1/movies/:id", app.requirePermission("movies:read", app.GetMovieByID),
	)

	router.HandlerFunc(
		http.MethodPost, "/v1/movies", app.requirePermission("movies:write", app.createMovie),
	)

	router.HandlerFunc(
		http.MethodDelete, "/v1/movies/:id",
		app.requirePermission("movies:write", app.DeleteMovieByID),
	)

	router.HandlerFunc(
		http.MethodPatch, "/v1/movies/:id", app.requirePermission("movies:write", app.updateMovie),
	)

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)

	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	router.HandlerFunc(
		http.MethodPut, "/v1/tokens/authentication", app.createAuthenticationTokenHandler,
	)

	return app.recoverPanic(app.enableCORS(app.rateLimiter(app.authenticate(router))))
}
