package middleware

import (
	"errors"
	"sync"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"

	"expense-splitter/types"
)

// Sentinel errors from Validate so callers can map them to HTTP statuses.
var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrProviderUnavailable = errors.New("authentication provider unavailable")
)

// JWTValidator validates Keycloak access tokens: signature (against the realm
// JWKS), issuer, audience and expiry. The JWKS is fetched lazily on first use
// and refreshed in the background, so the server can start before Keycloak is
// ready.
type JWTValidator struct {
	cfg     types.KeycloakConfig
	parser  *jwt.Parser
	issuers map[string]struct{}

	mu sync.Mutex
	kf keyfunc.Keyfunc
}

func NewJWTValidator(cfg types.KeycloakConfig) *JWTValidator {
	issuers := make(map[string]struct{}, len(cfg.Issuers))
	for _, iss := range cfg.Issuers {
		issuers[iss] = struct{}{}
	}
	return &JWTValidator{
		cfg: cfg,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{"RS256", "RS384", "RS512", "ES256", "ES384", "ES512"}),
		),
		issuers: issuers,
	}
}

// Validate verifies the raw bearer token and returns the authenticated identity.
// It returns ErrProviderUnavailable if the JWKS cannot be fetched, and
// ErrInvalidToken for any validation failure.
func (v *JWTValidator) Validate(raw string) (*types.Identity, error) {
	kf, err := v.keys()
	if err != nil {
		return nil, ErrProviderUnavailable
	}

	mc := jwt.MapClaims{}
	token, err := v.parser.ParseWithClaims(raw, mc, kf.Keyfunc)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if err := v.checkIssuerAudience(mc); err != nil {
		return nil, err
	}
	return identityFromClaims(mc), nil
}

// keys returns the JWKS keyfunc, building it on first call. A build failure
// means Keycloak is unreachable; the next call retries.
func (v *JWTValidator) keys() (keyfunc.Keyfunc, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.kf != nil {
		return v.kf, nil
	}
	kf, err := keyfunc.NewDefault([]string{v.cfg.JWKSURL})
	if err != nil {
		return nil, err
	}
	v.kf = kf
	return kf, nil
}

// checkIssuerAudience enforces issuer and audience beyond the signature/expiry
// checks the parser already performs.
func (v *JWTValidator) checkIssuerAudience(mc jwt.MapClaims) error {
	if len(v.issuers) > 0 {
		iss, _ := mc["iss"].(string)
		if _, ok := v.issuers[iss]; !ok {
			return ErrInvalidToken
		}
	}
	if v.cfg.Audience != "" && !audienceContains(mc["aud"], v.cfg.Audience) {
		return ErrInvalidToken
	}
	return nil
}

func identityFromClaims(mc jwt.MapClaims) *types.Identity {
	str := func(key string) string {
		s, _ := mc[key].(string)
		return s
	}
	verified, _ := mc["email_verified"].(bool)
	return &types.Identity{
		Subject:           str("sub"),
		Email:             str("email"),
		EmailVerified:     verified,
		Name:              str("name"),
		PreferredUsername: str("preferred_username"),
	}
}

// audienceContains handles the `aud` claim being either a single string or an
// array of strings, both of which Keycloak emits depending on configuration.
func audienceContains(aud any, want string) bool {
	switch v := aud.(type) {
	case string:
		return v == want
	case []string:
		for _, a := range v {
			if a == want {
				return true
			}
		}
	case []any:
		for _, a := range v {
			if s, ok := a.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}
