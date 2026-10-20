package catalog

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RouteDependencies struct {
	DB *gorm.DB
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository)
	handler := NewHandler(service)

	group := router.Group("/products")
	group.GET("", handler.List)
	group.GET("/:product_public_id", handler.Detail)
}
