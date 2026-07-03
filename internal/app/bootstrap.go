package app

import (
	"context"
	"fmt"
	"log/slog"

	"gameco-technical-assessment/internal/config"
	platformauth "gameco-technical-assessment/internal/platform/auth"
	"gameco-technical-assessment/internal/platform/cache"
	"gameco-technical-assessment/internal/platform/database"
	"gameco-technical-assessment/internal/platform/logger"
	"gameco-technical-assessment/internal/workers/jobs"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Container struct {
	Config    config.Config
	Logger    *slog.Logger
	DB        *gorm.DB
	Redis     *redis.Client
	Jobs      *jobs.Enqueuer
	JWTSigner *platformauth.JWTManager
}

func Bootstrap(ctx context.Context) (*Container, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	appLogger := logger.New(cfg.App.Env)

	db, err := database.NewGORM(cfg)
	if err != nil {
		return nil, fmt.Errorf("init gorm: %w", err)
	}

	redisClient := cache.NewRedis(cfg)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	jobEnqueuer := jobs.NewEnqueuer(cfg.Redis)

	return &Container{
		Config:    cfg,
		Logger:    appLogger,
		DB:        db,
		Redis:     redisClient,
		Jobs:      jobEnqueuer,
		JWTSigner: platformauth.NewJWTManager(cfg.App.JWTSecret),
	}, nil
}

func (c *Container) Close() error {
	if c.Jobs != nil {
		if err := c.Jobs.Close(); err != nil {
			return err
		}
	}

	if err := database.CloseGORM(c.DB); err != nil {
		return err
	}

	return c.Redis.Close()
}
