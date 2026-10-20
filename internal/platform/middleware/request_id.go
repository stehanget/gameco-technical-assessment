package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDKey = "request_id"

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if !validRequestID(requestID) {
			requestID = uuid.NewString()
		}

		c.Set(RequestIDKey, requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

func RequestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get(RequestIDKey)
	value, _ := requestID.(string)
	return value
}

func validRequestID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && strings.ToLower(value) == parsed.String()
}
