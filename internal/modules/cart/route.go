package cart

import (
	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/constants/ratelimit"
	"gameco-technical-assessment/internal/constants/role"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RouteDependencies struct {
	DB           *gorm.DB
	Redis        *redis.Client
	JWTSigner    *platformauth.JWTManager
	ResolveActor middleware.ActorResolver
	APISignature config.APISignatureConfig
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository)
	handler := NewHandler(service)

	group := router.Group("/cart")
	group.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	group.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	group.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	group.Use(middleware.RequireRoles(role.Buyer))
	group.GET("", handler.Get)
	group.PUT("/items/:product_public_id", handler.SetItem)
	group.DELETE("/items/:product_public_id", handler.DeleteItem)
}
