package catalog

import (
	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/constants/permission"
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

	group := router.Group("/products")
	group.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Anonymous:     ratelimit.GlobalAnonymousPolicy,
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	group.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	group.GET("", handler.List)
	group.GET("/:product_public_id", handler.Detail)

	categoryOptions := router.Group("/categories/options")
	categoryOptions.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Anonymous:     ratelimit.GlobalAnonymousPolicy,
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	categoryOptions.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	categoryOptions.GET("", handler.CategoryOptions)

	seller := router.Group("/seller/products")
	seller.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	seller.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	seller.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	seller.Use(middleware.RequireRoles(role.Seller))
	seller.Use(middleware.RequirePermissions(permission.ProductRead))
	seller.GET("", handler.SellerList)
	seller.GET("/:product_public_id", handler.SellerDetail)
	seller.POST("", middleware.RequirePermissions(permission.ProductCreate), handler.Create)
	seller.PATCH("/:product_public_id", middleware.RequirePermissions(permission.ProductUpdate), handler.Update)
}
