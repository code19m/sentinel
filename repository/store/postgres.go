package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/code19m/sentinel/entity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (r *pgStore) CheckAndMarkAlerted(ctx context.Context, e entity.ErrorInfo, cooldownMinutes int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgStore.CheckAndMarkAlerted: %w", err)
	}
	defer tx.Rollback(ctx)

	// 🔒 ADVISORY LOCK - based on service+operation hash
	// This ensures only ONE transaction can check/mark for this service+operation at a time
	// even when no rows exist yet
	lockKey := hashServiceOperation(e.Service, e.Operation)
	_, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1);`, lockKey)
	if err != nil {
		return fmt.Errorf("pgStore.CheckAndMarkAlerted: failed to acquire lock: %w", err)
	}

	// Check cooldown
	var lastAlertedAt time.Time
	var lastAlertedID string

	row := tx.QueryRow(ctx, `
		SELECT id, created_at
		FROM errors
		WHERE service = $1 AND operation = $2 AND alerted = true
		ORDER BY created_at DESC
		LIMIT 1;
	`, e.Service, e.Operation)

	err = row.Scan(&lastAlertedID, &lastAlertedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("pgStore.CheckAndMarkAlerted: %w", err)
	}

	if err == nil {
		cooldownDuration := time.Duration(cooldownMinutes) * time.Minute
		if time.Since(lastAlertedAt) < cooldownDuration {
			// Still in cooldown period
			return ErrAlertCooldown
		}
	}

	// Cooldown passed, mark as alerted
	_, err = tx.Exec(ctx, `
		UPDATE errors 
		SET alerted = true
		WHERE id = $1;
	`, e.ID)
	if err != nil {
		return fmt.Errorf("pgStore.CheckAndMarkAlerted: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("pgStore.CheckAndMarkAlerted: %w", err)
	}

	return nil
}

func (r *pgStore) GetErrorFrequency(ctx context.Context, service, operation string, minutesBack int) (int, error) {
	var count int

	row := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM errors
		WHERE service = $1 AND operation = $2 AND created_at > NOW() - INTERVAL '1 minute' * $3;
	`, service, operation, minutesBack)

	err := row.Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("pgStore.GetErrorFrequency: %w", err)
	}

	return count, nil
}

// hashServiceOperation creates a consistent hash for service+operation
// to use as advisory lock key
func hashServiceOperation(service, operation string) int64 {
	// Simple hash function - combine service and operation into a stable int64
	combined := service + ":" + operation
	var hash int64
	for i, c := range combined {
		hash = hash*31 + int64(c) + int64(i)
	}
	// Ensure positive value
	if hash < 0 {
		hash = -hash
	}
	return hash
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
