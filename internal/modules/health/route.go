package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRoutes, handler *Handler) {
	router.GET("/health", handler.Health)
	router.GET("/ready", handler.Ready)
}
