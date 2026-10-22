package config

import (
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		App: AppConfig{
			Name:                               "test",
			Port:                               "8080",
			ShutdownTimeout:                    10 * time.Second,
			MaxRequestBodyBytes:                DefaultMaxRequestBodyBytes,
			JWTSecret:                          "test-secret",
			PaymentWebhookEventRetention:       7 * 24 * time.Hour,
			PaymentWebhookCleanupInterval:      time.Hour,
			CheckoutIdempotencyRetention:       30 * 24 * time.Hour,
			CheckoutIdempotencyCleanupInterval: time.Hour,
			AccessTokenTTL:                     time.Minute,
			RefreshTokenTTL:                    time.Hour,
		},
		Postgres: PostgresConfig{Host: "localhost", Port: "5432", User: "user", Database: "db"},
		Redis:    RedisConfig{Addr: "localhost:6379"},
	}
}

func TestValidateRejectsPlaceholderJWTSecret(t *testing.T) {
	config := validConfig()
	config.App.JWTSecret = "change-me"
	if err := config.Validate(); err == nil {
		t.Fatal("expected placeholder JWT secret to be rejected")
	}
}

func TestValidateAcceptsConfiguredJWTSecret(t *testing.T) {
	if err := validConfig().Validate(); err != nil {
		t.Fatalf("expected configured JWT secret to pass validation: %v", err)
	}
}

func TestLoadPostgresDoesNotRequireApplicationSecrets(t *testing.T) {
	t.Setenv("APP_JWT_SECRET", "")

	config := LoadPostgres()
	if config.Postgres.Host == "" || config.Postgres.Database == "" {
		t.Fatal("expected PostgreSQL configuration to load without application secrets")
	}
}
