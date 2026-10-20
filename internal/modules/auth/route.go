package auth

import (
	"context"
	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/modules/auth/permission"
	platformauth "gameco-technical-assessment/internal/platform/auth"
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
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository, deps.Redis, deps.JWTSigner, deps.AppConfig)
	handler := NewHandler(service)
	resolve := func(ctx context.Context, jti string) ([]string, []string, error) {
		access, err := service.ValidateSession(ctx, jti)
		if err != nil {
			return nil, nil, err
		}
		return access.Roles, access.Permissions, nil
	}

	group := router.Group("/auth")
	group.POST("/register", handler.Register)
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.Refresh)
	group.POST("/logout", handler.Logout)
	group.GET("/me", middleware.Auth(deps.JWTSigner, resolve), middleware.RequirePermissions(permission.AuthMe), handler.Me)
}
