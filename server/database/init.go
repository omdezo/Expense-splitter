package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// New opens a PostgreSQL connection pool, verifies it with a ping, and returns
// a DB handle.
func New(url string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return &DB{Pool: pool}, nil
}

// Close releases the underlying connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}
