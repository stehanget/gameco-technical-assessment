package health

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRoutes, handler *Handler) {
	router.GET("/health", handler.Health)
	router.GET("/livez", handler.Live)
	router.GET("/readyz", handler.Ready)
}
