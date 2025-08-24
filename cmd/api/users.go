package main

import (
	"errors"
	"net/http"

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
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
		default:
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	if data.ValidateUser(v, user); !v.Vaild() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.writeJSON(
		w, http.StatusCreated,
		envelope{
			"message": "user registered successfully",
			"user":    user,
		},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
