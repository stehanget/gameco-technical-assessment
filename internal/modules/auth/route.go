package auth

import (
	"gameco-technical-assessment/internal/modules/auth/permission"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.RouterGroup, handler *Handler, jwtManager *platformauth.JWTManager, resolve middleware.SessionResolver) {
	group := router.Group("/auth")
	group.POST("/register", handler.Register)
	group.POST("/login", handler.Login)
	group.POST("/refresh", handler.Refresh)
	group.POST("/logout", handler.Logout)
	group.GET("/me", middleware.Auth(jwtManager, resolve), middleware.RequirePermissions(permission.AuthMe), handler.Me)
}
