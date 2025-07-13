package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

// ServiceIntegration provides common functionality for services to integrate with the registry
type ServiceIntegration struct {
	registry    *ServiceRegistry
	client      *ServiceClient
	instance    *ServiceInstance
	heartbeatTicker *time.Ticker
}

// ServiceConfig holds configuration for service integration
type ServiceConfig struct {
	ServiceID   string
	ServiceType ServiceType
	Port        int
	Metadata    map[string]string
}

// NewServiceIntegration creates a new service integration
func NewServiceIntegration(config ServiceConfig) (*ServiceIntegration, error) {
	// Get the local IP address
	address, err := getLocalIP()
	if err != nil {
		return nil, fmt.Errorf("failed to get local IP: %w", err)
	}
	
	// Override with environment variable if set
	if envAddr := os.Getenv("SERVICE_ADDRESS"); envAddr != "" {
		address = envAddr
	}
	
	registry := NewServiceRegistry()
	client := NewServiceClient(registry)
	
	instance := &ServiceInstance{
		ID:       config.ServiceID,
		Type:     config.ServiceType,
		Address:  address,
		Port:     config.Port,
		Health:   "healthy",
		Metadata: config.Metadata,
		LastSeen: time.Now(),
	}
	
	integration := &ServiceIntegration{
		registry: registry,
		client:   client,
		instance: instance,
	}
	
	// Register this service
	if err := registry.Register(instance); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}
	
	// Start heartbeat
	integration.startHeartbeat()
	
	return integration, nil
}

// GetRegistry returns the service registry
func (s *ServiceIntegration) GetRegistry() *ServiceRegistry {
	return s.registry
}

// GetClient returns the service client
func (s *ServiceIntegration) GetClient() *ServiceClient {
	return s.client
}

// GetInstance returns this service's instance info
func (s *ServiceIntegration) GetInstance() *ServiceInstance {
	return s.instance
}

// CallAPI makes a call to the API service
func (s *ServiceIntegration) CallAPI(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return s.client.CallServiceJSON(ctx, ServiceTypeAPIServer, method, path, body, result)
}

// CallTCP makes a call to the TCP service
func (s *ServiceIntegration) CallTCP(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return s.client.CallServiceJSON(ctx, ServiceTypeTCPServer, method, path, body, result)
}

// CallWeb makes a call to the Web service
func (s *ServiceIntegration) CallWeb(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return s.client.CallServiceJSON(ctx, ServiceTypeWebServer, method, path, body, result)
}

// UpdateMetadata updates this service's metadata
func (s *ServiceIntegration) UpdateMetadata(metadata map[string]string) {
	s.instance.Metadata = metadata
}

// Shutdown gracefully shuts down the service integration
func (s *ServiceIntegration) Shutdown(ctx context.Context) error {
	log.Printf("🔗 Shutting down service integration for %s", s.instance.ID)
	
	// Stop heartbeat
	if s.heartbeatTicker != nil {
		s.heartbeatTicker.Stop()
	}
	
	// Deregister service
	return s.registry.Deregister(s.instance.ID)
}

// startHeartbeat starts sending periodic heartbeats
func (s *ServiceIntegration) startHeartbeat() {
	s.heartbeatTicker = time.NewTicker(30 * time.Second)
	
	go func() {
		for range s.heartbeatTicker.C {
			if err := s.registry.Heartbeat(s.instance.ID); err != nil {
				log.Printf("❌ Failed to send heartbeat for %s: %v", s.instance.ID, err)
			}
		}
	}()
}

// getLocalIP attempts to get the local IP address
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// DiscoverServices is a helper to discover and log available services
func (s *ServiceIntegration) DiscoverServices() {
	log.Printf("🔍 Discovering available services...")
	
	for _, serviceType := range []ServiceType{ServiceTypeTCPServer, ServiceTypeWebServer, ServiceTypeAPIServer} {
		instances := s.registry.Discover(serviceType)
		if len(instances) > 0 {
			log.Printf("  📡 %s: %d instances", serviceType, len(instances))
			for _, instance := range instances {
				log.Printf("    - %s at %s:%d (%s)", 
					instance.ID, instance.Address, instance.Port, instance.Health)
			}
		}
	}
}

// WaitForService waits for at least one healthy instance of a service type
func (s *ServiceIntegration) WaitForService(ctx context.Context, serviceType ServiceType, timeout time.Duration) error {
	log.Printf("⏳ Waiting for %s service...", serviceType)
	
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		instances := s.registry.Discover(serviceType)
		if len(instances) > 0 {
			log.Printf("✅ Found %s service: %s at %s:%d", 
				serviceType, instances[0].ID, instances[0].Address, instances[0].Port)
			return nil
		}
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
			// Continue waiting
		}
	}
	
	return fmt.Errorf("timeout waiting for %s service", serviceType)
}

// RegisterWithLoadBalancer registers this service with any available load balancers
func (s *ServiceIntegration) RegisterWithLoadBalancer() error {
	instances := s.registry.Discover(ServiceTypeLoadBalancer)
	
	for _, lb := range instances {
		log.Printf("🔗 Registering with load balancer: %s", lb.ID)
		
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		
		err := s.client.CallServiceJSON(ctx, ServiceTypeLoadBalancer, "POST", "/register", s.instance, nil)
		if err != nil {
			log.Printf("⚠️ Failed to register with load balancer %s: %v", lb.ID, err)
		}
	}
	
	return nil
}