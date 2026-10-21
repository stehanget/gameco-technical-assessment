package httpx

import (
	"errors"
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool  `json:"success" binding:"required"`
	Data    any   `json:"data" binding:"required"`
	Meta    *Meta `json:"meta" binding:"required"`
}

type ErrorResponse struct {
	Success   bool    `json:"success" binding:"required"`
	Code      string  `json:"code" binding:"required"`
	Message   string  `json:"message" binding:"required"`
	RequestID *string `json:"request_id" binding:"required" extensions:"x-nullable"`
}

type Meta struct {
	RequestID *string `json:"request_id" binding:"required" extensions:"x-nullable"`
}

func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    requestMeta(c),
	})
}

func SuccessWithMeta(c *gin.Context, statusCode int, data any, meta any) {
	c.JSON(statusCode, struct {
		Success bool `json:"success"`
		Data    any  `json:"data"`
		Meta    any  `json:"meta"`
	}{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func HandleError(c *gin.Context, err error) {
	var appErr *faults.Error
	if !errors.As(err, &appErr) {
		if err != nil {
			_ = c.Error(err)
		}
		code, message := errorContract(http.StatusInternalServerError)
		writeErrorResponse(c, http.StatusInternalServerError, code, message)
		return
	}

	statusCode, fallback := errorContractForKind(appErr.Kind)
	code := appErr.Code
	if code == "" {
		code = fallback.Code
	}
	message := appErr.Message
	if message == "" {
		message = fallback.Message
	}
	if err != nil {
		_ = c.Error(err)
	}
	writeErrorResponse(c, statusCode, code, message)
}

func writeErrorResponse(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success:   false,
		Code:      code,
		Message:   message,
		RequestID: requestIDPointer(c),
	})
}

func errorContract(statusCode int) (string, string) {
	definition := definitionForStatus(statusCode)
	return definition.Code, definition.Message
}

func errorContractForKind(kind faults.Kind) (int, faults.Definition) {
	switch kind {
	case faults.KindInvalidInput:
		return http.StatusBadRequest, errorconst.InvalidRequest
	case faults.KindUnauthenticated:
		return http.StatusUnauthorized, errorconst.Unauthorized
	case faults.KindForbidden:
		return http.StatusForbidden, errorconst.Forbidden
	case faults.KindNotFound:
		return http.StatusNotFound, errorconst.NotFound
	case faults.KindConflict:
		return http.StatusConflict, errorconst.Conflict
	case faults.KindRateLimit:
		return http.StatusTooManyRequests, errorconst.RateLimitExceeded
	case faults.KindUnavailable:
		return http.StatusServiceUnavailable, errorconst.ServiceUnavailable
	default:
		return http.StatusInternalServerError, errorconst.InternalServerError
	}
}

func definitionForStatus(statusCode int) faults.Definition {
	switch statusCode {
	case http.StatusBadRequest:
		return errorconst.InvalidRequest
	case http.StatusUnauthorized:
		return errorconst.Unauthorized
	case http.StatusForbidden:
		return errorconst.Forbidden
	case http.StatusNotFound:
		return errorconst.NotFound
	case http.StatusConflict:
		return errorconst.Conflict
	case http.StatusTooManyRequests:
		return errorconst.RateLimitExceeded
	case http.StatusServiceUnavailable:
		return errorconst.ServiceUnavailable
	default:
		return errorconst.InternalServerError
	}
}

func requestMeta(c *gin.Context) *Meta {
	requestID := requestIDFromContext(c)
	if requestID == "" {
		return nil
	}

	return &Meta{RequestID: requestIDPointer(c)}
}

func requestIDPointer(c *gin.Context) *string {
	requestID := requestIDFromContext(c)
	if requestID == "" {
		return nil
	}
	return &requestID
}

func requestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get("request_id")
	value, _ := requestID.(string)
	return value
}
