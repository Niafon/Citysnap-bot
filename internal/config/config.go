package config

import (
	"fmt"
	"os"
)

type Config struct {
	TelegramToken string
	DatabaseURL   string
	RedisURL      string
	AppEnv        string
}

func MustLoad() *Config {
	cfg := &Config{
		TelegramToken: getEnv("TELEGRAM_TOKEN", ""),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://app:secret@localhost:5432/citysnap?sslmode=disable"),
		RedisURL:      getEnv("REDIS_URL", "localhost:6379"),
		AppEnv:        getEnv("APP_ENV", "development"),
	}

	if cfg.TelegramToken == "" {
		fmt.Println("WARN: TELEGRAM_TOKEN not set, bot will not start")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
