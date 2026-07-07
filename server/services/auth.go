package services

import (
	"context"
	"errors"

	"github.com/golang-jwt/jwt/v5"

	"expense-splitter/keycloak"
	"expense-splitter/types"
)

// SignUp creates the account in Keycloak (the identity provider) and then
// provisions the matching local users row, linked by the Keycloak subject.
func (s *Services) SignUp(ctx context.Context, req types.RegisterRequest) (*types.Principal, types.APIError) {
	subject, err := s.kc.CreateUser(ctx, req.Email, req.Password, req.Name)
	switch {
	case errors.Is(err, keycloak.ErrUserExists):
		return nil, types.NewConflictError("email already registered")
	case errors.Is(err, keycloak.ErrNotConfigured):
		return nil, types.NewServiceUnavailableError("registration is not configured")
	case errors.Is(err, keycloak.ErrUnavailable):
		return nil, types.NewServiceUnavailableError("authentication provider unavailable")
	case err != nil:
		s.logger.Errorw("signup: create keycloak user", "error", err)
		return nil, types.NewServerError()
	}

	return s.Register(ctx, types.Identity{Subject: subject, Email: req.Email, Name: req.Name})
}

// Login exchanges credentials for a Keycloak token via the password grant,
// then provisions-or-links the local account so a successful login NEVER
// leaves the caller in the "account not registered" limbo (Keycloak-side
// users like the seeded admin get their local row linked right here).
func (s *Services) Login(ctx context.Context, req types.LoginRequest) (*types.TokenResponse, types.APIError) {
	tok, err := s.kc.Login(ctx, req.Email, req.Password)
	switch {
	case errors.Is(err, keycloak.ErrInvalidCredentials):
		return nil, types.NewUnauthorizedError("invalid email or password")
	case errors.Is(err, keycloak.ErrNotConfigured):
		return nil, types.NewServiceUnavailableError("login is not configured")
	case errors.Is(err, keycloak.ErrUnavailable):
		return nil, types.NewServiceUnavailableError("authentication provider unavailable")
	case err != nil:
		s.logger.Errorw("login: keycloak token", "error", err)
		return nil, types.NewServerError()
	}

	resp := &types.TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
		TokenType:    tok.TokenType,
	}

	// The token came straight from Keycloak this instant, so its claims are
	// trusted without re-verifying the signature here.
	if identity := identityFromAccessToken(tok.AccessToken); identity != nil {
		p, apiErr := s.Register(ctx, *identity)
		if apiErr != nil {
			s.logger.Warnw("login: auto-provision failed", "email", identity.Email, "error", apiErr.Message)
		} else {
			resp.User = p
		}
	}
	return resp, nil
}

// Refresh trades a refresh token for a fresh pair — clients renew sessions
// without re-sending the password.
func (s *Services) Refresh(ctx context.Context, req types.RefreshRequest) (*types.TokenResponse, types.APIError) {
	tok, err := s.kc.Refresh(ctx, req.RefreshToken)
	switch {
	case errors.Is(err, keycloak.ErrInvalidCredentials):
		return nil, types.NewUnauthorizedError("refresh token is invalid or expired")
	case errors.Is(err, keycloak.ErrNotConfigured):
		return nil, types.NewServiceUnavailableError("login is not configured")
	case errors.Is(err, keycloak.ErrUnavailable):
		return nil, types.NewServiceUnavailableError("authentication provider unavailable")
	case err != nil:
		s.logger.Errorw("refresh: keycloak token", "error", err)
		return nil, types.NewServerError()
	}
	return &types.TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
		TokenType:    tok.TokenType,
	}, nil
}

// Logout revokes the Keycloak session behind the refresh token (idempotent).
func (s *Services) Logout(ctx context.Context, req types.LogoutRequest) types.APIError {
	err := s.kc.Logout(ctx, req.RefreshToken)
	switch {
	case errors.Is(err, keycloak.ErrNotConfigured):
		return types.NewServiceUnavailableError("login is not configured")
	case errors.Is(err, keycloak.ErrUnavailable):
		return types.NewServiceUnavailableError("authentication provider unavailable")
	case err != nil:
		s.logger.Errorw("logout: keycloak", "error", err)
		return types.NewServerError()
	}
	return nil
}

// identityFromAccessToken extracts the claims without signature verification —
// only for tokens Keycloak just handed us over the direct server-to-server
// call, never for inbound request tokens (the middleware verifies those).
func identityFromAccessToken(raw string) *types.Identity {
	claims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(raw, claims); err != nil {
		return nil
	}
	str := func(key string) string {
		s, _ := claims[key].(string)
		return s
	}
	if str("sub") == "" || str("email") == "" {
		return nil
	}
	return &types.Identity{
		Subject:           str("sub"),
		Email:             str("email"),
		Name:              str("name"),
		PreferredUsername: str("preferred_username"),
	}
}
