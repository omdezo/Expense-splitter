package database

import "expense-splitter/types"

// DB is the application's database handle. It wraps the pgx connection pool.
// A sqlc-generated *repo.Queries will be added here once the first query exists.
type DB struct {
	Pool *types.DBPool
}
