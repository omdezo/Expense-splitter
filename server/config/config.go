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
			JWKSURL:  os.Getenv("KEYCLOAK_JWKS_URL"),
			Issuers:  splitAndTrim(os.Getenv("KEYCLOAK_ISSUERS")),
			Audience: os.Getenv("KEYCLOAK_AUDIENCE"),
		},
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

	log.Printf("config loaded: port=%s postgres=%s@%s:%s/%s sslmode=%s",
		cfg.Port, cfg.Postgres.User, cfg.Postgres.Host,
		cfg.Postgres.Port, cfg.Postgres.Name, cfg.Postgres.SSLMode)

	if cfg.Keycloak.Enabled() {
		log.Printf("keycloak auth: jwks=%s issuers=%s audience=%s",
			cfg.Keycloak.JWKSURL, strings.Join(cfg.Keycloak.Issuers, ","), cfg.Keycloak.Audience)
	} else {
		log.Print("keycloak auth: DISABLED (KEYCLOAK_JWKS_URL not set) — protected endpoints will reject all requests")
	}

	return cfg
}

// splitAndTrim turns a comma-separated env value into a slice, dropping blanks
// and surrounding whitespace. Returns nil for an empty input.
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
