package config

import "fmt"

func (c Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME is required")
	}

	if c.App.Port == "" {
		return fmt.Errorf("APP_PORT is required")
	}

	if c.App.JWTSecret == "" || c.App.JWTSecret == "change-me" {
		return fmt.Errorf("APP_JWT_SECRET must be explicitly configured")
	}

	if c.App.Env == "production" && c.App.PaymentWebhookSecret == "" {
		return fmt.Errorf("APP_PAYMENT_WEBHOOK_SECRET must be explicitly configured")
	}

	if c.App.APISignature.Enabled || c.App.APISignature.Version != "" || c.App.APISignature.RotationCount != 0 || c.App.APISignature.TimestampWindow != 0 || c.App.APISignature.NonceKeyPrefix != "" || c.App.APISignature.MaxSignedRequestBodyBytes != 0 || len(c.App.APISignature.Keys) != 0 {
		if err := c.App.APISignature.Validate(); err != nil {
			return err
		}
	}

	if c.App.AccessTokenTTL <= 0 {
		return fmt.Errorf("APP_ACCESS_TOKEN_TTL must be greater than zero")
	}

	if c.App.RefreshTokenTTL <= 0 {
		return fmt.Errorf("APP_REFRESH_TOKEN_TTL must be greater than zero")
	}

	if c.App.PaymentWebhookEventRetention <= 0 || c.App.PaymentWebhookCleanupInterval <= 0 {
		return fmt.Errorf("payment webhook retention and cleanup interval must be greater than zero")
	}

	if c.App.CheckoutIdempotencyRetention <= 0 || c.App.CheckoutIdempotencyCleanupInterval <= 0 {
		return fmt.Errorf("checkout idempotency retention and cleanup interval must be greater than zero")
	}

	if c.Postgres.Host == "" || c.Postgres.Port == "" || c.Postgres.User == "" || c.Postgres.Database == "" {
		return fmt.Errorf("postgres configuration is incomplete")
	}

	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}

	return nil
}
