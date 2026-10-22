package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHandler(db *gorm.DB, redisClient *redis.Client) *Handler {
	return &Handler{db: db, redis: redisClient}
}

// Live godoc
// @Summary Liveness check
// @Description Returns 200 when the API process is alive. It does not probe dependencies.
// @Tags health
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /livez [get]
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, StatusResponse{Status: "ok"})
}

// Ready godoc
// @Summary Readiness check
// @Description Probes PostgreSQL and Redis. Returns 503 while either dependency is unavailable.
// @Tags health
// @Produce json
// @Success 200 {object} StatusResponse
// @Failure 503 {object} StatusResponse
// @Router /readyz [get]
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if h.db == nil || h.redis == nil {
		c.JSON(http.StatusServiceUnavailable, StatusResponse{Status: "not_ready"})
		return
	}
	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil || h.redis.Ping(ctx).Err() != nil {
		c.JSON(http.StatusServiceUnavailable, StatusResponse{Status: "not_ready"})
		return
	}
	c.JSON(http.StatusOK, StatusResponse{Status: "ok"})
}

// Health godoc
// @Summary Health check
// @Description Backward-compatible alias for the liveness endpoint.
// @Tags health
// @Produce json
// @Success 200 {object} StatusResponse
// @Router /health [get]
func (h *Handler) Health(c *gin.Context) {
	h.Live(c)
}
