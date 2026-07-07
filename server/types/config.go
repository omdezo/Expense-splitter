package types

import "fmt"

type Config struct {
	Port             string
	Postgres         PostgresConfig
	Keycloak         KeycloakConfig
	Storage          StorageConfig
	GlobalAdminEmail string
}

// StorageConfig points at the S3-compatible object store for proof images.
type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

func (s StorageConfig) Enabled() bool {
	return s.Endpoint != "" && s.AccessKey != "" && s.SecretKey != "" && s.Bucket != ""
}

type KeycloakConfig struct {
	JWKSURL  string
	Issuers  []string
	Audience string

	// Server-side calls into Keycloak (login token exchange + admin user
	// provisioning). BaseURL is the address the server itself reaches Keycloak
	// at (internal in Docker), distinct from the issuer minted into tokens.
	BaseURL       string
	Realm         string
	ClientID      string
	AdminUser     string
	AdminPassword string
}

func (k KeycloakConfig) Enabled() bool {
	return k.JWKSURL != ""
}

// LoginEnabled reports whether the password-grant login proxy can run.
func (k KeycloakConfig) LoginEnabled() bool {
	return k.BaseURL != "" && k.Realm != "" && k.ClientID != ""
}

// AdminEnabled reports whether admin-API user provisioning (register) can run.
func (k KeycloakConfig) AdminEnabled() bool {
	return k.LoginEnabled() && k.AdminUser != "" && k.AdminPassword != ""
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
