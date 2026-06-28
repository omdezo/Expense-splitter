// Package migration applies the SQL migrations under schema/ using goose.
package migration

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"expense-splitter/types"
)

//go:embed schema/*.sql
var embedMigrations embed.FS

// Run applies all pending migrations from schema/ to the database. goose works
// over database/sql, so the pgx pool is adapted with stdlib.OpenDBFromPool.
func Run(pool *types.DBPool) error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if err := goose.Up(db, "schema"); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
