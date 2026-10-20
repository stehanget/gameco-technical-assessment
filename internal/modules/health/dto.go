package health

type StatusResponse struct {
	Status string `json:"status" binding:"required"`
}
