package example

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gameco-technical-assessment/internal/workers/jobs"
)

type RouteDependencies struct {
	DB   *gorm.DB
	Jobs *jobs.Enqueuer
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	repository := NewRepository(deps.DB)
	service := NewService(repository, deps.Jobs)
	handler := NewHandler(service)

	group := router.Group("/examples")
	group.GET("", handler.List)
	group.POST("", handler.Create)
	group.POST("/async", handler.EnqueueCreate)
}
