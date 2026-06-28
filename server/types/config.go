package types

import "fmt"

type Config struct {
	Port     string
	Postgres PostgresConfig
	Keycloak KeycloakConfig
}

// KeycloakConfig holds what the auth middleware needs to validate Keycloak
// access tokens. JWKSURL (where the signing keys are fetched) is kept separate
// from Issuers (the `iss` values we trust) on purpose: in Docker the server
// reaches Keycloak at an internal address while tokens are minted from the
// host-published one, so the two rarely match.
type KeycloakConfig struct {
	JWKSURL  string
	Issuers  []string
	Audience string
}

// Enabled reports whether auth is configured. When false the auth middleware
// rejects every request, since there is no way to validate a token.
func (k KeycloakConfig) Enabled() bool {
	return k.JWKSURL != ""
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Name, p.SSLMode)
}
