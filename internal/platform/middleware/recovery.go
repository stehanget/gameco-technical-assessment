package middleware

import (
	"log/slog"
	"net/http"

	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "request_id", RequestIDFromContext(c), "error", recovered)
		httpx.Error(c, http.StatusInternalServerError, "internal server error")
	})
}
