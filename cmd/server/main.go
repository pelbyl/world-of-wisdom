package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"world-of-wisdom/internal/server"
	"world-of-wisdom/pkg/config"
)

func main() {
	var (
		port        = flag.String("port", normalizePort(getEnv("SERVER_PORT", "8080")), "TCP port to listen on")
		difficulty  = flag.Int("difficulty", getEnvInt("DIFFICULTY", 2), "Initial difficulty (1-6)")
		timeout     = flag.Duration("timeout", 30*time.Second, "Client timeout")
		adaptive    = flag.Bool("adaptive", getEnvBool("ADAPTIVE_MODE", true), "Enable adaptive difficulty")
		metricsPort = flag.String("metrics-port", normalizePort(getEnv("METRICS_PORT", "2112")), "Prometheus metrics port")
		algorithm   = flag.String("algorithm", getEnv("ALGORITHM", "argon2"), "PoW algorithm: sha256 or argon2")
		dbURL       = flag.String("db-url", "", "PostgreSQL connection URL (optional)")
	)
	flag.Parse()

	// Load configuration for database settings
	appConfig := config.LoadConfig()

	// Build database URL if not provided
	if *dbURL == "" {
		*dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			appConfig.PostgresUser, appConfig.PostgresPassword, appConfig.PostgresHost,
			appConfig.PostgresPort, appConfig.PostgresDB, appConfig.PostgresSSLMode)
	}

	cfg := server.Config{
		Port:         *port,
		Difficulty:   *difficulty,
		Timeout:      *timeout,
		AdaptiveMode: *adaptive,
		MetricsPort:  *metricsPort,
		Algorithm:    *algorithm,
		DatabaseURL:  *dbURL,
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	if err := srv.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func normalizePort(port string) string {
	if port == "" {
		return ":8080"
	}
	if port[0] != ':' {
		return ":" + port
	}
	return port
}