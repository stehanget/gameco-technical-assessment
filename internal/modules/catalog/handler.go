package catalog

import (
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"

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
		httpx.HandleError(c, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err))
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
