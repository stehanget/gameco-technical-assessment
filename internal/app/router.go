package app

import (
	moduleauth "gameco-technical-assessment/internal/modules/auth"
	modulecart "gameco-technical-assessment/internal/modules/cart"
	modulecatalog "gameco-technical-assessment/internal/modules/catalog"
	modulecheckout "gameco-technical-assessment/internal/modules/checkout"
	moduledevtools "gameco-technical-assessment/internal/modules/devtools"
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
	router.Use(middleware.RequestBodyLimit(container.Config.App.MaxRequestBodyBytes))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS())
	router.Use(middleware.Recovery(container.Logger))
	router.Use(middleware.Logging(container.Logger))

	if container.Config.App.Env != "production" {
		swaggerUI := ginSwagger.WrapHandler(
			swaggerFiles.Handler,
			ginSwagger.DocExpansion("none"),
		)
		router.GET("/swagger/*any", swaggerUI)
	}

	modulehealth.RegisterRoutes(router, modulehealth.NewHandler(container.DB, container.Redis))

	api := router.Group("/api/v1")

	authRepository := moduleauth.NewRepository(container.DB, container.Redis)
	authService := moduleauth.NewService(authRepository, container.Redis, container.JWTSigner, container.Config.App)
	actorResolver := moduleauth.NewActorResolver(authService)

	if container.Config.App.Env == "development" {
		moduledevtools.RegisterRoutes(api, moduledevtools.RouteDependencies{
			APISignature:         container.Config.App.APISignature,
			PaymentWebhookSecret: container.Config.App.PaymentWebhookSecret,
			Clock:                container.Clock,
		})
	}
	moduleauth.RegisterRoutes(api, moduleauth.RouteDependencies{
		Service:      authService,
		Logger:       container.Logger,
		Redis:        container.Redis,
		JWTSigner:    container.JWTSigner,
		APISignature: container.Config.App.APISignature,
	})
	modulecart.RegisterRoutes(api, modulecart.RouteDependencies{
		DB:           container.DB,
		Redis:        container.Redis,
		JWTSigner:    container.JWTSigner,
		ResolveActor: actorResolver,
		APISignature: container.Config.App.APISignature,
	})
	modulecatalog.RegisterRoutes(api, modulecatalog.RouteDependencies{
		DB:           container.DB,
		Redis:        container.Redis,
		JWTSigner:    container.JWTSigner,
		ResolveActor: actorResolver,
		APISignature: container.Config.App.APISignature,
	})
	modulecheckout.RegisterRoutes(api, modulecheckout.RouteDependencies{
		DB:                   container.DB,
		Redis:                container.Redis,
		JWTSigner:            container.JWTSigner,
		ResolveActor:         actorResolver,
		APISignature:         container.Config.App.APISignature,
		PaymentWebhookSecret: container.Config.App.PaymentWebhookSecret,
		Clock:                container.Clock,
	})

	return router
}
