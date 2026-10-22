package cart

import (
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// Get godoc
// @Summary Get active cart
// @Description Returns the authenticated buyer's active cart and current product prices.
// @Tags cart
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Success 200 {object} Response
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/cart [get]
func (h *Handler) Get(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	response, err := h.service.Get(c.Request.Context(), actor)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

// SetItem godoc
// @Summary Set cart item quantity
// @Description Adds a product to the active cart or replaces its quantity.
// @Tags cart
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param product_public_id path string true "Product public identifier" format(uuid)
// @Param request body SetItemRequest true "Cart item quantity"
// @Success 200 {object} Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 409 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/cart/items/{product_public_id} [put]
func (h *Handler) SetItem(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	var request SetItemRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	response, err := h.service.SetItem(c.Request.Context(), actor, c.Param("product_public_id"), request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

// DeleteItem godoc
// @Summary Remove cart item
// @Description Removes a product from the authenticated buyer's active cart.
// @Tags cart
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param product_public_id path string true "Product public identifier" format(uuid)
// @Success 200 {object} Response
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/cart/items/{product_public_id} [delete]
func (h *Handler) DeleteItem(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	response, err := h.service.DeleteItem(c.Request.Context(), actor, c.Param("product_public_id"))
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

func actorFromContext(c *gin.Context) (platformauth.Actor, error) {
	actor, ok := middleware.ActorFromContext(c)
	if !ok {
		return platformauth.Actor{}, faults.New(errorconst.AuthMissingContext, nil)
	}
	return actor, nil
}
