package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/database/repo"
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

	row, err := s.q.SetUserVerificationPending(ctx, p.UserID)
	if err != nil {
		s.logger.Errorw("submit verification: update", "error", err)
		return nil, types.NewServerError()
	}
	return principalFromRow(row.ID, row.Email, row.IsGlobalAdmin, row.VerificationStatus), nil
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
	if apiErr := s.authz.RequireGlobalAdmin(admin); apiErr != nil {
		return nil, apiErr
	}

	row, err := s.q.SetUserVerification(ctx, repo.SetUserVerificationParams{
		Status:     decision,
		VerifiedBy: admin.UserID,
		ID:         targetUserID,
	})
	switch {
	case err == nil:
		return principalFromRow(row.ID, row.Email, row.IsGlobalAdmin, row.VerificationStatus), nil
	case errors.Is(err, pgx.ErrNoRows):
		return nil, types.NewNotFoundError("user not found")
	default:
		s.logger.Errorw("set verification: update", "error", err)
		return nil, types.NewServerError()
	}
}
