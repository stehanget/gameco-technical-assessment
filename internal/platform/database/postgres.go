package database

import (
	"database/sql"
	"fmt"

	"gameco-technical-assessment/internal/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func BuildPostgresDSN(cfg config.Config) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.User,
		cfg.Postgres.Password,
		cfg.Postgres.Database,
		cfg.Postgres.SSLMode,
	)
}

func NewSQLDB(cfg config.Config) (*sql.DB, error) {
	return sql.Open("pgx", BuildPostgresDSN(cfg))
}
