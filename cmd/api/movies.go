package main

import (
	"errors"
	"net/http"

	"github.com/Yusufdot101/greenlight/internal/data"
	"github.com/Yusufdot101/greenlight/internal/validator"
)

// showMovieHandler gets the details of a specific movie, by id, and returns it
// if found or error otherwise
func (app *application) showMovieHandler(
	w http.ResponseWriter, r *http.Request,
) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	movie, err := app.models.Movies.GetOneMovie(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"movie": movie}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// showMoviesHandler shows multiple movies
func (app *application) listMoviesHandler(
	w http.ResponseWriter, r *http.Request,
) {
	var input struct {
		Title   string
		Year    int
		Runtime int
		Genres  []string
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})
	input.Year = app.readInt(qs, "year", -1, v)
	input.Runtime = app.readInt(qs, "runtime", -1, v)

	input.Page = app.readInt(qs, "page", 1, v)
	input.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Sort = app.readString(qs, "sort", "id")

	// negative means descending order
	input.SortSafeList = []string{
		"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime",
	}

	if data.ValidateFilters(v, input.Filters); !v.Vaild() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	movies, metadata, err := app.models.Movies.GetAll(
		input.Title, input.Year, input.Runtime, input.Genres, input.Filters,
	)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(
		w, http.StatusOK, envelope{"movies": movies, "metadata": metadata}, nil,
	)

	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// deleteMovieHandler deletes a specific movie by id. if not found it will
// return error
func (app *application) deleteMovieHandler(
	w http.ResponseWriter, r *http.Request,
) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(
		w,
		http.StatusOK,
		envelope{"message": "movie successfully deleted"},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

}

// createMovieHandler creates a new movie based on the client requst and
// performs validation on it. returns error if validation fails
func (app *application) createMovieHandler(
	w http.ResponseWriter, r *http.Request,
) {
	var input struct {
		Title   string       `json:"title"`
		Year    int32        `json:"year"`
		Runtime data.Runtime `json:"runtime"`
		Genres  []string     `json:"genres"`
	}

	err := app.readJSON(w, r, &input)

	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}
	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Vaild() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Movies.Insert(movie)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusCreated, envelope{"movie": &movie}, nil)
}

// updateMovieHandler updates the details of a specific movie by id
func (app *application) updateMovieHandler(
	w http.ResponseWriter, r *http.Request,
) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// verify the movie exists
	movie, err := app.models.Movies.GetOneMovie(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
			return
		}
		app.serverErrorResponse(w, r, err)
		return
	}

	// the fields that could be in the requst body
	var input struct {
		Title   *string       `json:"title"`
		Year    *int32        `json:"year"`
		Runtime *data.Runtime `json:"runtime"`
		Genres  []string      `json:"genres"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres

	}

	v := validator.New()

	if data.ValidateMovie(v, movie); !v.Vaild() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editErrorResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	app.writeJSON(w, http.StatusCreated, envelope{"movie": &movie}, nil)
}
