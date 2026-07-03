package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func Logging(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		logger.Info(
			"http request",
			"request_id", RequestIDFromContext(c),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency", time.Since(startedAt).String(),
			"client_ip", c.ClientIP(),
		)
	}
}
