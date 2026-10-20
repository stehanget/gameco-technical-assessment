package app

import (
	"gameco-technical-assessment/internal/config"
	moduleauth "gameco-technical-assessment/internal/modules/auth"
	moduleexample "gameco-technical-assessment/internal/modules/example"
	modulehealth "gameco-technical-assessment/internal/modules/health"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-contrib/cors"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func NewRouter(container *Container) *gin.Engine {
	if container.Config.App.Env == "production" || container.Config.App.Env == "test" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(newCORSMiddleware())
	router.Use(middleware.Recovery(container.Logger))
	router.Use(middleware.Logging(container.Logger))
	swaggerUI := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.DocExpansion("none"),
	)
	router.GET("/swagger/*any", swaggerUI)

	api := router.Group("/api/v1")
	modulehealth.RegisterRoutes(api, modulehealth.NewHandler())
	moduleauth.RegisterRoutes(api, moduleauth.RouteDependencies{
		DB:        container.DB,
		Redis:     container.Redis,
		JWTSigner: container.JWTSigner,
		AppConfig: container.Config.App,
	})
	moduleexample.RegisterRoutes(api, moduleexample.RouteDependencies{
		DB:   container.DB,
		Jobs: container.Jobs,
	})

	return router
}

func newCORSMiddleware() gin.HandlerFunc {
	return cors.New(config.CORSConfig())
}
