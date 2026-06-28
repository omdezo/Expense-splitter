package types

import "fmt"

type Config struct {
	Port             string
	Postgres         PostgresConfig
	Keycloak         KeycloakConfig
	GlobalAdminEmail string
}

type KeycloakConfig struct {
	JWKSURL  string
	Issuers  []string
	Audience string
}

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
