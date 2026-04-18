package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Query struct {
	ID           uuid.UUID
	Name         string
	Description  *string
	ConnectionID uuid.UUID
	SQL          string
	Parameters   json.RawMessage
	ColumnMasks  json.RawMessage
	RowMask				json.RawMessage
	OwnerSub     string
	OwnerEmail   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type CreateQueryParams struct {
	Name         string
	Description  *string
	ConnectionID uuid.UUID
	SQL          string
	Parameters   json.RawMessage
	ColumnMasks  json.RawMessage
	RowMask      json.RawMessage
	OwnerSub     string
	OwnerEmail   string
}

func coalesceJSON(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}

func (db *DB) CreateQuery(ctx context.Context, p CreateQueryParams) (*Query, error) {
const q = `
    INSERT INTO queries
        (name, description, connection_id, sql, parameters, column_masks, row_mask, owner_sub, owner_email)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    RETURNING id, name, description, connection_id, sql, parameters, column_masks, row_mask,
              owner_sub, owner_email, created_at, updated_at`

row := db.pool.QueryRowContext(ctx, q,
    p.Name, p.Description, p.ConnectionID, p.SQL,
    coalesceJSON(p.Parameters, "[]"),
		coalesceJSON(p.ColumnMasks, "[]"), coalesceJSON(p.RowMask, "null"), p.OwnerSub, p.OwnerEmail,
)
	return scanQuery(row)
}

func (db *DB) GetQuery(ctx context.Context, id uuid.UUID) (*Query, error) {
	const q = `
		SELECT id, name, description, connection_id, sql, parameters, column_masks,
					row_mask,
		       owner_sub, owner_email, created_at, updated_at
		FROM queries WHERE id = $1`

	row := db.pool.QueryRowContext(ctx, q, id)
	qry, err := scanQuery(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return qry, err
}

func (db *DB) ListQueries(ctx context.Context) ([]*Query, error) {
	const q = `
		SELECT id, name, description, connection_id, sql, parameters, column_masks,
		row_mask,
		       owner_sub, owner_email, created_at, updated_at
		FROM queries ORDER BY name`

	rows, err := db.pool.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list queries: %w", err)
	}
	defer rows.Close()

	var out []*Query
	for rows.Next() {
		qry, err := scanQuery(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, qry)
	}
	return out, rows.Err()
}

func (db *DB) UpdateQuery(ctx context.Context, id uuid.UUID, p CreateQueryParams) (*Query, error) {
	const q = `
		UPDATE queries
		SET name=$1, description=$2, connection_id=$3, sql=$4,
		    parameters=$5, column_masks=$6, row_mask=$7, updated_at=now()
		WHERE id=$8
		RETURNING id, name, description, connection_id, sql, parameters, column_masks,
							row_mask,
		          owner_sub, owner_email, created_at, updated_at`

	row := db.pool.QueryRowContext(ctx, q,
    p.Name, p.Description, p.ConnectionID, p.SQL,
    coalesceJSON(p.Parameters, "[]"),
    coalesceJSON(p.ColumnMasks, "[]"),
    coalesceJSON(p.RowMask, "null"),
    id,
)
	qry, err := scanQuery(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return qry, err
}

func (db *DB) DeleteQuery(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.ExecContext(ctx, `DELETE FROM queries WHERE id = $1`, id)
	return err
}

func scanQuery(s scanner) (*Query, error) {
	var q Query
	err := s.Scan(
		&q.ID, &q.Name, &q.Description, &q.ConnectionID, &q.SQL,
		&q.Parameters, &q.ColumnMasks, &q.RowMask, &q.OwnerSub, &q.OwnerEmail,
		&q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan query: %w", err)
	}
	return &q, nil
}