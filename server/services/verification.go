package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

func (s *Services) SubmitVerification(ctx context.Context, id types.Identity) (*types.Principal, types.APIError) {
	p, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("account not registered")
	case err != nil:
		s.logger.Errorw("submit verification: resolve principal", "error", err)
		return nil, types.NewServerError()
	}

	switch p.VerificationStatus {
	case types.VerificationPending:
		return p, nil
	case types.VerificationVerified:
		return nil, types.NewConflictError("account already verified")
	}

	const q = `
UPDATE users SET verification_status = 'pending_verification', updated_at = now()
WHERE id = $1::uuid
RETURNING id, email, is_global_admin, verification_status`
	updated := &types.Principal{}
	if err := s.db.QueryRow(ctx, q, p.UserID).
		Scan(&updated.UserID, &updated.Email, &updated.IsGlobalAdmin, &updated.VerificationStatus); err != nil {
		s.logger.Errorw("submit verification: update", "error", err)
		return nil, types.NewServerError()
	}
	return updated, nil
}

func (s *Services) SetVerification(ctx context.Context, actor types.Identity, targetUserID string, decision types.VerificationStatus) (*types.Principal, types.APIError) {
	if decision != types.VerificationVerified && decision != types.VerificationRejected {
		return nil, types.NewBadRequestError("decision must be verified or rejected")
	}

	admin, err := s.principalByKeycloakID(ctx, actor.Subject)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewForbiddenError("caller is not provisioned")
	case err != nil:
		s.logger.Errorw("set verification: resolve actor", "error", err)
		return nil, types.NewServerError()
	}
	if !admin.IsGlobalAdmin {
		return nil, types.NewForbiddenError("global admin only")
	}

	const q = `
UPDATE users SET verification_status = $1::verification_status, verified_by = $2::uuid, updated_at = now()
WHERE id = $3::uuid
RETURNING id, email, is_global_admin, verification_status`
	p := &types.Principal{}
	err = s.db.QueryRow(ctx, q, decision, admin.UserID, targetUserID).
		Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus)
	switch {
	case err == nil:
		return p, nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("user not found")
	default:
		s.logger.Errorw("set verification: update", "error", err)
		return nil, types.NewServerError()
	}
}
