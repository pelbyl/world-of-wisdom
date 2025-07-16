package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"world-of-wisdom/internal/apiserver"
	"world-of-wisdom/pkg/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var (
		port  = flag.String("port", normalizePort(getEnv("API_SERVER_PORT", "8081")), "API server port")
		dbURL = flag.String("db-url", "", "PostgreSQL connection URL (optional)")
	)
	flag.Parse()

	// Load configuration
	cfg := config.LoadConfig()

	// Build database URL if not provided
	if *dbURL == "" {
		*dbURL = fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
			cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresHost,
			cfg.PostgresPort, cfg.PostgresDB, cfg.PostgresSSLMode)
	}

	log.Printf("🚀 Starting API server on port %s", *port)
	log.Printf("📊 Connecting to database: %s", cfg.PostgresHost)

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer dbpool.Close()

	// Test connection
	if err := dbpool.Ping(ctx); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL database")

	// Create API server with handlers
	apiServer := apiserver.NewServer(dbpool)

	// Setup Echo routes
	e := apiServer.SetupRoutes()

	log.Printf("🌐 API server listening on %s", *port)
	log.Printf("📚 API documentation available at http://localhost%s/api/docs", *port)
	
	// Start server in goroutine
	go func() {
		if err := e.Start(*port); err != nil && err != http.ErrServerClosed {
			log.Printf("API server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down API server...")
	
	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}
	
	log.Printf("✅ API server gracefully stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func normalizePort(port string) string {
	if port == "" {
		return ":8081"
	}
	if port[0] != ':' {
		return ":" + port
	}
	return port
}