package services

import (
	"context"

	"expense-splitter/database/repo"
	"expense-splitter/keycloak"
	"expense-splitter/types"
)

type Services struct {
	db     *types.DBPool
	q      *repo.Queries
	logger *types.Logger
	authz  *Authorizer
	kc     *keycloak.Client
}

func New(db *types.DBPool, logger *types.Logger, kc *keycloak.Client) *Services {
	q := repo.New(db)
	return &Services{db: db, q: q, logger: logger, authz: NewAuthorizer(q, logger), kc: kc}
}

func (s *Services) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}
