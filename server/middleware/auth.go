package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"expense-splitter/types"
)

const identityContextKey = "auth.identity"

// Auth provides Echo authentication middleware backed by a JWTValidator. It only
// establishes *who* the caller is; authorization (what they may do) lives in
// permissions.go and the service layer.
type Auth struct {
	cfg types.KeycloakConfig
	jwt *JWTValidator
}

func NewAuth(cfg types.KeycloakConfig) *Auth {
	return &Auth{cfg: cfg, jwt: NewJWTValidator(cfg)}
}

// Require rejects any request without a valid Keycloak bearer token. On success
// it stores the authenticated identity in the context (read it with GetIdentity)
// and calls the next handler.
func (a *Auth) Require() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !a.cfg.Enabled() {
				return c.JSON(http.StatusServiceUnavailable, types.NewServiceUnavailableError("authentication is not configured"))
			}

			raw, err := bearerToken(c.Request())
			if err != nil {
				return c.JSON(http.StatusUnauthorized, types.NewUnauthorizedError(err.Error()))
			}

			identity, err := a.jwt.Validate(raw)
			if err != nil {
				if errors.Is(err, ErrProviderUnavailable) {
					c.Logger().Errorf("auth: %v", err)
					return c.JSON(http.StatusServiceUnavailable, types.NewServiceUnavailableError("authentication provider unavailable"))
				}
				return c.JSON(http.StatusUnauthorized, types.NewUnauthorizedError("invalid token"))
			}

			c.Set(identityContextKey, identity)
			return next(c)
		}
	}
}

// GetIdentity returns the authenticated identity, or nil if the request did not
// pass Require.
func GetIdentity(c echo.Context) *types.Identity {
	if v, ok := c.Get(identityContextKey).(*types.Identity); ok {
		return v
	}
	return nil
}

func bearerToken(r *http.Request) (string, error) {
	h := r.Header.Get("Authorization")
	if h == "" {
		return "", errors.New("missing Authorization header")
	}
	const prefix = "Bearer "
	if len(h) <= len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", errors.New("Authorization header must be a Bearer token")
	}
	token := strings.TrimSpace(h[len(prefix):])
	if token == "" {
		return "", errors.New("empty bearer token")
	}
	return token, nil
}
