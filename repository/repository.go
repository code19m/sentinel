package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/code19m/sentinel/entity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("object not found")

type Store interface {
	Add(ctx context.Context, e entity.ErrorInfo) error
	Update(ctx context.Context, e entity.ErrorInfo) error
	FindLast(ctx context.Context, service, operation string, alerted bool) (entity.ErrorInfo, error)
}

func NewPgStore(pool *pgxpool.Pool) (*pgStore, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	store := &pgStore{pool: pool}
	err := store.initDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewPgStore: %w", err)
	}

	return store, nil
}

type pgStore struct {
	pool *pgxpool.Pool
}

func (r *pgStore) Add(ctx context.Context, e entity.ErrorInfo) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO errors (id, code, message, details, service, operation, created_at, alerted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`, e.ID, e.Code, e.Message, e.Details, e.Service, e.Operation, e.CreatedAt, e.Alerted)
	if err != nil {
		return fmt.Errorf("pgStore.Add: %w", err)
	}
	return nil
}

func (r *pgStore) Update(ctx context.Context, e entity.ErrorInfo) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE errors 
		SET alerted = $2
		WHERE id = $1;
	`, e.ID, e.Alerted)
	if err != nil {
		return fmt.Errorf("pgStore.Update: %w", err)
	}
	return nil
}

func (r *pgStore) FindLast(ctx context.Context, service, operation string, alerted bool) (entity.ErrorInfo, error) {
	e := entity.ErrorInfo{}

	row := r.pool.QueryRow(ctx, `
		SELECT id, code, message, details, service, operation, created_at, alerted
		FROM errors
		WHERE service = $1 AND operation = $2 AND alerted = $3
		ORDER BY created_at DESC
		LIMIT 1;
	`, service, operation, alerted)

	err := row.Scan(
		&e.ID, &e.Code, &e.Message, &e.Details, &e.Service, &e.Operation, &e.CreatedAt, &e.Alerted,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, fmt.Errorf("pgStore.FindLast: %w", err)
	}

	return e, nil
}

func (r *pgStore) initDB(ctx context.Context) error {
	// Create tables and indexes if not exists
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS errors (
			id UUID PRIMARY KEY,
			code TEXT NOT NULL,
			message TEXT NOT NULL,
			details JSONB NOT NULL,
			service TEXT NOT NULL,
			operation TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			alerted BOOLEAN NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_errors_service_operation_alerted
		ON errors (service, operation, alerted);

		CREATE INDEX IF NOT EXISTS idx_errors_created_at
		ON errors (created_at);
	`)
	if err != nil {
		return fmt.Errorf("pgStore.initDB: %w", err)
	}

	return nil
}
