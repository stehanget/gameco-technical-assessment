package catalog

import (
	"gameco-technical-assessment/internal/constants/permission"
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

	group := router.Group("/products")
	group.GET("", handler.List)
	group.GET("/:product_public_id", handler.Detail)

	router.GET("/categories/options", handler.CategoryOptions)

	seller := router.Group("/seller/products")
	seller.Use(middleware.Auth(deps.JWTSigner, deps.ResolveActor))
	seller.Use(middleware.RequireRoles(role.Seller))
	seller.Use(middleware.RequirePermissions(permission.ProductRead))
	seller.GET("", handler.SellerList)
	seller.GET("/:product_public_id", handler.SellerDetail)
	seller.POST("", middleware.RequirePermissions(permission.ProductCreate), handler.Create)
	seller.PATCH("/:product_public_id", middleware.RequirePermissions(permission.ProductUpdate), handler.Update)
}
