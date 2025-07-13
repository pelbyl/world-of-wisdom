package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"world-of-wisdom/pkg/services"
)

type Gateway struct {
	registry     *services.HTTPServiceClient
	healthChecks map[string]time.Time
	healthMutex  sync.RWMutex
	roundRobin   map[services.ServiceType]int
	rrMutex      sync.RWMutex
}

func NewGateway(registryURL string) *Gateway {
	return &Gateway{
		registry:     services.NewHTTPServiceClient(registryURL),
		healthChecks: make(map[string]time.Time),
		roundRobin:   make(map[services.ServiceType]int),
	}
}

func (g *Gateway) healthCheck(instance *services.ServiceInstance) bool {
	g.healthMutex.RLock()
	lastCheck, exists := g.healthChecks[instance.ID]
	g.healthMutex.RUnlock()

	// Only check health every 30 seconds
	if exists && time.Since(lastCheck) < 30*time.Second {
		return instance.Health == "healthy"
	}

	// Perform health check
	healthURL := fmt.Sprintf("http://%s:%d/health", instance.Address, instance.Port)
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(healthURL)
	if err != nil {
		log.Printf("Health check failed for %s: %v", instance.ID, err)
		return false
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK

	g.healthMutex.Lock()
	g.healthChecks[instance.ID] = time.Now()
	g.healthMutex.Unlock()

	return healthy
}

func (g *Gateway) selectHealthyInstance(serviceType services.ServiceType) (*services.ServiceInstance, error) {
	instances, err := g.registry.DiscoverServices(serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no instances found for service type: %s", serviceType)
	}

	// Filter healthy instances
	var healthyInstances []*services.ServiceInstance
	for _, instance := range instances {
		if g.healthCheck(instance) {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return nil, fmt.Errorf("no healthy instances found for service type: %s", serviceType)
	}

	// Round-robin selection
	g.rrMutex.Lock()
	index := g.roundRobin[serviceType] % len(healthyInstances)
	g.roundRobin[serviceType] = (index + 1) % len(healthyInstances)
	g.rrMutex.Unlock()

	return healthyInstances[index], nil
}

func (g *Gateway) proxyRequest(c *gin.Context, serviceType services.ServiceType) {
	instance, err := g.selectHealthyInstance(serviceType)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Service unavailable",
			"details": err.Error(),
		})
		return
	}

	// Create reverse proxy
	target, err := url.Parse(fmt.Sprintf("http://%s:%d", instance.Address, instance.Port))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to parse target URL",
		})
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Modify the request
	c.Request.URL.Host = target.Host
	c.Request.URL.Scheme = target.Scheme
	c.Request.Header.Set("X-Forwarded-Host", c.Request.Header.Get("Host"))
	c.Request.Header.Set("X-Gateway-Service", string(serviceType))

	proxy.ServeHTTP(c.Writer, c.Request)
}

func main() {
	var (
		port        = flag.String("port", ":8084", "Gateway port")
		registryURL = flag.String("registry", "http://service-registry:8083", "Service registry URL")
	)
	flag.Parse()

	log.Printf("🌐 Starting API Gateway on port %s", *port)
	log.Printf("🔗 Service Registry: %s", *registryURL)

	gateway := NewGateway(*registryURL)

	// Register gateway with service registry
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gatewayInstance := &services.ServiceInstance{
		ID:      "gateway-1",
		Type:    services.ServiceTypeLoadBalancer,
		Address: "gateway",
		Port:    8084,
		Health:  "healthy",
		Metadata: map[string]string{
			"version": "1.0.0",
			"role":    "api-gateway",
		},
	}

	if err := gateway.registry.RegisterService(ctx, gatewayInstance); err != nil {
		log.Printf("⚠️  Failed to register with service registry: %v", err)
	}

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Enable CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "api-gateway",
			"version":   "1.0.0",
		})
	})

	// Gateway stats endpoint
	router.GET("/gateway/stats", func(c *gin.Context) {
		gateway.healthMutex.RLock()
		healthCount := len(gateway.healthChecks)
		gateway.healthMutex.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"healthChecks":    healthCount,
			"registryURL":     *registryURL,
			"timestamp":       time.Now().UTC(),
			"gatewayInstance": gatewayInstance,
		})
	})

	// API routes - proxy to API servers
	api := router.Group("/api")
	{
		api.Any("/v1/*path", func(c *gin.Context) {
			gateway.proxyRequest(c, services.ServiceTypeAPIServer)
		})
	}

	// WebSocket proxy to web servers
	router.GET("/ws", func(c *gin.Context) {
		gateway.proxyRequest(c, services.ServiceTypeWebServer)
	})

	// Metrics proxy to monitoring
	router.GET("/metrics", func(c *gin.Context) {
		gateway.proxyRequest(c, services.ServiceTypeMonitor)
	})

	// Service discovery endpoint
	router.GET("/services", func(c *gin.Context) {
		serviceType := c.Query("type")
		if serviceType == "" {
			// Return all services
			allServices := make(map[services.ServiceType][]*services.ServiceInstance)
			for _, svcType := range []services.ServiceType{
				services.ServiceTypeTCPServer,
				services.ServiceTypeWebServer,
				services.ServiceTypeAPIServer,
				services.ServiceTypeMonitor,
			} {
				instances, err := gateway.registry.DiscoverServices(svcType)
				if err == nil {
					allServices[svcType] = instances
				}
			}
			c.JSON(http.StatusOK, gin.H{"services": allServices})
			return
		}

		instances, err := gateway.registry.DiscoverServices(services.ServiceType(serviceType))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"serviceType": serviceType,
			"instances":   instances,
			"count":       len(instances),
		})
	})

	// Start heartbeat routine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := gateway.registry.SendHeartbeat(ctx, gatewayInstance.ID); err != nil {
				log.Printf("⚠️  Failed to send heartbeat: %v", err)
			}
			cancel()
		}
	}()

	log.Printf("🌟 API Gateway ready at http://localhost%s", *port)
	log.Printf("🔗 Service Discovery: http://localhost%s/services", *port)
	log.Printf("📊 Gateway Stats: http://localhost%s/gateway/stats", *port)
	log.Printf("📝 API Proxy: http://localhost%s/api/v1/*", *port)

	if err := router.Run(*port); err != nil {
		log.Fatalf("❌ Failed to start gateway: %v", err)
	}
}
