package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	app "gameco-technical-assessment/internal/app"
	"gameco-technical-assessment/internal/config"
	authmodule "gameco-technical-assessment/internal/modules/auth"
	catalogmodule "gameco-technical-assessment/internal/modules/catalog"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/seeds"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAuthRegisterLoginThenMe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open("file:auth-e2e?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	if err := db.AutoMigrate(
		&authmodule.User{},
		&authmodule.Role{},
		&authmodule.Permission{},
		&authmodule.RolePermission{},
		&authmodule.UserRole{},
		&authmodule.AuthSession{},
		&catalogmodule.Category{},
		&catalogmodule.Product{},
	); err != nil {
		t.Fatalf("migrate sqlite database: %v", err)
	}
	if _, err := seeds.NewRunner(db).Run(); err != nil {
		t.Fatalf("seed authorization catalog: %v", err)
	}

	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	container := &app.Container{
		Config: config.Config{
			App: config.AppConfig{
				Env:             "test",
				JWTSecret:       "test-secret",
				AccessTokenTTL:  15 * time.Minute,
				RefreshTokenTTL: time.Hour,
			},
			RateLimit: config.RateLimitConfig{
				Anonymous:     config.RateLimitPolicyConfig{Limit: 1000, Window: time.Minute},
				Authenticated: config.RateLimitPolicyConfig{Limit: 1000, Window: time.Minute},
			},
		},
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        db,
		Redis:     redisClient,
		JWTSigner: platformauth.NewJWTManager("test-secret"),
	}
	router := app.NewRouter(container)

	registerBody := `{"email":"e2e@example.com","password":"correct-horse-battery-staple"}`
	registerRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(registerBody))
	registerRequest.Header.Set("Content-Type", "application/json")
	registerResponse := httptest.NewRecorder()
	router.ServeHTTP(registerResponse, registerRequest)

	if registerResponse.Code != http.StatusCreated {
		t.Fatalf("register: expected status 201, got %d: %s", registerResponse.Code, registerResponse.Body.String())
	}

	var registerPayload authTokenEnvelope
	if err := json.Unmarshal(registerResponse.Body.Bytes(), &registerPayload); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	if !registerPayload.Success || registerPayload.Data.AccessToken == "" {
		t.Fatalf("register: expected successful token response, got %+v", registerPayload)
	}

	loginBody := `{"email":"e2e@example.com","password":"correct-horse-battery-staple"}`
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, loginRequest)

	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login: expected status 200, got %d: %s", loginResponse.Code, loginResponse.Body.String())
	}

	var loginPayload authTokenEnvelope
	if err := json.Unmarshal(loginResponse.Body.Bytes(), &loginPayload); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if !loginPayload.Success || loginPayload.Data.AccessToken == "" || loginPayload.Data.RefreshToken == "" {
		t.Fatalf("login: expected successful token response, got %+v", loginPayload)
	}

	refreshBody, err := json.Marshal(map[string]string{"refresh_token": loginPayload.Data.RefreshToken})
	if err != nil {
		t.Fatalf("encode refresh request: %v", err)
	}
	refreshRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	refreshRequest.Header.Set("Content-Type", "application/json")
	refreshResponse := httptest.NewRecorder()
	router.ServeHTTP(refreshResponse, refreshRequest)
	if refreshResponse.Code != http.StatusOK {
		t.Fatalf("refresh: expected status 200, got %d: %s", refreshResponse.Code, refreshResponse.Body.String())
	}
	var refreshPayload authTokenEnvelope
	if err := json.Unmarshal(refreshResponse.Body.Bytes(), &refreshPayload); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if refreshPayload.Data.AccessToken == "" || refreshPayload.Data.RefreshToken == "" {
		t.Fatalf("refresh: expected rotated token pair, got %+v", refreshPayload)
	}

	meRequest := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+refreshPayload.Data.AccessToken)
	meResponse := httptest.NewRecorder()
	router.ServeHTTP(meResponse, meRequest)

	if meResponse.Code != http.StatusOK {
		t.Fatalf("me: expected status 200, got %d: %s", meResponse.Code, meResponse.Body.String())
	}

	var mePayload userEnvelope
	if err := json.Unmarshal(meResponse.Body.Bytes(), &mePayload); err != nil {
		t.Fatalf("decode me response: %v", err)
	}
	if !mePayload.Success {
		t.Fatalf("me: expected successful response, got %+v", mePayload)
	}
	if mePayload.Data.Email != "e2e@example.com" {
		t.Fatalf("me: expected email e2e@example.com, got %q", mePayload.Data.Email)
	}
	if !contains(mePayload.Data.Roles, "buyer") || !contains(mePayload.Data.Roles, "seller") {
		t.Fatalf("me: expected buyer and seller roles, got %v", mePayload.Data.Roles)
	}

	categoryOptionsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/categories/options", nil)
	categoryOptionsResponse := httptest.NewRecorder()
	router.ServeHTTP(categoryOptionsResponse, categoryOptionsRequest)
	if categoryOptionsResponse.Code != http.StatusOK {
		t.Fatalf("category options: expected status 200, got %d: %s", categoryOptionsResponse.Code, categoryOptionsResponse.Body.String())
	}

	sellerProductsRequest := httptest.NewRequest(http.MethodGet, "/api/v1/seller/products", nil)
	sellerProductsRequest.Header.Set("Authorization", "Bearer "+refreshPayload.Data.AccessToken)
	sellerProductsResponse := httptest.NewRecorder()
	router.ServeHTTP(sellerProductsResponse, sellerProductsRequest)
	if sellerProductsResponse.Code != http.StatusOK {
		t.Fatalf("seller products: expected status 200, got %d: %s", sellerProductsResponse.Code, sellerProductsResponse.Body.String())
	}

	reusedRefreshRequest := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
	reusedRefreshRequest.Header.Set("Content-Type", "application/json")
	reusedRefreshResponse := httptest.NewRecorder()
	router.ServeHTTP(reusedRefreshResponse, reusedRefreshRequest)
	if reusedRefreshResponse.Code != http.StatusUnauthorized {
		t.Fatalf("reused refresh: expected status 401, got %d: %s", reusedRefreshResponse.Code, reusedRefreshResponse.Body.String())
	}
}

type authTokenEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}

type userEnvelope struct {
	Success bool `json:"success"`
	Data    struct {
		Email string   `json:"email"`
		Roles []string `json:"roles"`
	} `json:"data"`
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == expected {
			return true
		}
	}
	return false
}
