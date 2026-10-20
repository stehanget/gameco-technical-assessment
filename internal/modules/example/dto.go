package example

import (
	"github.com/google/uuid"

	"gameco-technical-assessment/internal/platform/httpx"
	"gameco-technical-assessment/internal/workers/jobs"
)

type CreateRequest struct {
	Name string `json:"name" binding:"required,min=2,max=255"`
}

type Response struct {
	PublicID uuid.UUID `json:"public_id" binding:"required"`
	Name     string    `json:"name" binding:"required"`
}

type EnqueueResponse struct {
	ID    string `json:"id" binding:"required"`
	Type  string `json:"type" binding:"required"`
	Queue string `json:"queue" binding:"required"`
	State string `json:"state" binding:"required"`
}

func mapEnqueueResult(result jobs.EnqueueResult) EnqueueResponse {
	return EnqueueResponse{
		ID:    result.ID,
		Type:  result.Type,
		Queue: result.Queue,
		State: result.State,
	}
}

type ListResponse struct {
	Success bool        `json:"success" binding:"required"`
	Data    []Response  `json:"data" binding:"required"`
	Meta    *httpx.Meta `json:"meta" binding:"required"`
}

type ItemResponse struct {
	Success bool        `json:"success" binding:"required"`
	Data    Response    `json:"data" binding:"required"`
	Meta    *httpx.Meta `json:"meta" binding:"required"`
}

type EnqueueItemResponse struct {
	Success bool            `json:"success" binding:"required"`
	Data    EnqueueResponse `json:"data" binding:"required"`
	Meta    *httpx.Meta     `json:"meta" binding:"required"`
}
