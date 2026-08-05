// Package config provides application configuration loading.
package config

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Env    string
	Server ServerConfig
	Logger LoggerConfig
	DB     DBConfig
	Auth   AuthConfig
	Upload UploadConfig
}

type UploadConfig struct {
	Dir           string
	MaxBytes      int64
	MaxPerListing int
}

type AuthConfig struct {
	JWTSecret string
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type LoggerConfig struct {
	Level string
}

type DBConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Env: getEnv("ENV", "dev"),

		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvAsDuration("SERVER_READ_TIMEOUT", "15s"),
			WriteTimeout: getEnvAsDuration("SERVER_WRITE_TIMEOUT", "15s"),
			IdleTimeout:  getEnvAsDuration("SERVER_IDLE_TIMEOUT", "60s"),
		},

		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
		DB: DBConfig{
			URL:             mustGetEnv("DB_URL"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},

		Auth: AuthConfig{
			JWTSecret: mustGetEnv("JWT_SECRET"),
		},

		Upload: UploadConfig{
			Dir: getEnv("UPLOAD_DIR", "./uploads"),
			MaxBytes: getEnvAsInt64("MAX_UPLOAD_BYTES", 5<<20),
			MaxPerListing: getEnvAsInt("MAX_IMAGES_PER_LISTING", 5),
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		slog.Error("required environment variable not set",
			"key", key,
		)
		os.Exit(1)
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		slog.Warn("invalid int environment variable, using default",
			"key", key,
			"value", strconv.Quote(valueStr),
			"default", defaultValue,
			"error", err,
		)
		return defaultValue
	}
	return value
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseInt(valueStr, 10, 64)
	if err != nil {
		slog.Warn("invalid int environment variable, using default",
			"key", key,
			"value", strconv.Quote(valueStr),
			"default", defaultValue,
			"error", err,
		)
		return defaultValue
	}
	return value
}

func getEnvAsDuration(key, defaultValue string) time.Duration {
	valueStr := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		// Fallback to parsing the default if provided value is invalid
		duration, err = time.ParseDuration(defaultValue)
		if err != nil {
			return 0
		}
	}
	return duration
}
