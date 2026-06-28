package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

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

	const link = `
UPDATE users SET keycloak_id = $1::uuid, updated_at = now()
WHERE email = $2 AND keycloak_id IS NULL
RETURNING id, email, is_global_admin, verification_status`
	p := &types.Principal{}
	err = tx.QueryRow(ctx, link, id.Subject, id.Email).
		Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus)
	switch {
	case err == nil:
		if err := tx.Commit(ctx); err != nil {
			s.logger.Errorw("register: commit link", "error", err)
			return nil, types.NewServerError()
		}
		return p, nil
	case !errors.Is(err, pgx.ErrNoRows):
		s.logger.Errorw("register: link by email", "error", err)
		return nil, types.NewServerError()
	}

	const ins = `
INSERT INTO users (keycloak_id, email)
VALUES ($1::uuid, $2)
ON CONFLICT (email) DO NOTHING
RETURNING id, email, is_global_admin, verification_status`
	err = tx.QueryRow(ctx, ins, id.Subject, id.Email).
		Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus)
	switch {
	case err == nil:
		if err := tx.Commit(ctx); err != nil {
			s.logger.Errorw("register: commit insert", "error", err)
			return nil, types.NewServerError()
		}
		return p, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewConflictError("email already registered")
	default:
		s.logger.Errorw("register: insert", "error", err)
		return nil, types.NewServerError()
	}
}

func (s *Services) principalByKeycloakID(ctx context.Context, subject string) (*types.Principal, error) {
	const q = `SELECT id, email, is_global_admin, verification_status FROM users WHERE keycloak_id = $1::uuid`
	p := &types.Principal{}
	if err := s.db.QueryRow(ctx, q, subject).
		Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus); err != nil {
		return nil, err
	}
	return p, nil
}
