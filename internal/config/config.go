package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
}

type AppConfig struct {
	Name            string
	Env             string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func Load() (Config, error) {
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := getEnvDuration("APP_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := getEnvDuration("APP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getEnvDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	accessTokenTTL, err := getEnvDuration("APP_ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}

	refreshTokenTTL, err := getEnvDuration("APP_REFRESH_TOKEN_TTL", 168*time.Hour)
	if err != nil {
		return Config{}, err
	}

	return Config{
		App: AppConfig{
			Name:            getEnv("APP_NAME", "gameco-technical-assessment"),
			Env:             getEnv("APP_ENV", "development"),
			Port:            getEnv("APP_PORT", "8080"),
			ReadTimeout:     readTimeout,
			WriteTimeout:    writeTimeout,
			ShutdownTimeout: shutdownTimeout,
			JWTSecret:       getEnv("APP_JWT_SECRET", "change-me"),
			AccessTokenTTL:  accessTokenTTL,
			RefreshTokenTTL: refreshTokenTTL,
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "postgres"),
			Password: getEnv("POSTGRES_PASSWORD", "postgres"),
			Database: getEnv("POSTGRES_DB", "gameco_technical_assessment"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
	}, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}

	return parsed, nil
}
