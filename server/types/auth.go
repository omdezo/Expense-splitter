package types

import "strings"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (r *RegisterRequest) Validate() APIError {
	email := strings.TrimSpace(r.Email)
	if email == "" || !strings.Contains(email, "@") {
		return NewBadRequestError("a valid email is required")
	}
	if len(r.Password) < 8 {
		return NewBadRequestError("password must be at least 8 characters")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() APIError {
	if strings.TrimSpace(r.Email) == "" || r.Password == "" {
		return NewBadRequestError("email and password are required")
	}
	return nil
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshRequest) Validate() APIError {
	if strings.TrimSpace(r.RefreshToken) == "" {
		return NewBadRequestError("refresh_token is required")
	}
	return nil
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *LogoutRequest) Validate() APIError {
	if strings.TrimSpace(r.RefreshToken) == "" {
		return NewBadRequestError("refresh_token is required")
	}
	return nil
}

type TokenResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresIn    int        `json:"expires_in"`
	TokenType    string     `json:"token_type"`
	User         *Principal `json:"user,omitempty"`
}
