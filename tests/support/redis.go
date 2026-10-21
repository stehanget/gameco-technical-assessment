package testsupport

import (
	"testing"

	"gameco-technical-assessment/internal/config"
	"gameco-technical-assessment/internal/platform/cache"

	"github.com/redis/go-redis/v9"
)

func OpenRedis(t *testing.T, cfg config.Config) *redis.Client {
	t.Helper()
	client := cache.NewRedis(cfg)
	if err := client.Ping(t.Context()).Err(); err != nil {
		t.Fatalf("ping Redis: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}
