package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ServiceType represents different service types in the system
type ServiceType string

const (
	ServiceTypeTCP       ServiceType = "tcp-server"
	ServiceTypeWeb       ServiceType = "web-server" 
	ServiceTypeAPI       ServiceType = "api-server"
	ServiceTypeLoadBalancer ServiceType = "load-balancer"
)

// ServiceInstance represents a running service instance
type ServiceInstance struct {
	ID       string            `json:"id"`
	Type     ServiceType       `json:"type"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Health   string            `json:"health"` // healthy, unhealthy, unknown
	Metadata map[string]string `json:"metadata,omitempty"`
	LastSeen time.Time         `json:"lastSeen"`
}

// ServiceRegistry manages service discovery and health checking
type ServiceRegistry struct {
	mu        sync.RWMutex
	services  map[string]*ServiceInstance
	listeners map[ServiceType][]chan ServiceEvent
}

// ServiceEvent represents service lifecycle events
type ServiceEvent struct {
	Type     string           `json:"type"` // registered, deregistered, health_changed
	Instance *ServiceInstance `json:"instance"`
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	registry := &ServiceRegistry{
		services:  make(map[string]*ServiceInstance),
		listeners: make(map[ServiceType][]chan ServiceEvent),
	}
	
	// Start health check routine
	go registry.healthCheckRoutine()
	
	return registry
}

// Register adds a service instance to the registry
func (r *ServiceRegistry) Register(instance *ServiceInstance) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instance.LastSeen = time.Now()
	instance.Health = "healthy"
	
	r.services[instance.ID] = instance
	
	log.Printf("🔗 Service registered: %s (%s) at %s:%d", 
		instance.ID, instance.Type, instance.Address, instance.Port)
	
	// Notify listeners
	r.notifyListeners(instance.Type, ServiceEvent{
		Type:     "registered",
		Instance: instance,
	})
	
	return nil
}

// Deregister removes a service instance from the registry
func (r *ServiceRegistry) Deregister(serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if instance, exists := r.services[serviceID]; exists {
		delete(r.services, serviceID)
		
		log.Printf("🔗 Service deregistered: %s (%s)", serviceID, instance.Type)
		
		// Notify listeners
		r.notifyListeners(instance.Type, ServiceEvent{
			Type:     "deregistered", 
			Instance: instance,
		})
	}
	
	return nil
}

// Discover returns all healthy instances of a specific service type
func (r *ServiceRegistry) Discover(serviceType ServiceType) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var instances []*ServiceInstance
	for _, instance := range r.services {
		if instance.Type == serviceType && instance.Health == "healthy" {
			instances = append(instances, instance)
		}
	}
	
	return instances
}

// GetService returns a specific service instance by ID
func (r *ServiceRegistry) GetService(serviceID string) (*ServiceInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	instance, exists := r.services[serviceID]
	return instance, exists
}

// ListAll returns all registered services
func (r *ServiceRegistry) ListAll() map[string]*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a copy to avoid concurrent map access
	result := make(map[string]*ServiceInstance)
	for id, instance := range r.services {
		result[id] = instance
	}
	
	return result
}

// Subscribe allows services to listen for events of specific service types
func (r *ServiceRegistry) Subscribe(serviceType ServiceType) <-chan ServiceEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	eventChan := make(chan ServiceEvent, 10)
	r.listeners[serviceType] = append(r.listeners[serviceType], eventChan)
	
	return eventChan
}

// UpdateHealth updates the health status of a service
func (r *ServiceRegistry) UpdateHealth(serviceID, health string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if instance, exists := r.services[serviceID]; exists {
		oldHealth := instance.Health
		instance.Health = health
		instance.LastSeen = time.Now()
		
		if oldHealth != health {
			log.Printf("🏥 Service health changed: %s (%s) %s -> %s", 
				serviceID, instance.Type, oldHealth, health)
			
			// Notify listeners
			r.notifyListeners(instance.Type, ServiceEvent{
				Type:     "health_changed",
				Instance: instance,
			})
		}
		
		return nil
	}
	
	return fmt.Errorf("service %s not found", serviceID)
}

// Heartbeat updates the last seen time for a service
func (r *ServiceRegistry) Heartbeat(serviceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if instance, exists := r.services[serviceID]; exists {
		instance.LastSeen = time.Now()
		return nil
	}
	
	return fmt.Errorf("service %s not found", serviceID)
}

// notifyListeners sends events to all listeners of a service type
func (r *ServiceRegistry) notifyListeners(serviceType ServiceType, event ServiceEvent) {
	if listeners, exists := r.listeners[serviceType]; exists {
		for _, listener := range listeners {
			select {
			case listener <- event:
			default:
				// Channel full, skip
			}
		}
	}
}

// healthCheckRoutine periodically checks service health
func (r *ServiceRegistry) healthCheckRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		r.performHealthChecks()
	}
}

// performHealthChecks checks all services and marks stale ones as unhealthy
func (r *ServiceRegistry) performHealthChecks() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	now := time.Now()
	staleThreshold := 2 * time.Minute
	
	for serviceID, instance := range r.services {
		if now.Sub(instance.LastSeen) > staleThreshold && instance.Health == "healthy" {
			instance.Health = "unhealthy"
			
			log.Printf("🏥 Service marked unhealthy due to stale heartbeat: %s (%s)", 
				serviceID, instance.Type)
			
			// Notify listeners
			r.notifyListeners(instance.Type, ServiceEvent{
				Type:     "health_changed",
				Instance: instance,
			})
		}
	}
}

// GetHealthyInstance returns a single healthy instance of a service type (load balancing)
func (r *ServiceRegistry) GetHealthyInstance(serviceType ServiceType) (*ServiceInstance, error) {
	instances := r.Discover(serviceType)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no healthy instances of service type %s", serviceType)
	}
	
	// Simple round-robin for now
	return instances[0], nil
}