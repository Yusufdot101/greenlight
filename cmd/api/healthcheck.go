package main

import (
	"net/http"
)

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	env := envelope{
		"status": "available",
		"system info": map[string]string{
			"environment": app.config.env, "version": version,
		},
	}

	err := app.writeJSON(w, http.StatusOK, env)
	if err != nil {
		app.serverError(w, r, err)
	}
}
