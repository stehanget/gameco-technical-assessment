package pagination

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// FilterFingerprint returns a stable fingerprint for the named effective list inputs.
func FilterFingerprint(fields map[string]string) string {
	payload, _ := json.Marshal(fields)
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}
