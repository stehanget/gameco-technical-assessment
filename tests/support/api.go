package testsupport

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	app "gameco-technical-assessment/internal/app"
	"gameco-technical-assessment/internal/config"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
)

func NewAPI(t *testing.T, cfg config.Config) *httptest.Server {
	t.Helper()
	db := OpenPostgres(t, cfg)
	redisClient := OpenRedis(t, cfg)
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
	cfg.RateLimit.Anonymous.Limit = 1000
	cfg.RateLimit.Anonymous.Window = time.Minute
	cfg.RateLimit.Authenticated.Limit = 1000
	cfg.RateLimit.Authenticated.Window = time.Minute
	if err := cfg.Validate(); err != nil {
		t.Fatalf("validate integration config: %v", err)
	}
	return cfg
}
