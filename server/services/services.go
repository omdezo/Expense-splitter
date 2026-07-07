package services

import (
	"context"

	"expense-splitter/database/repo"
	"expense-splitter/keycloak"
	"expense-splitter/storage"
	"expense-splitter/types"
)

type Services struct {
	db     *types.DBPool
	q      *repo.Queries
	logger *types.Logger
	authz  *Authorizer
	kc     *keycloak.Client
	store  *storage.Client
}

func New(db *types.DBPool, logger *types.Logger, kc *keycloak.Client, store *storage.Client) *Services {
	q := repo.New(db)
	return &Services{db: db, q: q, logger: logger, authz: NewAuthorizer(q, logger), kc: kc, store: store}
}

func (s *Services) Health(ctx context.Context) error {
	return s.db.Ping(ctx)
}
