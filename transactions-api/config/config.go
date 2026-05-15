package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port          string
	LogDir        string
	LogFile       string
	QueueSize     int
	FlushInterval time.Duration
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		LogDir:        getEnv("LOG_DIR", "/app/logs"),
		LogFile:       getEnv("LOG_FILE", "xapi.log"),
		QueueSize:     getEnvInt("QUEUE_SIZE", 100),
		FlushInterval: getEnvDuration("FLUSH_INTERVAL", 5*time.Second),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
