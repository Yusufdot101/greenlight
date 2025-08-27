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

var (
	ErrNoRecord    = errors.New("no record")
	ErrEditConflic = errors.New("edit conflict")
)

type Movie struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"-"`
	Title     string    `json:"title"`
	Runtime   Runtime   `json:"runtime,omitempty"`
	Year      int32     `json:"year,omitempty"`
	Genres    []string  `json:"genres,omitempty"`
	Version   int32     `json:"version"`
}

func ValidateMovie(v *validator.Validator, movie *Movie) {
	v.CheckAdd(movie.Title != "", "title", "must be provided")
	v.CheckAdd(len(movie.Title) <= 500, "title", "cannot more than 500 characters")

	v.CheckAdd(movie.Runtime != 0, "runtime", "must be provided")
	v.CheckAdd(movie.Runtime >= 0, "runtime", "must be postive integer")

	v.CheckAdd(movie.Year != 0, "year", "must be provided")
	v.CheckAdd(movie.Year >= 1888, "year", "must be at least 1888")
	v.CheckAdd(movie.Year <= int32(time.Now().Year()), "year", "must be at least 1888")

	v.CheckAdd(len(movie.Genres) >= 1, "genres", "must at have at least one")
	v.CheckAdd(len(movie.Genres) <= 5, "genres", "cannot have more than five")
	v.CheckAdd(validator.ListUnique(movie.Genres...), "genres", "cannot have duplicates")
}

type MovieModel struct {
	DB *sql.DB
}

func (model *MovieModel) InsertMovie(movie *Movie) error {
	query := `
	INSERT INTO movies (title, runtime, year, genres)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at, version
	`
	args := []any{
		movie.Title,
		movie.Runtime,
		movie.Year,
		pq.Array(movie.Genres),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, query, args...).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Version,
	)
	if err != nil {
		return err
	}

	return nil
}

func (model *MovieModel) GetByID(id int64) (*Movie, error) {
	query := `
		SELECT id, created_at, title, runtime, year, genres, version
		FROM movies
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var movie Movie
	err := model.DB.QueryRowContext(ctx, query, id).Scan(
		&movie.ID,
		&movie.CreatedAt,
		&movie.Title,
		&movie.Runtime,
		&movie.Year,
		pq.Array(&movie.Genres),
		&movie.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecord
		default:
			return nil, err
		}
	}

	return &movie, nil
}

func (model *MovieModel) DeleteByID(id int64) error {
	query := `
		DELETE FROM movies
		WHERE id = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	res, err := model.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNoRecord
	}

	return nil
}

func (model *MovieModel) UpdateMovie(movie *Movie) error {
	query := `
		UPDATE movies
		SET title = $1, runtime = $2, year = $3, genres = $4, version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version
	`
	args := []any{
		movie.Title,
		movie.Runtime,
		movie.Year,
		pq.Array(movie.Genres),
		movie.ID,
		movie.Version,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := model.DB.QueryRowContext(ctx, query, args...).Scan(&movie.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflic
		default:
			return err
		}
	}

	return nil
}

func (model *MovieModel) ListMovies(
	title string, year int, genres []string, filter Filter,
) ([]*Movie, *Metadata, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(), id, created_at, title, runtime, year, genres, version FROM movies
		WHERE (to_tsvector('simple', title) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (year = $2 OR $2 = -1)
		AND (genres @> $3 OR $3 = '{}')
		ORDER BY %s %s, id ASC
		LIMIT $4
		OFFSET $5
	`, filter.SortColumn(), filter.SortDirection())
	args := []any{
		title,
		year,
		pq.Array(genres),
		filter.Limit(),
		filter.Offset(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := model.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var movies []*Movie
	totalRecords := 0
	for rows.Next() {
		var movie Movie
		err := rows.Scan(
			&totalRecords,
			&movie.ID,
			&movie.CreatedAt,
			&movie.Title,
			&movie.Runtime,
			&movie.Year,
			pq.Array(&movie.Genres),
			&movie.Version,
		)
		if err != nil {
			return nil, nil, err
		}

		movies = append(movies, &movie)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, err
	}

	metadata := NewMetadata(filter.Page, filter.PageSize, totalRecords)

	return movies, metadata, nil
}
