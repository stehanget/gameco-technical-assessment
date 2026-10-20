package auth

import (
	"context"
	"time"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/constants/permission"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RouteDependencies struct {
	DB        *gorm.DB
	Redis     *redis.Client
	JWTSigner *platformauth.JWTManager
	AppConfig config.AppConfig
	RateLimit config.RateLimitConfig
	Clock     platformclock.Clock
}

const authRateLimitPolicyName = "auth"

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepositoryWithClock(deps.DB, deps.Redis, deps.Clock)
	service := NewServiceWithClock(repository, deps.Redis, deps.JWTSigner, deps.AppConfig, deps.Clock)
	handler := NewHandler(service)
	resolve := func(ctx context.Context, jti string) ([]string, []string, error) {
		access, err := service.ValidateSession(ctx, jti)
		if err != nil {
			return nil, nil, err
		}
		return access.Roles, access.Permissions, nil
	}

	group := router.Group("/auth")
	public := group.Group("")
	public.Use(middleware.RateLimitWithClock(deps.Redis, middleware.RateLimitPolicySet{
		Anonymous: middleware.RateLimitPolicy{
			Name:   authRateLimitPolicyName,
			Limit:  10,
			Window: time.Minute,
		},
	}, deps.JWTSigner, deps.Clock))
	public.POST("/register", handler.Register)
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.Refresh)
	public.POST("/logout", handler.Logout)

	protected := group.Group("")
	protected.Use(middleware.Auth(deps.JWTSigner, resolve))
	protected.Use(middleware.RateLimitWithClock(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: middleware.RateLimitPolicy{
			Name:   "authenticated-user",
			Limit:  60,
			Window: time.Minute,
		},
	}, deps.JWTSigner, deps.Clock))
	protected.GET("/me", middleware.RequirePermissions(permission.AuthMe), handler.Me)
}
