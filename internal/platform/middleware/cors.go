package middleware

import (
	"gameco-technical-assessment/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
	return cors.New(config.CORSConfig())
}
