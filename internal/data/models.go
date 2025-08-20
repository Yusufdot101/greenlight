package data

import (
	"database/sql"
	"errors"
)

var (
	ErrRecordNotFound = errors.New("record not found")
)

type Models struct {
	Movies MovieModel
}

// NewModels returns a Models struct containing the initialized models
func NewModels(db *sql.DB) Models {
	return Models{Movies: MovieModel{DB: db}}
}
