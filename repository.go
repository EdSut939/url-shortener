package main

import (
	"context"
	"database/sql"
)

type UrlRepository interface {
	Insert(ctx context.Context, shortCode, originalUrl string, ttl *int64) error
	GetByShortCode(ctx context.Context, shortCode string) (UrlRecord, error)
	IncrementVisits(ctx context.Context, id int64) error
}

type PostgresUrlRepository struct {
	db *sql.DB
}

func NewPostgresUrlRepository(db *sql.DB) *PostgresUrlRepository {
	return &PostgresUrlRepository{db: db}
}

func (repo *PostgresUrlRepository) Insert(ctx context.Context, shortCode, originalUrl string, ttl *int64) error {
	_, err := repo.db.ExecContext(
		ctx,
		"INSERT INTO urls (short_code, original_url, ttl) VALUES ($1, $2, $3)",
		shortCode, originalUrl, ttl,
	)
	return err
}

func (repo *PostgresUrlRepository) GetByShortCode(ctx context.Context, shortCode string) (UrlRecord, error) {
	var r UrlRecord
	err := repo.db.QueryRowContext(
		ctx,
		"SELECT id, original_url, ttl, created_at, visits FROM urls WHERE short_code = $1",
		shortCode,
	).Scan(&r.Id, &r.OriginalUrl, &r.Ttl, &r.CreatedAt, &r.Visits)
	return r, err
}

func (repo *PostgresUrlRepository) IncrementVisits(ctx context.Context, id int64) error {
	_, err := repo.db.ExecContext(ctx, "UPDATE urls SET visits = visits + 1 WHERE id = $1", id)
	return err
}
