package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	App   AppConfig
	DB    DBConfig
	Redis RedisConfig
}

type AppConfig struct {
	Env  string
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type RedisConfig struct {
	Host string
	Port int
}

func (a AppConfig) Address() string {
	return fmt.Sprintf(":%d", a.Port)
}

func Load() (Config, error) {
	cfg := Config{
		App: AppConfig{
			Env:  getEnvOrDefault("APP_ENV", "local"),
			Port: getIntOrDefault("APP_PORT", 8080),
		},
		DB: DBConfig{
			Host:     getEnvOrDefault("DB_HOST", "127.0.0.1"),
			Port:     getIntOrDefault("DB_PORT", 3306),
			User:     getEnvOrDefault("DB_USER", "stock"),
			Password: getEnvOrDefault("DB_PASSWORD", "stockpass"),
			Name:     getEnvOrDefault("DB_NAME", "stockbot"),
		},
		Redis: RedisConfig{
			Host: getEnvOrDefault("REDIS_HOST", "127.0.0.1"),
			Port: getIntOrDefault("REDIS_PORT", 6379),
		},
	}

	if err := validatePort("APP_PORT", cfg.App.Port); err != nil {
		return Config{}, err
	}
	if err := validatePort("DB_PORT", cfg.DB.Port); err != nil {
		return Config{}, err
	}
	if err := validatePort("REDIS_PORT", cfg.Redis.Port); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func validatePort(key string, port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535", key)
	}
	return nil
}
