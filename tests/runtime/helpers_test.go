//go:build runtime

package runtime_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
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
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
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

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}
}

func runtimeEmail() string {
	return fmt.Sprintf("runtime-%d@example.com", time.Now().UnixNano())
}
