package checkout

import (
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// Create godoc
// @Summary Checkout active cart
// @Description Creates one pending-payment order from the authenticated buyer's active cart. Retries with the same Idempotency-Key replay the original result.
// @Tags checkout
// @Produce json
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
