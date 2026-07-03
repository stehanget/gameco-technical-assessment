package database

import (
	"context"
	"fmt"

	"gameco-technical-assessment/internal/config"

	"github.com/pressly/goose/v3"
)

func RunGoose(ctx context.Context, cfg config.Config, command string) error {
	db, err := NewSQLDB(cfg)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.RunContext(ctx, command, db, "migrations"); err != nil {
		return fmt.Errorf("run goose %s: %w", command, err)
	}

	return nil
}
