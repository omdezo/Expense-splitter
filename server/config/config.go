package config

import (
	"log"
	"os"
	"strings"

	"expense-splitter/types"
)

func Load() *types.Config {
	cfg := &types.Config{
		Port: os.Getenv("SERVER_PORT"),
		Postgres: types.PostgresConfig{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     os.Getenv("DB_NAME"),
			SSLMode:  os.Getenv("DB_SSLMODE"),
		},
		Keycloak: types.KeycloakConfig{
			JWKSURL:       os.Getenv("KEYCLOAK_JWKS_URL"),
			Issuers:       splitAndTrim(os.Getenv("KEYCLOAK_ISSUERS")),
			Audience:      os.Getenv("KEYCLOAK_AUDIENCE"),
			BaseURL:       os.Getenv("KEYCLOAK_BASE_URL"),
			Realm:         os.Getenv("KEYCLOAK_REALM"),
			ClientID:      os.Getenv("KEYCLOAK_CLIENT_ID"),
			AdminUser:     os.Getenv("KEYCLOAK_ADMIN_USER"),
			AdminPassword: os.Getenv("KEYCLOAK_ADMIN_PASSWORD"),
		},
		GlobalAdminEmail: os.Getenv("GLOBAL_ADMIN_EMAIL"),
	}

	if cfg.Postgres.Host == "" {
		log.Fatal("config error: DB_HOST is not set")
	}
	if cfg.Postgres.Port == "" {
		log.Fatal("config error: DB_PORT is not set")
	}
	if cfg.Postgres.User == "" {
		log.Fatal("config error: DB_USER is not set")
	}
	if cfg.Postgres.Password == "" {
		log.Fatal("config error: DB_PASSWORD is not set")
	}
	if cfg.Postgres.Name == "" {
		log.Fatal("config error: DB_NAME is not set")
	}

	if cfg.Postgres.SSLMode == "" {
		cfg.Postgres.SSLMode = "disable"
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.GlobalAdminEmail == "" {
		cfg.GlobalAdminEmail = "admin@expense-splitter.local"
	}

	log.Printf("config loaded: port=%s postgres=%s@%s:%s/%s sslmode=%s",
		cfg.Port, cfg.Postgres.User, cfg.Postgres.Host,
		cfg.Postgres.Port, cfg.Postgres.Name, cfg.Postgres.SSLMode)

	if cfg.Keycloak.BaseURL == "" {
		cfg.Keycloak.BaseURL = baseFromJWKS(cfg.Keycloak.JWKSURL)
	}
	if cfg.Keycloak.Realm == "" {
		cfg.Keycloak.Realm = "expense-splitter"
	}
	if cfg.Keycloak.ClientID == "" {
		if cfg.Keycloak.Audience != "" {
			cfg.Keycloak.ClientID = cfg.Keycloak.Audience
		} else {
			cfg.Keycloak.ClientID = "expense-splitter-api"
		}
	}

	if cfg.Keycloak.Enabled() {
		log.Printf("keycloak auth: jwks=%s issuers=%s audience=%s",
			cfg.Keycloak.JWKSURL, strings.Join(cfg.Keycloak.Issuers, ","), cfg.Keycloak.Audience)
		log.Printf("keycloak calls: base=%s realm=%s client=%s login=%t admin=%t",
			cfg.Keycloak.BaseURL, cfg.Keycloak.Realm, cfg.Keycloak.ClientID,
			cfg.Keycloak.LoginEnabled(), cfg.Keycloak.AdminEnabled())
	} else {
		log.Print("keycloak auth: DISABLED (KEYCLOAK_JWKS_URL not set) — protected endpoints will reject all requests")
	}

	return cfg
}

// baseFromJWKS derives the Keycloak base URL (scheme://host[:port]) from the
// JWKS URL, so a single KEYCLOAK_JWKS_URL is enough when KEYCLOAK_BASE_URL is
// not set explicitly. "http://keycloak:8080/realms/x/..." -> "http://keycloak:8080".
func baseFromJWKS(jwks string) string {
	if i := strings.Index(jwks, "/realms/"); i > 0 {
		return jwks[:i]
	}
	return ""
}

func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
