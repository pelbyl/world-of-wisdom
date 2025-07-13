package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"world-of-wisdom/pkg/services"
)

type Monitor struct {
	registry         *services.HTTPServiceClient
	serviceHealth    *prometheus.GaugeVec
	serviceInstances *prometheus.GaugeVec
	responseTime     *prometheus.HistogramVec
	lastChecked      *prometheus.GaugeVec
}

type ServiceStatus struct {
	ServiceType    string           `json:"serviceType"`
	HealthyCount   int              `json:"healthyCount"`
	UnhealthyCount int              `json:"unhealthyCount"`
	TotalInstances int              `json:"totalInstances"`
	Instances      []InstanceDetail `json:"instances"`
	LastChecked    time.Time        `json:"lastChecked"`
}

type InstanceDetail struct {
	ID           string            `json:"id"`
	Address      string            `json:"address"`
	Port         int               `json:"port"`
	Health       string            `json:"health"`
	LastSeen     time.Time         `json:"lastSeen"`
	Metadata     map[string]string `json:"metadata"`
	ResponseTime float64           `json:"responseTime"`
}

type SystemOverview struct {
	TotalServices   int             `json:"totalServices"`
	HealthyServices int             `json:"healthyServices"`
	ServiceStatuses []ServiceStatus `json:"serviceStatuses"`
	Timestamp       time.Time       `json:"timestamp"`
	SystemHealth    string          `json:"systemHealth"`
}

func NewMonitor(registryURL string) *Monitor {
	return &Monitor{
		registry: services.NewHTTPServiceClient(registryURL),
		serviceHealth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "service_health_status",
				Help: "Health status of services (1=healthy, 0=unhealthy)",
			},
			[]string{"service_type", "service_id", "address"},
		),
		serviceInstances: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "service_instances_total",
				Help: "Total number of service instances by type",
			},
			[]string{"service_type", "status"},
		),
		responseTime: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "service_health_check_duration_seconds",
				Help:    "Duration of service health checks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service_type", "service_id"},
		),
		lastChecked: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "service_last_health_check_timestamp",
				Help: "Timestamp of last health check",
			},
			[]string{"service_type", "service_id"},
		),
	}
}

func (m *Monitor) registerMetrics() {
	prometheus.MustRegister(m.serviceHealth)
	prometheus.MustRegister(m.serviceInstances)
	prometheus.MustRegister(m.responseTime)
	prometheus.MustRegister(m.lastChecked)
}

func (m *Monitor) checkServiceHealth(instance *services.ServiceInstance) (bool, float64) {
	start := time.Now()

	healthURL := fmt.Sprintf("http://%s:%d/health", instance.Address, instance.Port)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(healthURL)
	duration := time.Since(start).Seconds()

	if err != nil {
		log.Printf("Health check failed for %s (%s): %v", instance.ID, healthURL, err)
		return false, duration
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode == http.StatusOK

	// Update Prometheus metrics
	healthValue := 0.0
	if healthy {
		healthValue = 1.0
	}

	m.serviceHealth.WithLabelValues(
		string(instance.Type),
		instance.ID,
		fmt.Sprintf("%s:%d", instance.Address, instance.Port),
	).Set(healthValue)

	m.responseTime.WithLabelValues(
		string(instance.Type),
		instance.ID,
	).Observe(duration)

	m.lastChecked.WithLabelValues(
		string(instance.Type),
		instance.ID,
	).SetToCurrentTime()

	return healthy, duration
}

func (m *Monitor) monitorServices() {
	serviceTypes := []services.ServiceType{
		services.ServiceTypeTCPServer,
		services.ServiceTypeWebServer,
		services.ServiceTypeAPIServer,
		services.ServiceTypeLoadBalancer,
	}

	for _, serviceType := range serviceTypes {
		instances, err := m.registry.DiscoverServices(serviceType)
		if err != nil {
			log.Printf("Failed to discover %s services: %v", serviceType, err)
			continue
		}

		healthyCount := 0
		unhealthyCount := 0

		for _, instance := range instances {
			healthy, _ := m.checkServiceHealth(instance)
			if healthy {
				healthyCount++
			} else {
				unhealthyCount++
			}
		}

		// Update instance count metrics
		m.serviceInstances.WithLabelValues(string(serviceType), "healthy").Set(float64(healthyCount))
		m.serviceInstances.WithLabelValues(string(serviceType), "unhealthy").Set(float64(unhealthyCount))

		log.Printf("Service %s: %d healthy, %d unhealthy", serviceType, healthyCount, unhealthyCount)
	}
}

func (m *Monitor) getSystemOverview() (*SystemOverview, error) {
	serviceTypes := []services.ServiceType{
		services.ServiceTypeTCPServer,
		services.ServiceTypeWebServer,
		services.ServiceTypeAPIServer,
		services.ServiceTypeLoadBalancer,
	}

	var statuses []ServiceStatus
	totalServices := 0
	healthyServices := 0

	for _, serviceType := range serviceTypes {
		instances, err := m.registry.DiscoverServices(serviceType)
		if err != nil {
			continue
		}

		status := ServiceStatus{
			ServiceType:    string(serviceType),
			TotalInstances: len(instances),
			LastChecked:    time.Now(),
		}

		for _, instance := range instances {
			healthy, responseTime := m.checkServiceHealth(instance)

			detail := InstanceDetail{
				ID:           instance.ID,
				Address:      instance.Address,
				Port:         instance.Port,
				Health:       instance.Health,
				LastSeen:     instance.LastSeen,
				Metadata:     instance.Metadata,
				ResponseTime: responseTime,
			}

			if healthy {
				detail.Health = "healthy"
				status.HealthyCount++
			} else {
				detail.Health = "unhealthy"
				status.UnhealthyCount++
			}

			status.Instances = append(status.Instances, detail)
		}

		statuses = append(statuses, status)
		totalServices += status.TotalInstances
		healthyServices += status.HealthyCount
	}

	// Determine overall system health
	systemHealth := "critical"
	if healthyServices == totalServices && totalServices > 0 {
		systemHealth = "healthy"
	} else if healthyServices > totalServices/2 {
		systemHealth = "degraded"
	}

	// Sort statuses by service type for consistent output
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].ServiceType < statuses[j].ServiceType
	})

	return &SystemOverview{
		TotalServices:   totalServices,
		HealthyServices: healthyServices,
		ServiceStatuses: statuses,
		Timestamp:       time.Now(),
		SystemHealth:    systemHealth,
	}, nil
}

func main() {
	var (
		port        = flag.String("port", ":8085", "Monitor port")
		registryURL = flag.String("registry", "http://service-registry:8083", "Service registry URL")
		interval    = flag.Duration("interval", 30*time.Second, "Health check interval")
	)
	flag.Parse()

	log.Printf("📊 Starting Monitor Service on port %s", *port)
	log.Printf("🔗 Service Registry: %s", *registryURL)
	log.Printf("⏱️  Health Check Interval: %v", *interval)

	monitor := NewMonitor(*registryURL)
	monitor.registerMetrics()

	// Register monitor with service registry
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	monitorInstance := &services.ServiceInstance{
		ID:      "monitor-1",
		Type:    services.ServiceTypeMonitor,
		Address: "monitor",
		Port:    8085,
		Health:  "healthy",
		Metadata: map[string]string{
			"version": "1.0.0",
			"role":    "health-monitor",
		},
	}

	if err := monitor.registry.RegisterService(ctx, monitorInstance); err != nil {
		log.Printf("⚠️  Failed to register with service registry: %v", err)
	}

	// Setup Gin router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Enable CORS
	router.Use(cors.Default())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "monitor",
			"version":   "1.0.0",
		})
	})

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// System overview endpoint
	router.GET("/system", func(c *gin.Context) {
		overview, err := monitor.getSystemOverview()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, overview)
	})

	// Service status endpoint
	router.GET("/services/:type", func(c *gin.Context) {
		serviceType := services.ServiceType(c.Param("type"))
		instances, err := monitor.registry.DiscoverServices(serviceType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var details []InstanceDetail
		healthyCount := 0

		for _, instance := range instances {
			healthy, responseTime := monitor.checkServiceHealth(instance)

			detail := InstanceDetail{
				ID:           instance.ID,
				Address:      instance.Address,
				Port:         instance.Port,
				Health:       instance.Health,
				LastSeen:     instance.LastSeen,
				Metadata:     instance.Metadata,
				ResponseTime: responseTime,
			}

			if healthy {
				detail.Health = "healthy"
				healthyCount++
			} else {
				detail.Health = "unhealthy"
			}

			details = append(details, detail)
		}

		c.JSON(http.StatusOK, ServiceStatus{
			ServiceType:    string(serviceType),
			HealthyCount:   healthyCount,
			UnhealthyCount: len(instances) - healthyCount,
			TotalInstances: len(instances),
			Instances:      details,
			LastChecked:    time.Now(),
		})
	})

	// Start monitoring routine
	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()

		// Initial check
		monitor.monitorServices()

		for range ticker.C {
			monitor.monitorServices()
		}
	}()

	// Start heartbeat routine
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := monitor.registry.SendHeartbeat(ctx, monitorInstance.ID); err != nil {
				log.Printf("⚠️  Failed to send heartbeat: %v", err)
			}
			cancel()
		}
	}()

	log.Printf("🌟 Monitor Service ready at http://localhost%s", *port)
	log.Printf("📊 System Overview: http://localhost%s/system", *port)
	log.Printf("📈 Prometheus Metrics: http://localhost%s/metrics", *port)

	if err := router.Run(*port); err != nil {
		log.Fatalf("❌ Failed to start monitor: %v", err)
	}
}
