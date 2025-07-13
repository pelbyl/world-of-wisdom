package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ServiceClient provides HTTP client functionality for inter-service communication
type ServiceClient struct {
	registry   *ServiceRegistry
	httpClient *http.Client
}

// NewServiceClient creates a new service client
func NewServiceClient(registry *ServiceRegistry) *ServiceClient {
	return &ServiceClient{
		registry: registry,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CallService makes an HTTP request to a service instance
func (c *ServiceClient) CallService(ctx context.Context, serviceType ServiceType, method, path string, body interface{}) (*http.Response, error) {
	// Discover a healthy instance
	instance, err := c.registry.GetHealthyInstance(serviceType)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service %s: %w", serviceType, err)
	}

	// Build URL
	url := fmt.Sprintf("http://%s:%d%s", instance.Address, instance.Port, path)

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "wisdom-service-client/1.0")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Mark service as unhealthy on network errors
		c.registry.UpdateHealth(instance.ID, "unhealthy")
		return nil, fmt.Errorf("request failed to %s: %w", url, err)
	}

	// Update service heartbeat on successful request
	c.registry.Heartbeat(instance.ID)

	return resp, nil
}

// CallServiceJSON makes a JSON request and unmarshals the response
func (c *ServiceClient) CallServiceJSON(ctx context.Context, serviceType ServiceType, method, path string, body interface{}, result interface{}) error {
	resp, err := c.CallService(ctx, serviceType, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service returned error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HealthCheck performs a health check on a service instance
func (c *ServiceClient) HealthCheck(ctx context.Context, instance *ServiceInstance) error {
	url := fmt.Sprintf("http://%s:%d/health", instance.Address, instance.Port)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// Broadcast sends a request to all healthy instances of a service type
func (c *ServiceClient) Broadcast(ctx context.Context, serviceType ServiceType, method, path string, body interface{}) ([]BroadcastResult, error) {
	instances := c.registry.Discover(serviceType)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no healthy instances of service type %s", serviceType)
	}

	results := make([]BroadcastResult, len(instances))

	// Send requests to all instances concurrently
	done := make(chan int, len(instances))

	for i, instance := range instances {
		go func(idx int, inst *ServiceInstance) {
			defer func() { done <- idx }()

			url := fmt.Sprintf("http://%s:%d%s", inst.Address, inst.Port, path)

			var bodyReader io.Reader
			if body != nil {
				bodyBytes, err := json.Marshal(body)
				if err != nil {
					results[idx] = BroadcastResult{
						Instance: inst,
						Error:    err,
					}
					return
				}
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
			if err != nil {
				results[idx] = BroadcastResult{
					Instance: inst,
					Error:    err,
				}
				return
			}

			req.Header.Set("Content-Type", "application/json")

			resp, err := c.httpClient.Do(req)
			if err != nil {
				results[idx] = BroadcastResult{
					Instance: inst,
					Error:    err,
				}
				return
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				results[idx] = BroadcastResult{
					Instance:   inst,
					StatusCode: resp.StatusCode,
					Error:      err,
				}
				return
			}

			results[idx] = BroadcastResult{
				Instance:   inst,
				StatusCode: resp.StatusCode,
				Response:   bodyBytes,
			}
		}(i, instance)
	}

	// Wait for all requests to complete
	for i := 0; i < len(instances); i++ {
		<-done
	}

	return results, nil
}

// BroadcastResult represents the result of a broadcast request to a single instance
type BroadcastResult struct {
	Instance   *ServiceInstance `json:"instance"`
	StatusCode int              `json:"statusCode,omitempty"`
	Response   []byte           `json:"response,omitempty"`
	Error      error            `json:"error,omitempty"`
}
