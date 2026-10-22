package app

import (
	"io"
	"log/slog"
	"testing"

	"gameco-technical-assessment/internal/config"
	platformauth "gameco-technical-assessment/internal/platform/auth"

	"github.com/gin-gonic/gin"
)

func TestDevelopmentSignatureRoutesAreOnlyRegisteredInDevelopment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name        string
		env         string
		wantPresent bool
	}{
		{name: "development", env: "development", wantPresent: true},
		{name: "test", env: "test", wantPresent: false},
		{name: "production", env: "production", wantPresent: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter(&Container{
				Config:    config.Config{App: config.AppConfig{Env: test.env}},
				Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
				JWTSigner: platformauth.NewJWTManager("test-secret"),
			})
			found := false
			for _, route := range router.Routes() {
				if route.Method == "POST" && route.Path == "/api/v1/dev/signatures/api" {
					found = true
					break
				}
			}
			if found != test.wantPresent {
				t.Fatalf("development API signer route present=%v, want=%v", found, test.wantPresent)
			}
		})
	}
}
