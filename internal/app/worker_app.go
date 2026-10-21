package app

import (
	"context"

	"gameco-technical-assessment/internal/modules/checkout"
	"gameco-technical-assessment/internal/modules/example"
	"gameco-technical-assessment/internal/platform/identifier"
	"gameco-technical-assessment/internal/workers/jobs"
	"gameco-technical-assessment/internal/workers/maintenance"

	"github.com/hibiken/asynq"
)

type WorkerApp struct {
	container          *Container
	dispatcher         *jobs.Dispatcher
	webhookCleanup     *maintenance.PaymentWebhookCleanup
	idempotencyCleanup *maintenance.CheckoutIdempotencyCleanup
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

	webhookCleanup := maintenance.NewPaymentWebhookCleanup(
		checkout.NewRepository(container.DB),
		container.Config.App.PaymentWebhookEventRetention,
		container.Config.App.PaymentWebhookCleanupInterval,
		container.Logger,
	)
	idempotencyCleanup := maintenance.NewCheckoutIdempotencyCleanup(
		checkout.NewRepository(container.DB),
		container.Config.App.CheckoutIdempotencyRetention,
		container.Config.App.CheckoutIdempotencyCleanupInterval,
		container.Logger,
	)
	return &WorkerApp{container: container, dispatcher: dispatcher, webhookCleanup: webhookCleanup, idempotencyCleanup: idempotencyCleanup}, nil
}

func (a *WorkerApp) Run(ctx context.Context) error {
	a.container.Logger.Info("worker started")
	defer a.container.Logger.Info("worker stopped")
	cleanupContext, cancelCleanup := context.WithCancel(ctx)
	defer cancelCleanup()
	go a.webhookCleanup.Run(cleanupContext)
	go a.idempotencyCleanup.Run(cleanupContext)

	if err := a.dispatcher.Run(ctx); err != nil {
		_ = a.container.Close()
		return err
	}

	return a.container.Close()
}
