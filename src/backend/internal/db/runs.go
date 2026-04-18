package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Run struct {
	ID            uuid.UUID
	UserSub       string
	UserEmail     string
	UserGroups    string
	UserRole      string
	ConnectionID  uuid.UUID
	QueryID       *uuid.UUID
	SQL           string
	Parameters    json.RawMessage
	ExportFormat  *string
	MaskedColumns json.RawMessage
	StartedAt     time.Time
	FinishedAt    *time.Time
	DurationMs    *int
	RowCount      *int
	BytesReturned *int64
	Status        string
	ErrorMessage  *string
	ClientIP      *string
	UserAgent     *string
}

type InsertRunParams struct {
	UserSub       string
	UserEmail     string
	UserGroups    string
	UserRole      string
	ConnectionID  uuid.UUID
	QueryID       *uuid.UUID
	SQL           string
	Parameters    json.RawMessage
	ExportFormat  *string
	MaskedColumns json.RawMessage
	StartedAt     time.Time
	ClientIP      *string
	UserAgent     *string
}

type FinishRunParams struct {
	ID            uuid.UUID
	FinishedAt    time.Time
	DurationMs    int
	RowCount      int
	BytesReturned int64
	Status        string
	ErrorMessage  *string
}

func (db *DB) InsertRun(ctx context.Context, p InsertRunParams) (*Run, error) {
	const q = `
		INSERT INTO runs
			(user_sub, user_email, user_groups, user_role, connection_id, query_id,
			 sql, parameters, export_format, masked_columns, started_at, status, client_ip, user_agent)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'running',$12,$13)
		RETURNING id, user_sub, user_email, user_groups, user_role, connection_id, query_id,
		          sql, parameters, export_format, masked_columns, started_at, finished_at,
		          duration_ms, row_count, bytes_returned, status, error_message, client_ip, user_agent`

	if p.Parameters == nil {
		p.Parameters = json.RawMessage("null")
	}
	if p.MaskedColumns == nil {
		p.MaskedColumns = json.RawMessage("[]")
	}

	row := db.pool.QueryRowContext(ctx, q,
		p.UserSub, p.UserEmail, p.UserGroups, p.UserRole,
		p.ConnectionID, p.QueryID, p.SQL, p.Parameters,
		p.ExportFormat, p.MaskedColumns, p.StartedAt,
		p.ClientIP, p.UserAgent,
	)
	return scanRun(row)
}

func (db *DB) FinishRun(ctx context.Context, p FinishRunParams) error {
	const q = `
		UPDATE runs
		SET finished_at=$1, duration_ms=$2, row_count=$3,
		    bytes_returned=$4, status=$5, error_message=$6
		WHERE id=$7`

	_, err := db.pool.ExecContext(ctx, q,
		p.FinishedAt, p.DurationMs, p.RowCount,
		p.BytesReturned, p.Status, p.ErrorMessage, p.ID,
	)
	return err
}

func (db *DB) GetRun(ctx context.Context, id uuid.UUID) (*Run, error) {
	const q = `
		SELECT id, user_sub, user_email, user_groups, user_role, connection_id, query_id,
		       sql, parameters, export_format, masked_columns, started_at, finished_at,
		       duration_ms, row_count, bytes_returned, status, error_message, client_ip, user_agent
		FROM runs WHERE id = $1`

	row := db.pool.QueryRowContext(ctx, q, id)
	run, err := scanRun(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return run, err
}

func scanRun(s scanner) (*Run, error) {
	var r Run
	err := s.Scan(
		&r.ID, &r.UserSub, &r.UserEmail, &r.UserGroups, &r.UserRole,
		&r.ConnectionID, &r.QueryID, &r.SQL, &r.Parameters,
		&r.ExportFormat, &r.MaskedColumns, &r.StartedAt, &r.FinishedAt,
		&r.DurationMs, &r.RowCount, &r.BytesReturned, &r.Status,
		&r.ErrorMessage, &r.ClientIP, &r.UserAgent,
	)
	if err != nil {
		return nil, fmt.Errorf("scan run: %w", err)
	}
	return &r, nil
}