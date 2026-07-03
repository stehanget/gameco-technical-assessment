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
	PublicID uuid.UUID `json:"public_id"`
	Name     string    `json:"name"`
}

type EnqueueResponse struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Queue string `json:"queue"`
	State string `json:"state"`
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
	Success bool        `json:"success"`
	Data    []Response  `json:"data"`
	Meta    *httpx.Meta `json:"meta,omitempty"`
}

type ItemResponse struct {
	Success bool        `json:"success"`
	Data    Response    `json:"data"`
	Meta    *httpx.Meta `json:"meta,omitempty"`
}

type EnqueueItemResponse struct {
	Success bool            `json:"success"`
	Data    EnqueueResponse `json:"data"`
	Meta    *httpx.Meta     `json:"meta,omitempty"`
}
