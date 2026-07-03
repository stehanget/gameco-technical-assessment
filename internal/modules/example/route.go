package example

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	group := router.Group("/examples")
	group.GET("", handler.List)
	group.POST("", handler.Create)
	group.POST("/async", handler.EnqueueCreate)
}
