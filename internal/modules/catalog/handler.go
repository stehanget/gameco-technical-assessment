package catalog

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

// List godoc
// @Summary List public products
// @Description Lists active, non-deleted products with filtering, search, sorting, and cursor pagination
// @Tags products
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Param q query string false "Case-insensitive search over product name and description"
// @Param category query string false "Category slug"
// @Param min_price query int false "Minimum price amount"
// @Param max_price query int false "Maximum price amount"
// @Param sort_by query string false "Sort field" Enums(created_at,name,price) default(created_at)
// @Param sort_order query string false "Sort direction" Enums(asc,desc) default(desc)
// @Param cursor query string false "Opaque cursor returned by the previous response"
// @Param per_page query int false "Number of records per page" minimum(1) maximum(100) default(20)
// @Success 200 {object} ListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/products [get]
func (h *Handler) List(c *gin.Context) {
	var request ListQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}

	result, err := h.service.List(c.Request.Context(), request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	requestID, _ := c.Get("request_id")
	requestIDValue, _ := requestID.(string)
	var requestIDPointer *string
	if requestIDValue != "" {
		requestIDPointer = &requestIDValue
	}
	httpx.SuccessWithMeta(c, http.StatusOK, result.Items, ListMeta{
		RequestID:  requestIDPointer,
		PerPage:    result.PerPage,
		HasMore:    result.HasMore,
		NextCursor: result.NextCursor,
	})
}

// Detail godoc
// @Summary Get public product
// @Description Returns one active, non-deleted product by its UUIDv7 public identifier
// @Tags products
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Param product_public_id path string true "Product public identifier" format(uuid)
// @Success 200 {object} ItemResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/products/{product_public_id} [get]
func (h *Handler) Detail(c *gin.Context) {
	response, err := h.service.FindByPublicID(c.Request.Context(), c.Param("product_public_id"))
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

// CategoryOptions godoc
// @Summary List category options
// @Description Lists canonical categories for public catalog filters and seller product forms
// @Tags categories
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Success 200 {object} CategoryOptionsResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/categories/options [get]
func (h *Handler) CategoryOptions(c *gin.Context) {
	options, err := h.service.CategoryOptions(c.Request.Context())
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, options)
}

// SellerList godoc
// @Summary List seller products
// @Description Lists non-deleted products owned by the authenticated seller
// @Tags seller-products
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param q query string false "Case-insensitive search over product name and description"
// @Param category query string false "Category slug"
// @Param min_price query int false "Minimum price amount"
// @Param max_price query int false "Maximum price amount"
// @Param sort_by query string false "Sort field" Enums(created_at,name,price) default(created_at)
// @Param sort_order query string false "Sort direction" Enums(asc,desc) default(desc)
// @Param cursor query string false "Opaque cursor returned by the previous response"
// @Param per_page query int false "Number of records per page" minimum(1) maximum(100) default(20)
// @Success 200 {object} SellerListResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/seller/products [get]
func (h *Handler) SellerList(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	var request ListQuery
	if err := c.ShouldBindQuery(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	result, err := h.service.ListBySeller(c.Request.Context(), actor, request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	requestID, _ := c.Get(middleware.RequestIDKey)
	requestIDValue, _ := requestID.(string)
	var requestIDPointer *string
	if requestIDValue != "" {
		requestIDPointer = &requestIDValue
	}
	httpx.SuccessWithMeta(c, http.StatusOK, result.Items, ListMeta{
		RequestID:  requestIDPointer,
		PerPage:    result.PerPage,
		HasMore:    result.HasMore,
		NextCursor: result.NextCursor,
	})
}

// SellerDetail godoc
// @Summary Get seller product
// @Description Returns one non-deleted product owned by the authenticated seller
// @Tags seller-products
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param product_public_id path string true "Product public identifier" format(uuid)
// @Success 200 {object} SellerItemResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/seller/products/{product_public_id} [get]
func (h *Handler) SellerDetail(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	response, err := h.service.FindSellerProductByPublicID(c.Request.Context(), actor, c.Param("product_public_id"))
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, response)
}

// Create godoc
// @Summary Create seller product
// @Tags seller-products
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param request body CreateProductRequest true "Product payload"
// @Success 201 {object} SellerItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/seller/products [post]
func (h *Handler) Create(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	var request CreateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	response, err := h.service.CreateProduct(c.Request.Context(), actor, request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, response)
}

// Update godoc
// @Summary Update seller product
// @Tags seller-products
// @Accept json
// @Produce json
// @Param X-API-Key header string false "Trusted integration key ID"
// @Param X-API-Timestamp header string false "Unix timestamp in seconds"
// @Param X-API-Nonce header string false "Unique request nonce"
// @Param X-API-Signature header string false "Versioned HMAC signature, for example v1=<hex>"
// @Security BearerAuth
// @Param product_public_id path string true "Product public identifier" format(uuid)
// @Param request body UpdateProductRequest true "Product fields"
// @Success 200 {object} SellerItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 403 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/seller/products/{product_public_id} [patch]
func (h *Handler) Update(c *gin.Context) {
	actor, err := actorFromContext(c)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}
	var request UpdateProductRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}
	response, err := h.service.UpdateProduct(c.Request.Context(), actor, c.Param("product_public_id"), request)
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
	if actor.UserID == 0 {
		return platformauth.Actor{}, faults.New(errorconst.AuthInvalidContext, nil)
	}
	return actor, nil
}
