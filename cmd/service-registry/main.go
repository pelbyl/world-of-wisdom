package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"world-of-wisdom/pkg/services"
)

func main() {
	var (
		port = flag.String("port", ":8083", "HTTP port for service registry")
	)
	flag.Parse()

	log.Printf("🔗 Starting Service Registry on port %s", *port)

	// Create service registry
	registry := services.NewServiceRegistry()

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Enable CORS for all origins in development
	router.Use(cors.Default())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "service-registry",
			"version":   "1.0.0",
		})
	})

	// Service registration endpoints
	api := router.Group("/api/v1")
	{
		// Register a service
		api.POST("/services", func(c *gin.Context) {
			var instance services.ServiceInstance
			if err := c.ShouldBindJSON(&instance); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := registry.Register(&instance); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusCreated, gin.H{"status": "registered", "instance": instance})
		})

		// Deregister a service
		api.DELETE("/services/:id", func(c *gin.Context) {
			serviceID := c.Param("id")
			if err := registry.Deregister(serviceID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "deregistered", "serviceId": serviceID})
		})

		// Discover services by type
		api.GET("/services/:type", func(c *gin.Context) {
			serviceType := services.ServiceType(c.Param("type"))
			instances := registry.Discover(serviceType)

			c.JSON(http.StatusOK, gin.H{
				"serviceType": serviceType,
				"instances":   instances,
				"count":       len(instances),
			})
		})

		// Get all services
		api.GET("/services", func(c *gin.Context) {
			allServices := registry.ListAll()
			c.JSON(http.StatusOK, gin.H{
				"services": allServices,
				"count":    len(allServices),
			})
		})

		// Get specific service
		api.GET("/service/:id", func(c *gin.Context) {
			serviceID := c.Param("id")
			instance, exists := registry.GetService(serviceID)
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"instance": instance})
		})

		// Update service health
		api.PUT("/services/:id/health", func(c *gin.Context) {
			serviceID := c.Param("id")

			var healthUpdate struct {
				Health string `json:"health" binding:"required"`
			}

			if err := c.ShouldBindJSON(&healthUpdate); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := registry.UpdateHealth(serviceID, healthUpdate.Health); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "updated", "health": healthUpdate.Health})
		})

		// Send heartbeat
		api.POST("/services/:id/heartbeat", func(c *gin.Context) {
			serviceID := c.Param("id")

			if err := registry.Heartbeat(serviceID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "heartbeat received"})
		})
	}

	// WebSocket endpoint for real-time service updates
	router.GET("/ws", func(c *gin.Context) {
		handleWebSocket(c, registry)
	})

	// Stats endpoint
	router.GET("/stats", func(c *gin.Context) {
		allServices := registry.ListAll()
		stats := make(map[services.ServiceType]int)

		for _, instance := range allServices {
			if instance.Health == "healthy" {
				stats[instance.Type]++
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"totalServices":  len(allServices),
			"servicesByType": stats,
			"timestamp":      time.Now().UTC(),
		})
	})

	log.Printf("🌟 Service Registry ready at http://localhost%s", *port)
	log.Printf("📋 Registry API: http://localhost%s/api/v1/services", *port)
	log.Printf("📊 Registry Stats: http://localhost%s/stats", *port)

	if err := router.Run(*port); err != nil {
		log.Fatalf("❌ Failed to start service registry: %v", err)
	}
}

func handleWebSocket(c *gin.Context, registry *services.ServiceRegistry) {
	// For now, return a simple message
	// In a full implementation, this would upgrade to WebSocket
	// and stream service events to clients
	c.JSON(http.StatusOK, gin.H{
		"message": "WebSocket endpoint for service events",
		"note":    "Full WebSocket implementation would stream real-time service changes",
	})
}
