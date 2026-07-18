package securityaudit

import (
	"database/sql"
	"errors"
)

var ErrEventNotFound = errors.New("prompt audit event not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}
