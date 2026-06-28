package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

// MeResponse is the body of GET /me: the token claims plus the resolved local
// user (nil when the caller is authenticated but not yet provisioned).
type MeResponse struct {
	Subject           string           `json:"subject"`
	Email             string           `json:"email"`
	EmailVerified     bool             `json:"email_verified"`
	Name              string           `json:"name"`
	PreferredUsername string           `json:"preferred_username"`
	LocalUser         *types.Principal `json:"local_user"`
}

// Me builds the /me response for an authenticated identity, resolving the local
// user (Principal) by email. A missing row is not an error — the caller is
// authenticated but not yet provisioned, so LocalUser stays nil.
func (s *Services) Me(ctx context.Context, id types.Identity) (*MeResponse, types.APIError) {
	resp := &MeResponse{
		Subject:           id.Subject,
		Email:             id.Email,
		EmailVerified:     id.EmailVerified,
		Name:              id.Name,
		PreferredUsername: id.PreferredUsername,
	}

	if id.Email == "" {
		return resp, nil
	}

	const q = `SELECT id, email, is_global_admin, verification_status FROM users WHERE email = $1`
	p := &types.Principal{}
	err := s.db.QueryRow(ctx, q, id.Email).Scan(&p.UserID, &p.Email, &p.IsGlobalAdmin, &p.VerificationStatus)
	switch {
	case err == nil:
		resp.LocalUser = p
	case errors.Is(err, pgx.ErrNoRows):
		// authenticated but not provisioned — LocalUser stays nil
	default:
		s.logger.Errorw("resolve principal by email", "error", err)
		return nil, types.NewServerError()
	}

	return resp, nil
}
