package middleware

import (
	"errors"
	"log/slog"
	"reflect"
	"regexp"
	"time"

	"gameco-technical-assessment/internal/platform/faults"

	"github.com/gin-gonic/gin"
)

var sensitiveDiagnosticPattern = regexp.MustCompile(`(?i)(password|passwd|secret|token|authorization|api[_-]?key|client[_-]?secret|refresh[_-]?token)([=:])([^\s,;]+)`)

func Logging(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		status := c.Writer.Status()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}

		attrs := []slog.Attr{
			slog.String("request_id", RequestIDFromContext(c)),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Float64("duration_ms", float64(time.Since(startedAt).Microseconds())/1000),
			slog.String("client_ip", c.ClientIP()),
		}

		if errorCount := len(c.Errors); errorCount > 0 {
			attrs = append(attrs, slog.Int("error_count", errorCount))
			appendErrorAttrs(&attrs, status, c.Errors.Last().Err)
		}

		message := "http request"
		if status >= 500 {
			message = "http request failed"
		}
		logger.LogAttrs(c.Request.Context(), requestLogLevel(status), message, attrs...)
	}
}

func requestLogLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func appendErrorAttrs(attrs *[]slog.Attr, status int, err error) {
	var fault *faults.Error
	if errors.As(err, &fault) {
		if fault.Code != "" {
			*attrs = append(*attrs, slog.String("error_code", fault.Code))
		}
		if fault.Kind != "" {
			*attrs = append(*attrs, slog.String("error_kind", string(fault.Kind)))
		}
		if status >= 500 && fault.Cause != nil {
			*attrs = append(*attrs, slog.String("error", redactDiagnostic(fault.Cause.Error())))
		}
		return
	}

	if err == nil {
		return
	}
	*attrs = append(*attrs, slog.String("error_type", reflect.TypeOf(err).String()))
	if status >= 500 {
		*attrs = append(*attrs, slog.String("error", redactDiagnostic(err.Error())))
	}
}

func redactDiagnostic(value string) string {
	return sensitiveDiagnosticPattern.ReplaceAllString(value, `${1}${2}[REDACTED]`)
}
