// Package db owns the app-database pool and migration runner.
package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/google/uuid"
)

type DB struct {
	pool *sql.DB
	Registry *registry
}

func Open(ctx context.Context, dsn string) (*DB, error) {
	pool, err := sql.Open("pgx", dsn)
		if err != nil {
		return nil, fmt.Errorf("db: open: %w", err)
	}
	if err := pool.PingContext(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: ping: %w", err)
	}
	if err := runMigrations(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return &DB{pool: pool, Registry: newRegistry()}, nil
}

//go:embed migrations/*.sql
var migrationsFS embed.FS

func (db *DB) Pool() *sql.DB {
	return db.pool
}

// Close closes the underlying pool.
func (db *DB) Close() error {
	return db.pool.Close()
}

func runMigrations(pool *sql.DB) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrations source: %w", err)
	}
	driver, err := pgx.WithInstance(pool, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("migrations driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}

// GetConnectionWithDSN returns the connection with its DSN decrypted.
func (db *DB) GetConnectionWithDSN(ctx context.Context, id uuid.UUID, decryptDSN func([]byte) (string, error)) (*Connection, error) {
	c, err := db.GetConnection(ctx, id)
	if err != nil || c == nil {
		return c, err
	}
	dsn, err := decryptDSN(c.DSNEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt dsn for %s: %w", id, err)
	}
	c.dsn = dsn
	return c, nil
}

func (db *DB) TargetPool(ctx context.Context, id uuid.UUID, decryptDSN func([]byte) (string, error)) (*sql.DB, error) {
	c, err := db.GetConnectionWithDSN(ctx, id, decryptDSN)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("connection %s not found", id)
	}
	return db.Registry.Get(ctx, c)
}