package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/validator"
)

func (app *application) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	v := validator.NewValidator()
	data.ValidateEmail(v, input.Email)
	data.ValidateEmail(v, input.Email)
	if !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	user, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.invalidCredentialsResponse(w)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	matches, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if !matches {
		app.invalidCredentialsResponse(w)
		return
	}

	token, err := app.models.Tokens.NewToken(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"authentication_token": token})
	if err != nil {
		app.serverError(w, r, err)
	}
}
