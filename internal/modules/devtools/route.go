package devtools

import (
	"gameco-technical-assessment/internal/config"
	platformclock "gameco-technical-assessment/internal/platform/clock"

	"github.com/gin-gonic/gin"
)

type RouteDependencies struct {
	APISignature         config.APISignatureConfig
	PaymentWebhookSecret string
	Clock                platformclock.Clock
}

func RegisterRoutes(router *gin.RouterGroup, deps RouteDependencies) {
	handler := NewHandler(deps)
	group := router.Group("/dev/signatures")
	group.POST("/api", handler.APISignature)
	group.POST("/webhook", handler.WebhookSignature)
}
