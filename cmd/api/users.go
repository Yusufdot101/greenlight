package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/validator"
)

func (app *application) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}
	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	v := validator.NewValidator()
	if data.ValidateUser(v, user); !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	err = app.models.Users.InsertUser(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with email address already exists")
			app.failedValidationResponse(w, v.Errors)

		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	token, err := app.models.Tokens.NewToken(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	data := map[string]any{
		"useID":           user.ID,
		"activationToken": token.Plaintext,
	}

	fn := func() {
		err := app.mailer.Send(user.Email, "user_welcome.tmpl.html", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	}
	app.background(fn)

	err = app.writeJSON(
		w,
		http.StatusAccepted,
		envelope{
			"message": "user created successfully",
			"user":    user,
		},
	)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Token string `json:"token"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	v := validator.NewValidator()
	if data.ValidateTokenPlaintext(v, input.Token); !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	user, err := app.models.Users.GetUserForToken(data.ScopeActivation, input.Token)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			v.AddError("token", "invaild or expired token")
			app.failedValidationResponse(w, v.Errors)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	user.Activated = true
	err = app.models.Users.UpadeteUser(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflic):
			app.editConflictResponse(w)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.models.Tokens.DeleteAllForUser(user.ID, data.ScopeActivation)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.writeJSON(
		w,
		http.StatusAccepted,
		envelope{
			"message": "user activated successfully",
			"user":    user,
		},
	)
	if err != nil {
		app.serverError(w, r, err)
	}
}
