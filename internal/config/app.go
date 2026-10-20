package config

import "time"

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

func loadAppConfig() (AppConfig, error) {
	readTimeout, err := getEnvDuration("APP_READ_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	writeTimeout, err := getEnvDuration("APP_WRITE_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	shutdownTimeout, err := getEnvDuration("APP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return AppConfig{}, err
	}

	accessTokenTTL, err := getEnvDuration("APP_ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return AppConfig{}, err
	}

	refreshTokenTTL, err := getEnvDuration("APP_REFRESH_TOKEN_TTL", 168*time.Hour)
	if err != nil {
		return AppConfig{}, err
	}

	return AppConfig{
		Name:            getEnv("APP_NAME", "gameco-technical-assessment"),
		Env:             getEnv("APP_ENV", "development"),
		Port:            getEnv("APP_PORT", "8080"),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		ShutdownTimeout: shutdownTimeout,
		JWTSecret:       getEnv("APP_JWT_SECRET", "change-me"),
		AccessTokenTTL:  accessTokenTTL,
		RefreshTokenTTL: refreshTokenTTL,
	}, nil
}
