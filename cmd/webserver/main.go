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

	"github.com/jackc/pgx/v5/pgxpool"
	"world-of-wisdom/api/db"
	"world-of-wisdom/internal/webserver"
	"world-of-wisdom/pkg/config"
)

func main() {
	var (
		port = flag.String("port", ":8081", "Web server port")
		tcpServer = flag.String("tcp-server", "localhost:8080", "TCP server address")
		algorithm = flag.String("algorithm", "argon2", "PoW algorithm: sha256 or argon2")
		dbURL    = flag.String("db-url", "", "PostgreSQL connection URL (optional)")
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

	log.Printf("🚀 Starting webserver on port %s", *port)
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

	// Initialize database queries
	queries := db.New()

	server := webserver.NewWebServer(*tcpServer, *algorithm, dbpool, queries)

	http.HandleFunc("/ws", server.HandleWebSocket)
	http.HandleFunc("/api/stats", server.HandleStats)
	http.HandleFunc("/api/simulate", server.HandleSimulateClient)
	http.HandleFunc("/api/clear", server.HandleClearState)

	go server.Start()

	log.Printf("Web server listening on %s", *port)
	go func() {
		if err := http.ListenAndServe(*port, nil); err != nil {
			log.Fatalf("Web server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down web server...")
	server.Stop()
}