package repo

import (
	"context"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func New(dsn string) (*Repository, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	r := &Repository{db: db}
	if err := r.initSchema(context.Background()); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}
