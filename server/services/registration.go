package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
	"expense-splitter/types"
)

func (s *Services) Register(ctx context.Context, id types.Identity) (*types.Principal, types.APIError) {
	if id.Subject == "" || id.Email == "" {
		return nil, types.NewBadRequestError("token is missing subject or email")
	}

	switch p, err := s.principalByKeycloakID(ctx, id.Subject); {
	case err == nil:
		return p, nil
	case !errors.Is(err, pgx.ErrNoRows):
		s.logger.Errorw("register: lookup by subject", "error", err)
		return nil, types.NewServerError()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		s.logger.Errorw("register: begin tx", "error", err)
		return nil, types.NewServerError()
	}
	defer tx.Rollback(ctx)
	qtx := s.q.WithTx(tx)

	linked, err := qtx.LinkUserKeycloakID(ctx, repo.LinkUserKeycloakIDParams{KeycloakID: id.Subject, Email: id.Email})
	switch {
	case err == nil:
		if err := tx.Commit(ctx); err != nil {
			s.logger.Errorw("register: commit link", "error", err)
			return nil, types.NewServerError()
		}
		return principalFromRow(linked.ID, linked.Email, linked.IsGlobalAdmin, linked.VerificationStatus), nil
	case !errors.Is(err, pgx.ErrNoRows):
		s.logger.Errorw("register: link by email", "error", err)
		return nil, types.NewServerError()
	}

	created, err := qtx.CreateUser(ctx, repo.CreateUserParams{KeycloakID: id.Subject, Email: id.Email})
	switch {
	case err == nil:
		if err := tx.Commit(ctx); err != nil {
			s.logger.Errorw("register: commit insert", "error", err)
			return nil, types.NewServerError()
		}
		return principalFromRow(created.ID, created.Email, created.IsGlobalAdmin, created.VerificationStatus), nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewConflictError("email already registered")
	default:
		s.logger.Errorw("register: insert", "error", err)
		return nil, types.NewServerError()
	}
}

func (s *Services) principalByKeycloakID(ctx context.Context, subject string) (*types.Principal, error) {
	row, err := s.q.GetUserByKeycloakID(ctx, subject)
	if err != nil {
		return nil, err
	}
	return principalFromRow(row.ID, row.Email, row.IsGlobalAdmin, row.VerificationStatus), nil
}

func (s *Services) principalByUserID(ctx context.Context, userID string) (*types.Principal, error) {
	row, err := s.q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return principalFromRow(row.ID, row.Email, row.IsGlobalAdmin, row.VerificationStatus), nil
}

func principalFromRow(id, email string, isGlobalAdmin bool, status types.VerificationStatus) *types.Principal {
	return &types.Principal{UserID: id, Email: email, IsGlobalAdmin: isGlobalAdmin, VerificationStatus: status}
}
