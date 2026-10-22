package checkout

import (
	"errors"
	"io"
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	platformclock "gameco-technical-assessment/internal/platform/clock"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service       *Service
	webhookSecret string
	clock         platformclock.Clock
}

func NewHandler(service *Service, webhookSecret string, timeSource platformclock.Clock) *Handler {
	if timeSource == nil {
		timeSource = platformclock.System{}
	}
	return &Handler{service: service, webhookSecret: webhookSecret, clock: timeSource}
}

// Create godoc
// @Summary Checkout active cart
// @Description Creates one pending-payment order from the authenticated buyer's active cart. Retries with the same Idempotency-Key replay the original result.
// @Tags checkout
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param Idempotency-Key header string true "Client-generated idempotency key"
// @Success 201 {object} Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/checkout [post]
func (h *Handler) Create(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		httpx.HandleError(c, faults.New(errorconst.AuthMissingContext, nil))
		return
	}
	response, err := h.service.Create(c.Request.Context(), actor, c.GetHeader("Idempotency-Key"))
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

// Detail godoc
// @Summary Get buyer order detail
// @Description Returns an order and its immutable item snapshots for the authenticated buyer. Orders owned by another buyer are not disclosed.
// @Tags orders
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param order_public_id path string true "Order public identifier" format(uuid)
// @Success 200 {object} Response
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/orders/{order_public_id} [get]
func (h *Handler) Detail(c *gin.Context) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		httpx.HandleError(c, faults.New(errorconst.AuthMissingContext, nil))
		return
	}
	response, err := h.service.Detail(c.Request.Context(), actor, c.Param("order_public_id"))
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

// PaymentWebhook godoc
// @Summary Receive payment-provider webhook
// @Description Verifies the simulated provider HMAC signature and freshness window, then atomically applies a payment event with durable replay protection.
// @Tags payment-webhooks
// @Accept json
// @Produce json
// @Param X-Payment-Timestamp header string true "Unix timestamp used in the signed message"
// @Param X-Payment-Signature header string true "Hex HMAC-SHA256 signature"
// @Success 204 "Webhook accepted and processed or safely replayed"
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/webhooks/payment [post]
func (h *Handler) PaymentWebhook(c *gin.Context) {
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, maxPaymentWebhookBodyBytes+1))
	if err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	if len(body) > maxPaymentWebhookBodyBytes {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookMalformedPayload, nil))
		return
	}
	verifyErr := verifyWebhookSignature(
		h.webhookSecret,
		c.GetHeader("X-Payment-Timestamp"),
		c.GetHeader("X-Payment-Signature"),
		body,
		h.clock.Now(),
	)
	if errors.Is(verifyErr, errWebhookTimestamp) {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookTimestampInvalid, verifyErr))
		return
	}
	if errors.Is(verifyErr, errWebhookSignature) {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookSignatureInvalid, verifyErr))
		return
	}
	request, err := decodePaymentWebhook(body)
	if errors.Is(err, errWebhookMalformedPayload) {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookMalformedPayload, err))
		return
	}
	if errors.Is(err, errWebhookInvalidEvent) {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookInvalidEvent, err))
		return
	}
	if err != nil {
		httpx.HandleError(c, faults.New(errorconst.PaymentWebhookMalformedPayload, err))
		return
	}
	if err := h.service.ProcessWebhook(c.Request.Context(), request, body, h.clock.Now()); err != nil {
		httpx.HandleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
