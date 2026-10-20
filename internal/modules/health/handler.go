package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

// Health godoc
// @Summary Health check
// @Description Returns liveness status
// @Tags health
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, StatusResponse{Status: "ok"})
}
