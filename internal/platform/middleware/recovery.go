package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

	"github.com/gin-gonic/gin"
)

func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
			slog.String("request_id", RequestIDFromContext(c)),
			slog.String("method", c.Request.Method),
			slog.String("route", c.FullPath()),
			slog.String("panic", redactDiagnostic(fmt.Sprint(recovered))),
			slog.String("stack", redactDiagnostic(string(debug.Stack()))),
		)
		httpx.HandleError(c, faults.New(faults.KindInternal, errorconst.InternalServerError, nil))
	})
}
