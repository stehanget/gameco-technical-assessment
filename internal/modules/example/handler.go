package example

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

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// List godoc
// @Summary List examples
// @Description Returns example records from the boilerplate module
// @Tags examples
// @Produce json
// @Success 200 {object} ListResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/examples [get]
func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List()
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusOK, items)
}

// Create godoc
// @Summary Create example
// @Description Creates a sample resource to demonstrate the module pattern
// @Tags examples
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Example payload"
// @Success 201 {object} ItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/examples [post]
func (h *Handler) Create(c *gin.Context) {
	var request CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}

	item, err := h.service.Create(request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusCreated, item)
}

// EnqueueCreate godoc
// @Summary Enqueue example creation
// @Description Pushes example creation into the Redis-backed worker queue
// @Tags examples
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Example payload"
// @Success 202 {object} EnqueueItemResponse
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 500 {object} httpx.ErrorResponse
// @Router /api/v1/examples/async [post]
func (h *Handler) EnqueueCreate(c *gin.Context) {
	var request CreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(errorconst.InvalidRequest, err))
		return
	}

	result, err := h.service.EnqueueCreate(c.Request.Context(), request)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusAccepted, mapEnqueueResult(result))
}
