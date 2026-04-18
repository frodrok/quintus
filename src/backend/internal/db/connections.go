package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Connection struct {
	ID                 uuid.UUID
	Name               string
	Driver             string
	DSNEncrypted       []byte
	ReadOnly           bool
	StatementTimeoutMs int
	CreatedAt          time.Time
	CreatedBySub       *string
	CreatedByEmail     *string
	dsn string
}

type CreateConnectionParams struct {
	Name               string
	Driver             string
	DSNEncrypted       []byte
	ReadOnly           bool
	StatementTimeoutMs int
	CreatedBySub       *string
	CreatedByEmail     *string
}

func (db *DB) CreateConnection(ctx context.Context, p CreateConnectionParams) (*Connection, error) {
	const q = `
		INSERT INTO connections
			(name, driver, dsn_encrypted, read_only, statement_timeout_ms, created_by_sub, created_by_email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, name, driver, dsn_encrypted, read_only, statement_timeout_ms,
		          created_at, created_by_sub, created_by_email`

	row := db.pool.QueryRowContext(ctx, q,
		p.Name, p.Driver, p.DSNEncrypted, p.ReadOnly, p.StatementTimeoutMs,
		p.CreatedBySub, p.CreatedByEmail,
	)
	return scanConnection(row)
}

func (db *DB) GetConnection(ctx context.Context, id uuid.UUID) (*Connection, error) {
	const q = `
		SELECT id, name, driver, dsn_encrypted, read_only, statement_timeout_ms,
		       created_at, created_by_sub, created_by_email
		FROM connections WHERE id = $1`

	row := db.pool.QueryRowContext(ctx, q, id)
	c, err := scanConnection(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

func (db *DB) ListConnections(ctx context.Context) ([]*Connection, error) {
	const q = `
		SELECT id, name, driver, dsn_encrypted, read_only, statement_timeout_ms,
		       created_at, created_by_sub, created_by_email
		FROM connections ORDER BY name`

	rows, err := db.pool.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list connections: %w", err)
	}
	defer rows.Close()

	var out []*Connection
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (db *DB) UpdateConnection(ctx context.Context, id uuid.UUID, p CreateConnectionParams) (*Connection, error) {
	const q = `
		UPDATE connections
		SET name=$1, driver=$2, dsn_encrypted=$3, read_only=$4, statement_timeout_ms=$5
		WHERE id=$6
		RETURNING id, name, driver, dsn_encrypted, read_only, statement_timeout_ms,
		          created_at, created_by_sub, created_by_email`

	row := db.pool.QueryRowContext(ctx, q,
		p.Name, p.Driver, p.DSNEncrypted, p.ReadOnly, p.StatementTimeoutMs, id,
	)
	c, err := scanConnection(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	db.Registry.Invalidate(id)
	return c, err
}

func (db *DB) DeleteConnection(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.ExecContext(ctx, `DELETE FROM connections WHERE id = $1`, id)
	if err == nil {
		db.Registry.Invalidate(id)
	}
	return err
}

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanConnection(s scanner) (*Connection, error) {
	var c Connection
	err := s.Scan(
		&c.ID, &c.Name, &c.Driver, &c.DSNEncrypted, &c.ReadOnly,
		&c.StatementTimeoutMs, &c.CreatedAt, &c.CreatedBySub, &c.CreatedByEmail,
	)
	if err != nil {
		return nil, fmt.Errorf("scan connection: %w", err)
	}
	return &c, nil
}