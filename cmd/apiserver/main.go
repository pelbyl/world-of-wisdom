package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"world-of-wisdom/api/db"
	"world-of-wisdom/internal/apiserver"
	"world-of-wisdom/pkg/config"
)

func main() {
	var (
		port     = flag.String("port", ":8082", "HTTP port to listen on")
		dbURL    = flag.String("db-url", "", "PostgreSQL connection URL (optional)")
		corsMode = flag.String("cors", "dev", "CORS mode: dev, production, or disabled")
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

	// Setup Gin
	if gin.Mode() == gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Configure CORS
	switch *corsMode {
	case "dev":
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "http://localhost:5173"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))
	case "production":
		router.Use(cors.New(cors.Config{
			AllowOrigins: []string{"https://yourdomain.com"},
			AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
			MaxAge:       12 * time.Hour,
		}))
	case "disabled":
		// No CORS middleware
	default:
		log.Printf("⚠️  Unknown CORS mode: %s, using dev mode", *corsMode)
		router.Use(cors.Default())
	}

	// Initialize database queries
	queries := db.New()

	// Setup API routes
	apiServer := apiserver.New(dbpool, queries)
	apiServer.SetupRoutes(router)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "wisdom-api",
			"version":   "1.0.0",
		})
	})

	// Start server
	log.Printf("🌟 API server ready at http://localhost%s", *port)
	log.Printf("📋 API documentation: http://localhost%s/api/v1/docs", *port)

	if err := router.Run(*port); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}