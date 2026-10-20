// Package main contains the Gameco HTTP API entry point.
//
// Swagger metadata and tag descriptions are centralized here so main.go stays
// focused on application startup. Endpoint-level annotations remain beside
// their handlers.
//
// @title gameco-technical-assessment API
// @version 1.0
// @description Gameco marketplace backend technical assessment API with Gin, GORM, Goose, JWT + Redis, and Swagger.
// @BasePath /
// @schemes http https
// @tag.name auth
// @tag.description Authentication, session lifecycle, and current-user authorization context.
// @tag.name examples
// @tag.description Development-only example endpoints used to validate the backend module foundation.
// @tag.name health
// @tag.description Liveness endpoint for runtime health checks.
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main
