package httpx

import "github.com/gin-gonic/gin"

type SuccessResponse struct {
	Success bool  `json:"success"`
	Data    any   `json:"data"`
	Meta    *Meta `json:"meta,omitempty"`
}

type ErrorResponse struct {
	Success   bool   `json:"success"`
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

func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, ErrorResponse{
		Success:   false,
		Message:   message,
		RequestID: requestIDFromContext(c),
	})
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
