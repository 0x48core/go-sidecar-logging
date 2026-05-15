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
	MaxBatchSize  int
	WatchInterval time.Duration
	ESUrl         string
	ESIndex       string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8081"),
		LogDir:        getEnv("LOG_DIR", "/app/logs"),
		LogFile:       getEnv("LOG_FILE", "xapi.log"),
		MaxBatchSize:  getEnvInt("MAX_BATCH_SIZE", 5),
		WatchInterval: getEnvDuration("WATCH_INTERVAL", 5*time.Second),
		ESUrl:         getEnv("ES_URL", "http://localhost:9200"),
		ESIndex:       getEnv("ES_INDEX", "application-logs"),
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
