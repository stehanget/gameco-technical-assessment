package auth

import (
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"
	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/platform/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register godoc
// @Summary Register user
// @Description Creates a user and returns an access token plus refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Register payload"
// @Success 201 {object} TokenEnvelope
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err))
		return
	}

	response, err := h.service.Register(c.Request.Context(), request, SessionMetadata{UserAgent: c.GetHeader("User-Agent"), IPAddress: c.ClientIP()})
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusCreated, response)
}

// Login godoc
// @Summary Login
// @Description Authenticates a user and returns an access token plus refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login payload"
// @Success 200 {object} TokenEnvelope
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err))
		return
	}

	response, err := h.service.Login(c.Request.Context(), request, SessionMetadata{UserAgent: c.GetHeader("User-Agent"), IPAddress: c.ClientIP()})
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusOK, response)
}

// Refresh godoc
// @Summary Refresh tokens
// @Description Exchanges a refresh token for a new access token and refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh payload"
// @Success 200 {object} TokenEnvelope
// @Failure 400 {object} httpx.ErrorResponse
// @Failure 401 {object} httpx.ErrorResponse
// @Router /api/v1/auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	var request RefreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err))
		return
	}

	response, err := h.service.Refresh(c.Request.Context(), request.RefreshToken, SessionMetadata{UserAgent: c.GetHeader("User-Agent"), IPAddress: c.ClientIP()})
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusOK, response)
}

// Logout godoc
// @Summary Logout
// @Description Revokes a refresh token session
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LogoutRequest true "Logout payload"
// @Success 200 {object} MessageEnvelope
// @Failure 400 {object} httpx.ErrorResponse
// @Router /api/v1/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var request LogoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		httpx.HandleError(c, faults.New(faults.KindInvalidInput, errorconst.InvalidRequest, err))
		return
	}

	if err := h.service.Logout(c.Request.Context(), request.RefreshToken); err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusOK, MessageResponse{Message: "logged out"})
}

// Me godoc
// @Summary Current user
// @Description Returns the authenticated user profile
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserEnvelope
// @Failure 401 {object} httpx.ErrorResponse
// @Failure 404 {object} httpx.ErrorResponse
// @Router /api/v1/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	access, ok := middleware.ActorFromContext(c)
	if !ok {
		httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthMissingContext, nil))
		return
	}
	if access.UserID == 0 {
		httpx.HandleError(c, faults.New(faults.KindUnauthenticated, errorconst.AuthInvalidContext, nil))
		return
	}

	response, err := h.service.Me(c.Request.Context(), access.UserID)
	if err != nil {
		httpx.HandleError(c, err)
		return
	}

	httpx.Success(c, http.StatusOK, response)
}
