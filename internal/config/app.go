package config

import (
	"fmt"
	"strings"
	"time"
)

const maxPreviousJWTSecrets = 3

type AppConfig struct {
	Name                               string
	Env                                string
	Port                               string
	ReadTimeout                        time.Duration
	WriteTimeout                       time.Duration
	ShutdownTimeout                    time.Duration
	JWTSecret                          string
	JWTPreviousSecrets                 []string
	PaymentWebhookSecret               string
	PaymentWebhookEventRetention       time.Duration
	PaymentWebhookCleanupInterval      time.Duration
	CheckoutIdempotencyRetention       time.Duration
	CheckoutIdempotencyCleanupInterval time.Duration
	AccessTokenTTL                     time.Duration
	RefreshTokenTTL                    time.Duration
}

func loadAppConfig() (AppConfig, error) {
	readTimeout, err := getEnvDuration("APP_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	writeTimeout, err := getEnvDuration("APP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	shutdownTimeout, err := getEnvDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	accessTokenTTL, err := getEnvDuration("APP_ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return AppConfig{}, err
	}

	refreshTokenTTL, err := getEnvDuration("APP_REFRESH_TOKEN_TTL", 24*time.Hour)
	if err != nil {
		return AppConfig{}, err
	}

	webhookRetention, err := getEnvDuration("APP_PAYMENT_WEBHOOK_EVENT_RETENTION", 7*24*time.Hour)
	if err != nil {
		return AppConfig{}, err
	}
	webhookCleanupInterval, err := getEnvDuration("APP_PAYMENT_WEBHOOK_CLEANUP_INTERVAL", time.Hour)
	if err != nil {
		return AppConfig{}, err
	}

	checkoutIdempotencyRetention, err := getEnvDuration("APP_CHECKOUT_IDEMPOTENCY_RETENTION", 30*24*time.Hour)
	if err != nil {
		return AppConfig{}, err
	}
	checkoutIdempotencyInterval, err := getEnvDuration("APP_CHECKOUT_IDEMPOTENCY_CLEANUP_INTERVAL", time.Hour)
	if err != nil {
		return AppConfig{}, err
	}

	jwtSecret, previousSecrets, err := loadJWTSecrets()
	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		Name:                               getEnv("APP_NAME", "gameco-technical-assessment"),
		Env:                                getEnv("APP_ENV", "development"),
		Port:                               getEnv("APP_PORT", "8080"),
		ReadTimeout:                        readTimeout,
		WriteTimeout:                       writeTimeout,
		ShutdownTimeout:                    shutdownTimeout,
		JWTSecret:                          jwtSecret,
		JWTPreviousSecrets:                 previousSecrets,
		PaymentWebhookSecret:               getEnv("APP_PAYMENT_WEBHOOK_SECRET", ""),
		PaymentWebhookEventRetention:       webhookRetention,
		PaymentWebhookCleanupInterval:      webhookCleanupInterval,
		CheckoutIdempotencyRetention:       checkoutIdempotencyRetention,
		CheckoutIdempotencyCleanupInterval: checkoutIdempotencyInterval,
		AccessTokenTTL:                     accessTokenTTL,
		RefreshTokenTTL:                    refreshTokenTTL,
	}, nil
}

func loadJWTSecrets() (string, []string, error) {
	current := getEnv("APP_JWT_SECRET", "")
	if current == "" {
		return "", nil, fmt.Errorf("APP_JWT_SECRET is required")
	}

	configured := getEnv("APP_JWT_PREVIOUS_SECRETS", "")
	if strings.TrimSpace(configured) == "" {
		return current, nil, nil
	}

	entries := strings.Split(configured, ",")
	if len(entries) > maxPreviousJWTSecrets {
		return "", nil, fmt.Errorf("APP_JWT_PREVIOUS_SECRETS supports at most %d keys", maxPreviousJWTSecrets)
	}

	previous := make([]string, 0, len(entries))
	seen := map[string]struct{}{current: {}}
	for _, entry := range entries {
		secret := strings.TrimSpace(entry)
		if secret == "" {
			return "", nil, fmt.Errorf("APP_JWT_PREVIOUS_SECRETS contains an empty key")
		}
		if _, exists := seen[secret]; exists {
			return "", nil, fmt.Errorf("APP_JWT_PREVIOUS_SECRETS contains a duplicate key")
		}
		seen[secret] = struct{}{}
		previous = append(previous, secret)
	}
	return current, previous, nil
}
