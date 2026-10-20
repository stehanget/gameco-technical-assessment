package config

import (
	"time"

	"github.com/gin-contrib/cors"
)

func AllowedCORSOrigins() []string {
	return []string{
		"http://localhost:3000",
		"http://localhost:3001",
	}
}

func AllowedCORSMethods() []string {
	return []string{
		"GET",
		"POST",
		"PUT",
		"PATCH",
		"DELETE",
		"OPTIONS",
	}
}

func AllowedCORSHeaders() []string {
	return []string{
		"Origin",
		"Content-Type",
		"Accept",
		"Authorization",
		"Idempotency-Key",
	}
}

func ExposedCORSHeaders() []string {
	return []string{"X-Request-ID", "Content-Disposition"}
}

func CORSConfig() cors.Config {
	return cors.Config{
		AllowOrigins:     AllowedCORSOrigins(),
		AllowMethods:     AllowedCORSMethods(),
		AllowHeaders:     AllowedCORSHeaders(),
		ExposeHeaders:    ExposedCORSHeaders(),
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}
}
