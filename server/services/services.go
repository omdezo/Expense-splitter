package services

import (
	"context"

	"expense-splitter/types"
)

type Services struct {
	db     *types.DBPool
	logger *types.Logger
	authz  Authorizer
}

func New(db *types.DBPool, logger *types.Logger) *Services {
	return &Services{db: db, logger: logger, authz: NewAuthorizer(db, logger)}
}

func (s *Services) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}
