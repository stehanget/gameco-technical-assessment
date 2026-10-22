package testsupport

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	app "gameco-technical-assessment/internal/app"
	"gameco-technical-assessment/internal/config"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
)

func NewAPI(t *testing.T, cfg config.Config) *httptest.Server {
	t.Helper()
	db := OpenPostgres(t, cfg)
	AssertMigrationsApplied(t, db)
	redisClient := OpenRedis(t, cfg)
	if err := redisClient.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush Redis test database: %v", err)
	}
	clock := platformclock.System{}
	container := &app.Container{
		Config:    cfg,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        db,
		Redis:     redisClient,
		JWTSigner: platformauth.NewJWTManager(cfg.App.JWTSecret),
		Clock:     clock,
	}
	server := httptest.NewServer(app.NewRouter(container))
	t.Cleanup(func() {
		server.Close()
	})
	return server
}

func IntegrationConfig(t *testing.T) config.Config {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load integration config: %v", err)
	}
	cfg.App.Env = "test"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate integration config: %v", err)
	}
	return cfg
}
