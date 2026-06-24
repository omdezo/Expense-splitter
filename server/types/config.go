package types

import "fmt"

type Config struct {
	Port     string
	Postgres PostgresConfig
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN builds the PostgreSQL connection string from the individual fields.
func (p PostgresConfig) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Name, p.SSLMode)
}
