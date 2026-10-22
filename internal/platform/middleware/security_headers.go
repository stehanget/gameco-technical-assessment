package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders adds browser-facing response protections that are safe for the API.
// Content-specific output encoding remains the responsibility of the rendering client.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		c.Next()
	}
}
