package config

import "fmt"

func (c Config) Validate() error {
	if c.App.Name == "" {
		return fmt.Errorf("APP_NAME is required")
	}

	if c.App.Port == "" {
		return fmt.Errorf("APP_PORT is required")
	}

	if c.App.JWTSecret == "" {
		return fmt.Errorf("APP_JWT_SECRET is required")
	}

	if c.App.AccessTokenTTL <= 0 {
		return fmt.Errorf("APP_ACCESS_TOKEN_TTL must be greater than zero")
	}

	if c.App.RefreshTokenTTL <= 0 {
		return fmt.Errorf("APP_REFRESH_TOKEN_TTL must be greater than zero")
	}

	if c.Postgres.Host == "" || c.Postgres.Port == "" || c.Postgres.User == "" || c.Postgres.Database == "" {
		return fmt.Errorf("postgres configuration is incomplete")
	}

	if c.Redis.Addr == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}

	return nil
}
