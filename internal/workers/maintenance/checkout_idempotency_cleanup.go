package maintenance

import (
	"context"
	"log/slog"
	"time"

	"gameco-technical-assessment/internal/modules/checkout"
)

type CheckoutIdempotencyCleanup struct {
	repository *checkout.Repository
	retention  time.Duration
	interval   time.Duration
	logger     *slog.Logger
}

func NewCheckoutIdempotencyCleanup(repository *checkout.Repository, retention, interval time.Duration, logger *slog.Logger) *CheckoutIdempotencyCleanup {
	return &CheckoutIdempotencyCleanup{repository: repository, retention: retention, interval: interval, logger: logger}
}

func (w *CheckoutIdempotencyCleanup) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			cutoff := now.UTC().Add(-w.retention)
			deleted, err := w.repository.CleanupCompletedIdempotencyKeys(ctx, cutoff)
			if err != nil {
				w.logger.Error("checkout idempotency cleanup failed", "error", err)
				continue
			}
			if deleted > 0 {
				w.logger.Info("checkout idempotency keys cleaned", "deleted", deleted)
			}
		}
	}
}
