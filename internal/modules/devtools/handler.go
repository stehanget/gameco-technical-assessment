package devtools

import (
	"net/http"
	"strconv"
	"strings"

	"gameco-technical-assessment/internal/config"
	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/modules/checkout"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	apiSignature         config.APISignatureConfig
	paymentWebhookSecret string
	clock                platformclock.Clock
}

func NewHandler(deps RouteDependencies) *Handler {
	clock := deps.Clock
	if clock == nil {
		clock = platformclock.System{}
	}
	return &Handler{apiSignature: deps.APISignature, paymentWebhookSecret: deps.PaymentWebhookSecret, clock: clock}
}

// APISignature godoc
// @Summary Generate an API request signature for development
// @Description Development-only helper. Generates request-signing headers without returning the configured secret.
// @Tags development-tools
// @Accept json
// @Produce json
// @Param request body APISignatureRequest true "API signature input; body must be the exact wire body"
// @Success 200 {object} Response
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/dev/signatures/api [post]
func (h *Handler) APISignature(c *gin.Context) {
	var request APISignatureRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	path := strings.TrimSpace(request.Path)
	if method == "" || path == "" || !strings.HasPrefix(path, "/") {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, nil))
		return
	}
	timestamp := h.clock.Now().Unix()
	if request.Timestamp != nil {
		timestamp = *request.Timestamp
	}
	nonce := strings.TrimSpace(request.Nonce)
	if nonce == "" {
		nonce = uuid.NewString()
	}
	timestampValue := strconv.FormatInt(timestamp, 10)
	signature, err := middleware.SignAPISignature(h.apiSignature, request.KeyID, timestampValue, nonce, method, path, []byte(request.Body))
	if err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	httpx.Success(c, http.StatusOK, SignatureResponse{
		Type: "api", Timestamp: timestamp, Nonce: nonce, Body: request.Body, Signature: signature,
		Headers: map[string]string{"X-API-Key": request.KeyID, "X-API-Timestamp": timestampValue, "X-API-Nonce": nonce, "X-API-Signature": signature},
	})
}

// WebhookSignature godoc
// @Summary Generate a payment webhook signature for development
// @Description Development-only helper. Generates payment webhook headers without returning the configured webhook secret.
// @Tags development-tools
// @Accept json
// @Produce json
// @Param request body WebhookSignatureRequest true "Webhook signature input; body must be the exact wire body"
// @Success 200 {object} Response
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/dev/signatures/webhook [post]
func (h *Handler) WebhookSignature(c *gin.Context) {
	var request WebhookSignatureRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	timestamp := h.clock.Now().Unix()
	if request.Timestamp != nil {
		timestamp = *request.Timestamp
	}
	timestampValue := strconv.FormatInt(timestamp, 10)
	signature, err := checkout.SignWebhookSignature(h.paymentWebhookSecret, timestampValue, []byte(request.Body))
	if err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	httpx.Success(c, http.StatusOK, SignatureResponse{
		Type: "webhook", Timestamp: timestamp, Body: request.Body, Signature: signature,
		Headers: map[string]string{"X-Payment-Timestamp": timestampValue, "X-Payment-Signature": signature},
	})
}
