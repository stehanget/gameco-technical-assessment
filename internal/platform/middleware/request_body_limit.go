package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequestBodyLimit caps request bodies before handlers or JSON decoders consume them.
func RequestBodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if maxBytes > 0 {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}
