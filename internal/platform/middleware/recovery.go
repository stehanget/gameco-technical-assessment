package middleware

import (
	"log/slog"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "request_id", RequestIDFromContext(c), "error", recovered)
		httpx.HandleError(c, faults.New(faults.KindInternal, errorconst.InternalServerError, nil))
	})
}
