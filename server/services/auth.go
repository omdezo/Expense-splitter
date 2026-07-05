package services

import (
	"context"
	"errors"

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

// Login exchanges credentials for a Keycloak token via the password grant.
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

	return &types.TokenResponse{
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		ExpiresIn:    tok.ExpiresIn,
		TokenType:    tok.TokenType,
	}, nil
}
