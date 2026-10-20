package app

import (
	moduleauth "gameco-technical-assessment/internal/modules/auth"
	modulehealth "gameco-technical-assessment/internal/modules/health"
	"gameco-technical-assessment/internal/platform/middleware"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func NewRouter(container *Container) *gin.Engine {
	if container.Config.App.Env == "production" || container.Config.App.Env == "test" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		panic(err)
	}
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())
	router.Use(middleware.Recovery(container.Logger))
	router.Use(middleware.Logging(container.Logger))
	swaggerUI := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.DocExpansion("none"),
	)
	router.GET("/swagger/*any", swaggerUI)

	modulehealth.RegisterRoutes(router, modulehealth.NewHandler())

	api := router.Group("/api/v1")
	api.Use(middleware.RateLimit(container.Redis, middleware.RateLimitPolicySet{
		Anonymous: middleware.RateLimitPolicy{
			Name:   "global-ip",
			Limit:  container.Config.RateLimit.Anonymous.Limit,
			Window: container.Config.RateLimit.Anonymous.Window,
		},
		Authenticated: middleware.RateLimitPolicy{
			Name:   "global-user",
			Limit:  container.Config.RateLimit.Authenticated.Limit,
			Window: container.Config.RateLimit.Authenticated.Window,
		},
	}, container.JWTSigner))

	moduleauth.RegisterRoutes(api, moduleauth.RouteDependencies{
		DB:        container.DB,
		Redis:     container.Redis,
		JWTSigner: container.JWTSigner,
		AppConfig: container.Config.App,
		RateLimit: container.Config.RateLimit,
	})

	return router
}
