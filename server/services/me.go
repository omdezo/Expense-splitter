package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"expense-splitter/types"
)

type MeResponse struct {
	Subject           string           `json:"subject"`
	Email             string           `json:"email"`
	EmailVerified     bool             `json:"email_verified"`
	Name              string           `json:"name"`
	PreferredUsername string           `json:"preferred_username"`
	LocalUser         *types.Principal `json:"local_user"`
}

func (s *Services) Me(ctx context.Context, id types.Identity) (*MeResponse, types.APIError) {
	resp := &MeResponse{
		Subject:           id.Subject,
		Email:             id.Email,
		EmailVerified:     id.EmailVerified,
		Name:              id.Name,
		PreferredUsername: id.PreferredUsername,
	}

	if id.Subject == "" {
		return resp, nil
	}

	p, err := s.principalByKeycloakID(ctx, id.Subject)
	switch {
	case err == nil:
		resp.LocalUser = p
	case errors.Is(err, pgx.ErrNoRows):

	default:
		s.logger.Errorw("me: resolve principal by subject", "error", err)
		return nil, types.NewServerError()
	}

	return resp, nil
}
