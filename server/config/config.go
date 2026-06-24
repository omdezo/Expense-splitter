package config

import (
	"log"
	"os"

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

	return cfg
}
