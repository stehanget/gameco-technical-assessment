package jobs

import (
	"context"
	"fmt"
	"log/slog"

	"gameco-technical-assessment/internal/config"

	"github.com/hibiken/asynq"
)

const defaultQueueName = "default"

type Enqueuer struct {
	client *asynq.Client
}

type EnqueueResult struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Queue string `json:"queue"`
	State string `json:"state"`
}

type Dispatcher struct {
	server *asynq.Server
	mux    *asynq.ServeMux
	logger *slog.Logger
}

func NewEnqueuer(cfg config.RedisConfig) *Enqueuer {
	return &Enqueuer{
		client: asynq.NewClient(redisClientOpt(cfg)),
	}
}

func (e *Enqueuer) Enqueue(ctx context.Context, task *asynq.Task, opts ...asynq.Option) (EnqueueResult, error) {
	info, err := e.client.EnqueueContext(ctx, task, opts...)
	if err != nil {
		return EnqueueResult{}, err
	}

	return EnqueueResult{
		ID:    info.ID,
		Type:  info.Type,
		Queue: info.Queue,
		State: info.State.String(),
	}, nil
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}

func NewDispatcher(cfg config.RedisConfig, logger *slog.Logger) *Dispatcher {
	server := asynq.NewServer(
		redisClientOpt(cfg),
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				defaultQueueName: 1,
			},
		},
	)

	return &Dispatcher{
		server: server,
		mux:    asynq.NewServeMux(),
		logger: logger,
	}
}

func (d *Dispatcher) Handle(taskType string, handler asynq.HandlerFunc) {
	d.mux.HandleFunc(taskType, handler)
}

func (d *Dispatcher) Run(ctx context.Context) error {
	if err := d.server.Start(d.mux); err != nil {
		return fmt.Errorf("start queue server: %w", err)
	}

	d.logger.Info("queue worker started", "queue", defaultQueueName)
	<-ctx.Done()
	d.logger.Info("queue worker stopping")
	d.server.Shutdown()
	return nil
}

func redisClientOpt(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}
