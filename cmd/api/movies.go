package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/validator"
)

func (app *application) createMovie(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title   string       `json:"title"`
		Runtime data.Runtime `json:"runtime"`
		Year    int32        `json:"year"`
		Genres  []string     `json:"genres"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	movie := &data.Movie{
		ID:        1,
		Title:     input.Title,
		Runtime:   input.Runtime,
		Year:      input.Year,
		Genres:    input.Genres,
		CreatedAt: time.Now(),
		Version:   1,
	}

	v := validator.NewValidator()
	if data.ValidateMovie(v, movie); !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	err = app.models.Movies.InsertMovie(movie)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	err = app.writeJSON(
		w, http.StatusCreated, envelope{
			"message": "movie created successfully",
			"movie":   movie,
		},
	)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) GetMovieByID(w http.ResponseWriter, r *http.Request) {
	id, err := app.readParamID(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie})
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) updateMovie(w http.ResponseWriter, r *http.Request) {
	id, err := app.readParamID(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	var input struct {
		Title   string       `json:"title"`
		Runtime data.Runtime `json:"runtime"`
		Year    int32        `json:"year"`
		Genres  []string     `json:"genres"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, err)
		return
	}

	if input.Title != "" {
		movie.Title = input.Title
	}
	if input.Runtime != 0 {
		movie.Runtime = input.Runtime
	}
	if input.Year != 0 {
		movie.Year = input.Year
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	v := validator.NewValidator()
	if data.ValidateMovie(v, movie); !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	err = app.models.Movies.UpdateMovie(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflic):
			app.editConflictResponse(w)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.writeJSON(
		w, http.StatusOK, envelope{
			"message": "movie updated successfully",
			"movie":   movie,
		},
	)
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) DeleteMovieByID(w http.ResponseWriter, r *http.Request) {
	id, err := app.readParamID(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Movies.DeleteByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecord):
			app.notFoundResponse(w, r)
		default:
			app.serverError(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie deleted successfully"})
	if err != nil {
		app.serverError(w, r, err)
	}
}

func (app *application) ListMovies(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title  string
		Year   int
		Genres []string
		data.Filter
	}

	qs := r.URL.Query()
	v := validator.NewValidator()

	input.Title = app.readString(qs, "title", "")
	input.Year = app.readInt(qs, "year", -1, v)
	input.Genres = app.readCSV(qs, "genres", []string{})
	input.Page = app.readInt(qs, "page", 1, v)
	input.PageSize = app.readInt(qs, "page_size", 100, v)
	input.Sort = app.readString(qs, "sort", "id")
	input.SafeSortList = []string{
		"id", "-id",
		"title", "-title",
		"runtime", "-runtime",
		"year", "-year",
		"genres", "-genres",
	}

	if data.ValidateFilters(v, &input.Filter); !v.IsValid() {
		app.failedValidationResponse(w, v.Errors)
		return
	}

	movies, metadata, err := app.models.Movies.ListMovies(input.Title, input.Year, input.Genres, input.Filter)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	app.writeJSON(
		w, http.StatusOK, envelope{
			"metadata": metadata, "movies": movies,
		},
	)
}
