package config

import (
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		App: AppConfig{
			Name:            "test",
			Port:            "8080",
			JWTSecret:       "test-secret",
			AccessTokenTTL:  time.Minute,
			RefreshTokenTTL: time.Hour,
		},
		Postgres: PostgresConfig{Host: "localhost", Port: "5432", User: "user", Database: "db"},
		Redis:    RedisConfig{Addr: "localhost:6379"},
		RateLimit: RateLimitConfig{
			Anonymous:     RateLimitPolicyConfig{Limit: 1, Window: time.Minute},
			Authenticated: RateLimitPolicyConfig{Limit: 1, Window: time.Minute},
		},
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
