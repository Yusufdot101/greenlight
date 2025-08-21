package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Yusufdot101/greenlight/internal/validator"

	"github.com/lib/pq"
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Year      int32     `json:"year,omitempty"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

// ValidateMovie validates the client input before operating on it
func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.Check(movie.Title != "", "title", "must be provided")
	v.Check(
		len(movie.Title) <= 500,
		"title",
		"must not be more than 500 bytes long",
	)

	v.Check(movie.Year != 0, "year", "must be provided")
	v.Check(movie.Year >= 1888, "year", "must be greater than 1888")
	v.Check(
		movie.Year <= int32(time.Now().Year()),
		"year",
		"must not be in the future",
	)

	v.Check(movie.Runtime != 0, "runtime", "must be provided")
	v.Check(movie.Runtime > 0, "runtime", "must be a positive integer")

	v.Check(movie.Genres != nil, "genres", "must be provided")
	v.Check(len(movie.Genres) >= 1, "genres", "must contain at least 1 genre")
	v.Check(
		len(movie.Genres) <= 5,
		"genres",
		"must not contain more than 5 genres",
	)

	v.Check(
		validator.Unique(movie.Genres),
		"genres",
		"must not contain duplicate values",
	)
}

// MovieModel is the model or layer where all movie related operations will occur
// like insert movie, get, update and delete movie
type MovieModel struct {
	DB *sql.DB
}

func (model MovieModel) Insert(movie *Movie) error {
	// we return id, created_at and version because those are the only ones
	// remaining to complete the movie details
	queryStatement := `
		INSERT INTO movies (title, year, runtime, genres)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version
	`

	// slice containing the values for the placeholder parameters from the
	// movie struct
	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
	}

	// create a context that will finish in 3 seconds so that the connection or
	// query is cancelled if its taking too long and free resources
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// incase the function finished is finishes before the context, we cancel
	// it when the function is exiting
	defer cancel()

	// scan the id, created_at and version into the movie
	return model.DB.QueryRowContext(ctx, queryStatement, args...).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Version,
	)
}
func (model MovieModel) GetOneMovie(id int64) (*Movie, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	queryStatement := `
		SELECT id, created_at, title, year, runtime, genres, version
		FROM movies
		WHERE id = $1
	`

	var movie Movie

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, queryStatement, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Year,
		&movie.Runtime,
		pq.Array(&movie.Genres),
		&movie.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound

		default:
			return nil, err
		}
	}
	return &movie, nil
}

func (model MovieModel) GetAll(title string, year int, runtime int, genres []string, filter Filters) ([]*Movie, Metadata, error) {
	queryStatement := fmt.Sprintf(`
	SELECT COUNT(*) OVER(), * FROM movies
	WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
	AND (year = $2 OR $2 = -1)
	AND (runtime = $3 OR $3 = -1)
	AND (genres @> $4 OR $4 = '{}')
	ORDER BY %s %s, id ASC
	LIMIT $5
	OFFSET $6
	`, filter.sortColumn(), filter.sortDirection())

	args := []any{
		title,
		year,
		runtime,
		pq.Array(genres),
		filter.limit(),
		filter.offset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := model.DB.QueryContext(ctx, queryStatement, args...)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrRecordNotFound
		default:
			return nil, Metadata{}, err
		}
	}

	defer rows.Close()

	totalRecords := 0
	movies := []*Movie{}
	for rows.Next() {
		var movie Movie
		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Year,
			&movie.Runtime,
			pq.Array(&movie.Genres),
			&movie.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}

		movies = append(movies, &movie)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filter.Page, filter.PageSize)
	return movies, metadata, nil
}

func (model MovieModel) Update(movie *Movie) error {
	queryStatement := `
		UPDATE movies
		SET title = $1, year = $2, runtime = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
	`

	args := []any{
		movie.Title,
		movie.Year,
		movie.Runtime,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, queryStatement, args...).Scan(&movie.ID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// we don't return ErrRecordNotFound as the record exists because
			// we called Get() with the id before calling this function, so it must
			// mean the version changed, which means someone else updated the movie
			// at the exact same time and the version in the database is not the one
			// we have
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (model MovieModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	queryStatement := `
		DELETE FROM movies
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := model.DB.ExecContext(ctx, queryStatement, id)
	if err != nil {
		return err
	}

	rowsEffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// DELETE doensn't return sql.ErrNoRows when there are no records, the affected
	// rows will be zero
	if rowsEffected != 0 {
		return ErrRecordNotFound
	}

	return nil
}
