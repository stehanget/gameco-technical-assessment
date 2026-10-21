package auth

import (
	"context"
	"time"

	"gameco-technical-assessment/internal/constants/permission"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RouteDependencies struct {
	Service   *Service
	Redis     *redis.Client
	JWTSigner *platformauth.JWTManager
}

const authRateLimitPolicyName = "auth"

func NewActorResolver(service *Service) middleware.ActorResolver {
	resolve := func(ctx context.Context, jti string) (platformauth.Actor, error) {
		access, err := service.ValidateSession(ctx, jti)
		if err != nil {
			return platformauth.Actor{}, err
		}
		return platformauth.Actor{UserID: access.UserID, Roles: access.Roles, Permissions: access.Permissions}, nil
	}
	return resolve
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	handler := NewHandler(deps.Service)
	resolveActor := NewActorResolver(deps.Service)

	group := router.Group("/auth")
	public := group.Group("")
	public.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Anonymous: middleware.RateLimitPolicy{
			Name:   authRateLimitPolicyName,
			Limit:  10,
			Window: time.Minute,
		},
	}, deps.JWTSigner))
	public.POST("/register", handler.Register)
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.Refresh)
	public.POST("/logout", handler.Logout)

	protected := group.Group("")
	protected.Use(middleware.Auth(deps.JWTSigner, resolveActor))
	protected.Use(middleware.RateLimit(deps.Redis, middleware.RateLimitPolicySet{
		Authenticated: middleware.RateLimitPolicy{
			Name:   "authenticated-user",
			Limit:  60,
			Window: time.Minute,
		},
	}, deps.JWTSigner))
	protected.GET("/me", middleware.RequirePermissions(permission.AuthMe), handler.Me)
}
