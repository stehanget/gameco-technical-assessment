//go:build runtime

package runtime_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	testsupport "gameco-technical-assessment/tests/support"
)

func runtimeBaseURL(t *testing.T) string {
	t.Helper()
	baseURL := strings.TrimRight(os.Getenv("RUNTIME_BASE_URL"), "/")
	if baseURL == "" {
		t.Fatal("RUNTIME_BASE_URL must be set")
	}
	return baseURL
}

func requestJSON(t *testing.T, method string, url string, body io.Reader, token string) (int, []byte) {
	t.Helper()
	return requestJSONWithHeaders(t, method, url, body, token, nil)
}

func requestJSONWithHeaders(t *testing.T, method string, url string, body io.Reader, token string, headers map[string]string) (int, []byte) {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = io.ReadAll(body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
	}
	request, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if _, alreadySigned := headers["X-API-Signature"]; !alreadySigned {
		if signatureHeaders := runtimeAPISignatureHeaders(t, method, url, bodyBytes); signatureHeaders != nil {
			for key, value := range signatureHeaders {
				request.Header.Set(key, value)
			}
		}
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read response %s %s: %v", method, url, err)
	}
	return response.StatusCode, responseBody
}

func runtimeAPISignatureHeaders(t *testing.T, method, rawURL string, body []byte) map[string]string {
	t.Helper()
	if strings.TrimSpace(os.Getenv("APP_API_SIGNATURE_ENABLED")) != "true" {
		return nil
	}
	keyID, secret := runtimeAPISigningCredentials(t)
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse runtime request URL: %v", err)
	}
	path := parsed.EscapedPath()
	if parsed.RawQuery != "" {
		path += "?" + parsed.RawQuery
	}
	return testsupport.APISignatureHeaders(method, path, body, time.Now().Unix(), fmt.Sprintf("runtime-%d", time.Now().UnixNano()), keyID, secret)
}

func runtimeAPISigningCredentials(t *testing.T) (string, string) {
	t.Helper()
	firstEntry := strings.SplitN(strings.TrimSpace(os.Getenv("APP_API_SIGNING_KEYS")), ",", 2)[0]
	keyID, secret, ok := strings.Cut(firstEntry, ":")
	if !ok || keyID == "" || secret == "" {
		t.Fatal("APP_API_SIGNING_KEYS must contain key-id:secret for runtime signature tests")
	}
	return keyID, secret
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}
}

func runtimeEmail() string {
	return fmt.Sprintf("runtime-%d@example.com", time.Now().UnixNano())
}
