package config

import (
	"testing"
	"time"
)

func TestLoadAPISigningKeys(t *testing.T) {
	t.Setenv("APP_API_SIGNING_KEYS", "integration-prod-01:secret:one,integration-stage-01:secret-two")

	keys, err := loadAPISigningKeys()
	if err != nil {
		t.Fatalf("load API signing keys: %v", err)
	}
	if keys["integration-prod-01"] != "secret:one" || keys["integration-stage-01"] != "secret-two" {
		t.Fatalf("unexpected API signing keys: %#v", keys)
	}
}

func TestLoadAPISigningKeysRejectsDuplicateKeyIDs(t *testing.T) {
	t.Setenv("APP_API_SIGNING_KEYS", "integration-prod-01:secret-one,integration-prod-01:secret-two")

	if _, err := loadAPISigningKeys(); err == nil {
		t.Fatal("expected duplicate API signing key ID error")
	}
}

func TestLoadAPISigningKeysAllowsEmptyConfiguration(t *testing.T) {
	t.Setenv("APP_API_SIGNING_KEYS", "")

	keys, err := loadAPISigningKeys()
	if err != nil {
		t.Fatalf("load empty API signing keys: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected no API signing keys, got %#v", keys)
	}
}

func TestLoadAPISignatureConfigDefaultsToDisabled(t *testing.T) {
	for _, key := range []string{
		"APP_API_SIGNATURE_ENABLED",
		"APP_API_SIGNATURE_VERSION",
		"APP_API_SIGNATURE_ROTATION_COUNT",
		"APP_API_SIGNATURE_TIMESTAMP_WINDOW",
		"APP_API_SIGNATURE_NONCE_KEY_PREFIX",
		"APP_API_SIGNATURE_MAX_BODY_BYTES",
		"APP_API_SIGNING_KEYS",
	} {
		t.Setenv(key, "")
	}

	config, err := loadAPISignatureConfig()
	if err != nil {
		t.Fatalf("load default API signature config: %v", err)
	}
	if config.Enabled || config.Version != "v1" || config.RotationCount != 4 || config.TimestampWindow != 5*time.Minute || config.NonceKeyPrefix != "api-signature:nonce:" || config.MaxSignedRequestBodyBytes != 1<<20 {
		t.Fatalf("unexpected default API signature config: %#v", config)
	}
}

func TestLoadAPISignatureConfigRejectsEnabledWithoutKeys(t *testing.T) {
	t.Setenv("APP_API_SIGNATURE_ENABLED", "true")
	t.Setenv("APP_API_SIGNING_KEYS", "")

	if _, err := loadAPISignatureConfig(); err == nil {
		t.Fatal("expected enabled API signature config without keys to fail")
	}
}

func TestAPISignatureConfigAcceptsSupportedRotationCounts(t *testing.T) {
	for _, rotationCount := range []int64{1, 3, 4, 5} {
		config := APISignatureConfig{
			Version:                   "v1",
			RotationCount:             rotationCount,
			TimestampWindow:           time.Minute,
			NonceKeyPrefix:            "api-signature:nonce:",
			MaxSignedRequestBodyBytes: 1024,
		}
		if err := config.Validate(); err != nil {
			t.Errorf("rotation count %d: unexpected validation error: %v", rotationCount, err)
		}
	}
}

func TestAPISignatureConfigRejectsUnsupportedRotationCounts(t *testing.T) {
	for _, rotationCount := range []int64{0, 6} {
		config := APISignatureConfig{
			Version:                   "v1",
			RotationCount:             rotationCount,
			TimestampWindow:           time.Minute,
			NonceKeyPrefix:            "api-signature:nonce:",
			MaxSignedRequestBodyBytes: 1024,
		}
		if err := config.Validate(); err == nil {
			t.Errorf("rotation count %d: expected validation error", rotationCount)
		}
	}
}
