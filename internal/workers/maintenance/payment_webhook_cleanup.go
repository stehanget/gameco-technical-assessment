package maintenance

import (
	"context"
	"log/slog"
	"time"

	"gameco-technical-assessment/internal/modules/checkout"
)

type PaymentWebhookCleanup struct {
	repository *checkout.Repository
	retention  time.Duration
	interval   time.Duration
	logger     *slog.Logger
}

func NewPaymentWebhookCleanup(repository *checkout.Repository, retention, interval time.Duration, logger *slog.Logger) *PaymentWebhookCleanup {
	return &PaymentWebhookCleanup{
		repository: repository,
		retention:  retention,
		interval:   interval,
		logger:     logger,
	}
}

func (w *PaymentWebhookCleanup) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cutoff := now.UTC().Add(-w.retention)
			deleted, err := w.repository.CleanupProcessedWebhookEvents(ctx, cutoff)
			if err != nil {
				w.logger.Error("payment webhook cleanup failed", "error", err)
				continue
			}
			if deleted > 0 {
				w.logger.Info("payment webhook events cleaned", "deleted", deleted)
			}
		}
	}
}
