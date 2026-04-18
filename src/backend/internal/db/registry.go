package db

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// registry holds a live *sql.DB per target connection.
type registry struct {
	mu    sync.RWMutex
	pools map[uuid.UUID]*sql.DB
}

func newRegistry() *registry {
	return &registry{pools: make(map[uuid.UUID]*sql.DB)}
}

// Get returns the pool for a connection, opening it if necessary.
func (r *registry) Get(ctx context.Context, c *Connection) (*sql.DB, error) {
	r.mu.RLock()
	p, ok := r.pools[c.ID]
	r.mu.RUnlock()
	if ok {
		return p, nil
	}
	return r.open(ctx, c)
}

func (r *registry) open(ctx context.Context, c *Connection) (*sql.DB, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Double-check after acquiring write lock.
	if p, ok := r.pools[c.ID]; ok {
		return p, nil
	}
	dsn := string(c.dsn) // set by DB.GetConnectionWithDSN
	p, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("registry: open %s: %w", c.ID, err)
	}
	p.SetMaxOpenConns(20)
	p.SetMaxIdleConns(5)
	if err := p.PingContext(ctx); err != nil {
		p.Close()
		return nil, fmt.Errorf("registry: ping %s: %w", c.ID, err)
	}
	r.pools[c.ID] = p
	return p, nil
}

// Invalidate closes and removes the pool for a connection (call on update/delete).
func (r *registry) Invalidate(id uuid.UUID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.pools[id]; ok {
		p.Close()
		delete(r.pools, id)
	}
}