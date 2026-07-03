package app

import (
	"context"

	"gameco-technical-assessment/internal/modules/example"
	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/workers/jobs"

	"github.com/hibiken/asynq"
)

type WorkerApp struct {
	container  *Container
	dispatcher *jobs.Dispatcher
}

func NewWorkerApp() (*WorkerApp, error) {
	container, err := Bootstrap(context.Background())
	if err != nil {
		return nil, err
	}

	dispatcher := jobs.NewDispatcher(container.Config.Redis, container.Logger)
	exampleRepository := example.NewRepository(container.DB)
	dispatcher.Handle(jobs.TypeExampleCreate, func(ctx context.Context, task *asynq.Task) error {
		payload, err := jobs.ParseExampleCreateTask(task)
		if err != nil {
			return err
		}

		publicID, generateErr := identifier.NewPublicID()
		if generateErr != nil {
			return generateErr
		}
		_, err = exampleRepository.Create(example.Example{PublicID: publicID, Name: payload.Name})
		return err
	})

	return &WorkerApp{container: container, dispatcher: dispatcher}, nil
}

func (a *WorkerApp) Run(ctx context.Context) error {
	a.container.Logger.Info("worker started")
	defer a.container.Logger.Info("worker stopped")

	if err := a.dispatcher.Run(ctx); err != nil {
		_ = a.container.Close()
		return err
	}

	return a.container.Close()
}
