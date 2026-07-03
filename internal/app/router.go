package app

import (
	"context"

	moduleauth "gameco-technical-assessment/internal/modules/auth"
	"gameco-technical-assessment/internal/modules/example"
	"gameco-technical-assessment/internal/modules/health"
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
	router.Use(middleware.RequestID())
	router.Use(middleware.Recovery(container.Logger))
	router.Use(middleware.Logging(container.Logger))
	swaggerUI := ginSwagger.WrapHandler(
		swaggerFiles.Handler,
		ginSwagger.DocExpansion("none"),
	)
	router.GET("/swagger/*any", swaggerUI)

	api := router.Group("/api/v1")
	health.RegisterRoutes(api, health.NewHandler())
	registerAuthRoutes(api, container)
	registerExampleRoutes(api, container)

	return router
}

func registerAuthRoutes(api *gin.RouterGroup, container *Container) {
	authRepository := moduleauth.NewRepository(container.DB)
	service := moduleauth.NewService(authRepository, container.Redis, container.JWTSigner, container.Config.App)
	handler := moduleauth.NewHandler(service)
	moduleauth.RegisterRoutes(api, handler, container.JWTSigner, func(ctx context.Context, jti string) ([]string, []string, error) {
		access, err := service.ValidateSession(ctx, jti)
		if err != nil {
			return nil, nil, err
		}
		return access.Roles, access.Permissions, nil
	})
}

func registerExampleRoutes(api *gin.RouterGroup, container *Container) {
	repository := example.NewRepository(container.DB)
	service := example.NewService(repository, container.Jobs)
	handler := example.NewHandler(service)
	example.RegisterRoutes(api, handler)
}
