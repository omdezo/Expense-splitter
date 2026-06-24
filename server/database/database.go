package database

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"expense-splitter/types"
)

func New(url string) (*types.DBPool, error) {
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}
