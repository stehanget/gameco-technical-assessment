package checkout

import (
	"time"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/constants/permission"
	"gameco-technical-assessment/internal/constants/ratelimit"
	"gameco-technical-assessment/internal/constants/role"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RouteDependencies struct {
	DB                   *gorm.DB
	Redis                *redis.Client
	JWTSigner            *platformauth.JWTManager
	ResolveActor         middleware.ActorResolver
	APISignature         config.APISignatureConfig
	PaymentWebhookSecret string
	Clock                platformclock.Clock
}

const webhookRateLimitPolicyName = "payment-webhook"

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository)
	handler := NewHandler(service, deps.PaymentWebhookSecret, deps.Clock)

	group := router.Group("")
	group.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	group.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	group.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	group.Use(middleware.RequireRoles(role.Buyer))
	group.POST("/checkout", handler.Create)
	group.GET("/orders/:order_public_id", handler.Detail)

	sellerOrders := router.Group("/seller/orders")
	sellerOrders.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	sellerOrders.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: ratelimit.GlobalAuthenticatedPolicy,
	}, deps.JWTSigner))
	sellerOrders.Use(middleware.RequireAPISignature(deps.APISignature, deps.Redis))
	sellerOrders.Use(middleware.RequireRoles(role.Seller))
	sellerOrders.Use(middleware.RequirePermissions(permission.SellerOrderRead))
	sellerOrders.GET("", handler.SellerOrders)

	public := router.Group("")
	public.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Anonymous: middleware.RateLimitPolicy{
			Name:   webhookRateLimitPolicyName,
			Limit:  10000,
			Window: time.Minute,
		},
	}, deps.JWTSigner))
	public.POST("/webhooks/payment", handler.PaymentWebhook)
}
