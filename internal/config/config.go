package config

type Config struct {
	App       AppConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	RateLimit RateLimitConfig
}

func Load() (Config, error) {
	appConfig, err := loadAppConfig()
	if err != nil {
		return Config{}, err
	}

	redisConfig, err := loadRedisConfig()
	if err != nil {
		return Config{}, err
	}

	return Config{
		App:       appConfig,
		Postgres:  loadPostgresConfig(),
		Redis:     redisConfig,
		RateLimit: loadRateLimitConfig(),
	}, nil
}
