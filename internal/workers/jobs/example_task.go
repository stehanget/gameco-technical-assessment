package jobs

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hibiken/asynq"
)

const TypeExampleCreate = "example:create"

type ExampleCreatePayload struct {
	Name string `json:"name"`
}

func NewExampleCreateTask(name string) (*asynq.Task, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("example name is required")
	}

	payload, err := json.Marshal(ExampleCreatePayload{Name: trimmed})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeExampleCreate, payload, asynq.Queue(defaultQueueName)), nil
}

func ParseExampleCreateTask(task *asynq.Task) (ExampleCreatePayload, error) {
	var payload ExampleCreatePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return ExampleCreatePayload{}, err
	}
	payload.Name = strings.TrimSpace(payload.Name)
	if payload.Name == "" {
		return ExampleCreatePayload{}, fmt.Errorf("example name is required")
	}
	return payload, nil
}
