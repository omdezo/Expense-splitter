package services

import (
	"context"

	"expense-splitter/types"
)

type Services struct {
	db     *types.DBPool
	logger *types.Logger
}

func New(db *types.DBPool, logger *types.Logger) *Services {
	return &Services{db: db, logger: logger}
}

func (s *Services) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}
