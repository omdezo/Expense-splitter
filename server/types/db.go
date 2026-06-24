package types

import "github.com/jackc/pgx/v5/pgxpool"

// DBPool is the shared PostgreSQL connection pool type used across the server.
type DBPool = pgxpool.Pool
