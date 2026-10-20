package httpx

import (
	"errors"
	"net/http"

	errorconst "gameco-technical-assessment/internal/constants/error"
	"gameco-technical-assessment/internal/platform/faults"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Success bool  `json:"success"`
	Data    any   `json:"data"`
	Meta    *Meta `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
}

func Success(c *gin.Context, statusCode int, data any) {
	c.JSON(statusCode, SuccessResponse{
		Success: true,
		Data:    data,
		Meta:    requestMeta(c),
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

	statusCode, fallbackCode, fallbackMessage := errorContractForKind(appErr.Kind)
	code := appErr.Code
	if code == "" {
		code = fallbackCode
	}
	message := messageForCode(code)
	if message == "" {
		message = fallbackMessage
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
		RequestID: requestIDFromContext(c),
	})
}

func errorContract(statusCode int) (string, string) {
	switch statusCode {
	case http.StatusBadRequest:
		return errorconst.InvalidRequest, errorconst.InvalidRequestMessage
	case http.StatusUnauthorized:
		return errorconst.Unauthorized, errorconst.UnauthorizedMessage
	case http.StatusForbidden:
		return errorconst.Forbidden, errorconst.ForbiddenMessage
	case http.StatusNotFound:
		return errorconst.NotFound, errorconst.NotFoundMessage
	case http.StatusConflict:
		return errorconst.Conflict, errorconst.ConflictMessage
	case http.StatusTooManyRequests:
		return errorconst.RateLimitExceeded, errorconst.RateLimitExceededMessage
	case http.StatusServiceUnavailable:
		return errorconst.ServiceUnavailable, errorconst.ServiceUnavailableMessage
	default:
		return errorconst.InternalServerError, errorconst.InternalServerErrorMessage
	}
}

func errorContractForKind(kind faults.Kind) (int, string, string) {
	switch kind {
	case faults.KindInvalidInput:
		return http.StatusBadRequest, errorconst.InvalidRequest, errorconst.InvalidRequestMessage
	case faults.KindUnauthenticated:
		return http.StatusUnauthorized, errorconst.Unauthorized, errorconst.UnauthorizedMessage
	case faults.KindForbidden:
		return http.StatusForbidden, errorconst.Forbidden, errorconst.ForbiddenMessage
	case faults.KindNotFound:
		return http.StatusNotFound, errorconst.NotFound, errorconst.NotFoundMessage
	case faults.KindConflict:
		return http.StatusConflict, errorconst.Conflict, errorconst.ConflictMessage
	case faults.KindRateLimit:
		return http.StatusTooManyRequests, errorconst.RateLimitExceeded, errorconst.RateLimitExceededMessage
	case faults.KindUnavailable:
		return http.StatusServiceUnavailable, errorconst.ServiceUnavailable, errorconst.ServiceUnavailableMessage
	default:
		return http.StatusInternalServerError, errorconst.InternalServerError, errorconst.InternalServerErrorMessage
	}
}

func messageForCode(code string) string {
	switch code {
	case errorconst.AuthEmailAlreadyRegistered:
		return errorconst.AuthEmailAlreadyRegisteredMessage
	case errorconst.AuthInvalidCredentials:
		return errorconst.AuthInvalidCredentialsMessage
	case errorconst.AuthRefreshTokenReused:
		return errorconst.AuthRefreshTokenReusedMessage
	case errorconst.AuthUserNotFound:
		return errorconst.AuthUserNotFoundMessage
	case errorconst.AuthNoAssignedRoles:
		return errorconst.AuthNoAssignedRolesMessage
	case errorconst.AuthMissingAuthorization:
		return errorconst.AuthMissingAuthorizationMessage
	case errorconst.AuthInvalidAuthorization:
		return errorconst.AuthInvalidAuthorizationMessage
	case errorconst.AuthInvalidToken:
		return errorconst.AuthInvalidTokenMessage
	case errorconst.AuthSessionInvalid:
		return errorconst.AuthSessionInvalidMessage
	case errorconst.AuthMissingContext:
		return errorconst.AuthMissingContextMessage
	case errorconst.AuthInvalidContext:
		return errorconst.AuthInvalidContextMessage
	case errorconst.AuthInsufficientPermissions:
		return errorconst.AuthInsufficientPermissionsMessage
	default:
		return ""
	}
}

func requestMeta(c *gin.Context) *Meta {
	requestID := requestIDFromContext(c)
	if requestID == "" {
		return nil
	}

	return &Meta{RequestID: requestID}
}

func requestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get("request_id")
	value, _ := requestID.(string)
	return value
}
