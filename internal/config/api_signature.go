package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type APISignatureConfig struct {
	Enabled                   bool
	Version                   string
	RotationCount             int64
	TimestampWindow           time.Duration
	NonceKeyPrefix            string
	MaxSignedRequestBodyBytes int
	Keys                      map[string]string
}

const (
	DefaultAPISignatureRotationCount = 4
	MaxAPISignatureRotationCount     = 5
)

func loadAPISignatureConfig() (APISignatureConfig, error) {
	enabled, err := getEnvBool("APP_API_SIGNATURE_ENABLED", false)
	if err != nil {
		return APISignatureConfig{}, err
	}
	rotationCount, err := getEnvInt("APP_API_SIGNATURE_ROTATION_COUNT", DefaultAPISignatureRotationCount)
	if err != nil {
		return APISignatureConfig{}, err
	}
	timestampWindow, err := getEnvDuration("APP_API_SIGNATURE_TIMESTAMP_WINDOW", 5*time.Minute)
	if err != nil {
		return APISignatureConfig{}, err
	}
	maxBodyBytes, err := getEnvInt("APP_API_SIGNATURE_MAX_BODY_BYTES", 1<<20)
	if err != nil {
		return APISignatureConfig{}, err
	}
	keys, err := loadAPISigningKeys()
	if err != nil {
		return APISignatureConfig{}, err
	}

	config := APISignatureConfig{
		Enabled:                   enabled,
		Version:                   getEnv("APP_API_SIGNATURE_VERSION", "v1"),
		RotationCount:             int64(rotationCount),
		TimestampWindow:           timestampWindow,
		NonceKeyPrefix:            getEnv("APP_API_SIGNATURE_NONCE_KEY_PREFIX", "api-signature:nonce:"),
		MaxSignedRequestBodyBytes: maxBodyBytes,
		Keys:                      keys,
	}
	if err := config.Validate(); err != nil {
		return APISignatureConfig{}, err
	}
	return config, nil
}

func (c APISignatureConfig) Validate() error {
	if strings.TrimSpace(c.Version) == "" || strings.ContainsAny(c.Version, "|,\n\r") {
		return fmt.Errorf("APP_API_SIGNATURE_VERSION must be a non-empty safe value")
	}
	if c.RotationCount < 1 || c.RotationCount > MaxAPISignatureRotationCount {
		return fmt.Errorf("APP_API_SIGNATURE_ROTATION_COUNT must be between 1 and %d", MaxAPISignatureRotationCount)
	}
	if c.TimestampWindow <= 0 {
		return fmt.Errorf("APP_API_SIGNATURE_TIMESTAMP_WINDOW must be greater than zero")
	}
	if strings.TrimSpace(c.NonceKeyPrefix) == "" {
		return fmt.Errorf("APP_API_SIGNATURE_NONCE_KEY_PREFIX must be non-empty")
	}
	if c.MaxSignedRequestBodyBytes <= 0 {
		return fmt.Errorf("APP_API_SIGNATURE_MAX_BODY_BYTES must be greater than zero")
	}
	if c.Enabled && len(c.Keys) == 0 {
		return fmt.Errorf("APP_API_SIGNING_KEYS must be configured when API signature verification is enabled")
	}
	return nil
}

func getEnvBool(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(getEnv(key, ""))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}
