package e2e

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"testing"
	"time"

	app "gameco-technical-assessment/internal/app"
	"gameco-technical-assessment/internal/config"
	authmodule "gameco-technical-assessment/internal/modules/auth"
	cartmodule "gameco-technical-assessment/internal/modules/cart"
	catalogmodule "gameco-technical-assessment/internal/modules/catalog"
	checkoutmodule "gameco-technical-assessment/internal/modules/checkout"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/seeds"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newE2ERouter(t *testing.T) *gin.Engine {
	router, _ := newE2ERouterWithDB(t)
	return router
}

func newE2ERouterWithDB(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:e2e-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
	})
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
		&authmodule.LoginSecurityState{},
		&catalogmodule.Category{},
		&catalogmodule.Product{},
		&cartmodule.Cart{},
		&cartmodule.CartItem{},
		&checkoutmodule.Order{},
		&checkoutmodule.OrderItem{},
		&checkoutmodule.Payment{},
		&checkoutmodule.IdempotencyKey{},
		&checkoutmodule.StockMovement{},
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
	return app.NewRouter(container), db
}
