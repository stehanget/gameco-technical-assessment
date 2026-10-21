package cart

import (
	"gameco-technical-assessment/internal/constants/role"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RouteDependencies struct {
	DB           *gorm.DB
	JWTSigner    *platformauth.JWTManager
	ResolveActor middleware.ActorResolver
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository)
	handler := NewHandler(service)

	group := router.Group("/cart")
	group.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	group.Use(middleware.RequireRoles(role.Buyer))
	group.GET("", handler.Get)
	group.PUT("/items/:product_public_id", handler.SetItem)
	group.DELETE("/items/:product_public_id", handler.DeleteItem)
}
