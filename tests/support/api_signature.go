package testsupport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"

	"gameco-technical-assessment/internal/config"
)

func APISignatureHeaders(method, path string, body []byte, timestamp int64, nonce, keyID, secret string) map[string]string {
	bodyHash := sha256.Sum256(body)
	fields := []string{
		method,
		path,
		strconv.FormatInt(timestamp, 10),
		nonce,
		hex.EncodeToString(bodyHash[:]),
	}
	rotationCount := int64(config.DefaultAPISignatureRotationCount)
	if configured := strings.TrimSpace(os.Getenv("APP_API_SIGNATURE_ROTATION_COUNT")); configured != "" {
		if parsed, err := strconv.ParseInt(configured, 10, 64); err == nil && parsed >= 1 && parsed <= config.MaxAPISignatureRotationCount {
			rotationCount = parsed
		}
	}
	rotation := int(timestamp % rotationCount)
	rotated := append(append([]string{}, fields[rotation:]...), fields[:rotation]...)

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(strings.Join(rotated, "|")))

	return map[string]string{
		"X-API-Key":       keyID,
		"X-API-Timestamp": strconv.FormatInt(timestamp, 10),
		"X-API-Nonce":     nonce,
		"X-API-Signature": "v1=" + hex.EncodeToString(mac.Sum(nil)),
	}
}
