package example

import (
	"context"
	"fmt"

	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/workers/jobs"
)

type Service struct {
	repository *Repository
	jobs       *jobs.Enqueuer
}

func NewService(repository *Repository, enqueuer *jobs.Enqueuer) *Service {
	return &Service{repository: repository, jobs: enqueuer}
}

func (s *Service) List() ([]Response, error) {
	examples, err := s.repository.List()
	if err != nil {
		return nil, err
	}

	response := make([]Response, 0, len(examples))
	for _, item := range examples {
		response = append(response, Response{PublicID: item.PublicID, Name: item.Name})
	}

	return response, nil
}

func (s *Service) Create(input CreateRequest) (Response, error) {
	publicID, err := identifier.NewPublicID()
	if err != nil {
		return Response{}, err
	}
	example, err := s.repository.Create(Example{PublicID: publicID, Name: input.Name})
	if err != nil {
		return Response{}, err
	}

	return Response{PublicID: example.PublicID, Name: example.Name}, nil
}

func (s *Service) EnqueueCreate(ctx context.Context, input CreateRequest) (jobs.EnqueueResult, error) {
	if s.jobs == nil {
		return jobs.EnqueueResult{}, fmt.Errorf("job queue is not configured")
	}

	task, err := jobs.NewExampleCreateTask(input.Name)
	if err != nil {
		return jobs.EnqueueResult{}, err
	}

	return s.jobs.Enqueue(ctx, task)
}
