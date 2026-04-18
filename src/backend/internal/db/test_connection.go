package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// TestConnection opens a short-lived connection to the target database
// and runs SELECT 1 to verify connectivity.
func (db *DB) TestConnection(ctx context.Context, id uuid.UUID, decryptDSN func([]byte) (string, error)) error {
	c, err := db.GetConnection(ctx, id)
	if err != nil {
		return fmt.Errorf("get connection: %w", err)
	}
	if c == nil {
		return fmt.Errorf("connection not found")
	}

	dsn, err := decryptDSN(c.DSNEncrypted)
	if err != nil {
		return fmt.Errorf("decrypt dsn: %w", err)
	}

	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	target, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer target.Close()

	if err := target.PingContext(tctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	if _, err := target.ExecContext(tctx, "SELECT 1"); err != nil {
		return fmt.Errorf("select 1: %w", err)
	}

	return nil
}