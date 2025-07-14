package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	// Database
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int

	// Server
	ServerPort    string
	APIServerPort string
	MetricsPort   string
	Algorithm     string
	Difficulty    int
	AdaptiveMode  bool
	Timeout       time.Duration

	// Environment
	Environment string
	LogLevel    string
}

func LoadConfig() *Config {
	return &Config{
		// Database defaults
		PostgresHost:     getEnvString("POSTGRES_HOST", "postgres"),
		PostgresPort:     getEnvInt("POSTGRES_PORT", 5432),
		PostgresUser:     getEnvString("POSTGRES_USER", "wisdom"),
		PostgresPassword: getEnvString("POSTGRES_PASSWORD", "wisdom123"),
		PostgresDB:       getEnvString("POSTGRES_DB", "wisdom"),
		PostgresSSLMode:  getEnvString("POSTGRES_SSL_MODE", "disable"),

		RedisHost:     getEnvString("REDIS_HOST", "redis"),
		RedisPort:     getEnvInt("REDIS_PORT", 6379),
		RedisPassword: getEnvString("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		// Server defaults
		ServerPort:    getEnvString("SERVER_PORT", ":8080"),
		APIServerPort: getEnvString("API_SERVER_PORT", ":8081"),
		MetricsPort:   getEnvString("METRICS_PORT", ":2112"),
		Algorithm:     getEnvString("ALGORITHM", "argon2"),
		Difficulty:    getEnvInt("DIFFICULTY", 2),
		AdaptiveMode:  getEnvBool("ADAPTIVE_MODE", true),
		Timeout:       getEnvDuration("TIMEOUT", 30*time.Second),

		// Environment
		Environment: getEnvString("ENV", "development"),
		LogLevel:    getEnvString("LOG_LEVEL", "info"),
	}
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
