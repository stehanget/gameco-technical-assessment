package config

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func loadRedisConfig() (RedisConfig, error) {
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return RedisConfig{}, err
	}

	return RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       redisDB,
	}, nil
}
