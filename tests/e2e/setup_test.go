//go:build e2e

package e2e

import (
	"io"
	"log/slog"
	"testing"
	"time"

	app "gameco-technical-assessment/internal/app"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/seeds"
	testsupport "gameco-technical-assessment/tests/support"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func newE2ERouter(t *testing.T) *gin.Engine {
	router, _ := newE2ERouterWithDB(t)
	return router
}

func newE2ERouterWithDB(t *testing.T) (*gin.Engine, *gorm.DB) {
	return newE2ERouterWithDBSignature(t, false)
}

func newE2ERouterWithDBSignature(t *testing.T, signatureEnabled bool) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := testsupport.IntegrationConfig(t)
	cfg.App.PaymentWebhookSecret = "test-payment-webhook-secret"
	cfg.App.APISignature.Enabled = signatureEnabled
	if signatureEnabled {
		cfg.App.APISignature.Keys = map[string]string{"e2e-integration": "e2e-api-signing-secret"}
	}
	db := testsupport.OpenPostgres(t, cfg)
	testsupport.AssertMigrationsApplied(t, db)
	if err := db.Exec(`TRUNCATE TABLE
		audit_events,
		stock_movements,
		payment_webhook_events,
		payments,
		checkout_idempotency_keys,
		order_items,
		orders,
		cart_items,
		carts,
		products,
		categories,
		user_roles,
		role_permissions,
		permissions,
		roles,
		user_login_security_states,
		auth_sessions,
		users
		RESTART IDENTITY CASCADE`).Error; err != nil {
		t.Fatalf("reset e2e PostgreSQL database: %v", err)
	}
	if _, err := seeds.NewRunner(db).Run(); err != nil {
		t.Fatalf("seed authorization catalog: %v", err)
	}

	mini := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = redisClient.Close() })

	container := &app.Container{
		Config:    cfg,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		DB:        db,
		Redis:     redisClient,
		JWTSigner: platformauth.NewJWTManager(cfg.App.JWTSecret),
		Clock:     platformclock.Fixed{Current: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)},
	}
	return app.NewRouter(container), db
}
