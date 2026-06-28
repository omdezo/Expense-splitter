package services

import (
	"context"

	"expense-splitter/types"
)

// Services is the business-logic layer between the HTTP handlers and the
// database. Handlers call its methods; the methods own queries, authorization
// checks and audit writes, and return either a result or a types.APIError.
type Services struct {
	db     *types.DBPool
	logger *types.Logger
}

func New(db *types.DBPool, logger *types.Logger) *Services {
	return &Services{db: db, logger: logger}
}

// Health pings the database. A nil return means healthy.
func (s *Services) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}
