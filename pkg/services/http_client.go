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

// HTTPServiceClient provides HTTP client functionality for service registry communication
type HTTPServiceClient struct {
	registryURL string
	httpClient  *http.Client
}

// NewHTTPServiceClient creates a new HTTP service client for registry communication
func NewHTTPServiceClient(registryURL string) *HTTPServiceClient {
	return &HTTPServiceClient{
		registryURL: registryURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterService registers a service with the registry
func (c *HTTPServiceClient) RegisterService(ctx context.Context, instance *ServiceInstance) error {
	url := fmt.Sprintf("%s/api/v1/services", c.registryURL)
	
	bodyBytes, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal service instance: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}

// DeregisterService removes a service from the registry
func (c *HTTPServiceClient) DeregisterService(ctx context.Context, serviceID string) error {
	url := fmt.Sprintf("%s/api/v1/services/%s", c.registryURL, serviceID)
	
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("service deregistration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}

// DiscoverServices discovers services of a specific type
func (c *HTTPServiceClient) DiscoverServices(serviceType ServiceType) ([]*ServiceInstance, error) {
	url := fmt.Sprintf("%s/api/v1/services/%s", c.registryURL, string(serviceType))
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to discover services: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("service discovery failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	var response struct {
		Instances []*ServiceInstance `json:"instances"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	return response.Instances, nil
}

// SendHeartbeat sends a heartbeat for a service
func (c *HTTPServiceClient) SendHeartbeat(ctx context.Context, serviceID string) error {
	url := fmt.Sprintf("%s/api/v1/services/%s/heartbeat", c.registryURL, serviceID)
	
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("heartbeat failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}

// UpdateHealth updates the health status of a service
func (c *HTTPServiceClient) UpdateHealth(ctx context.Context, serviceID, health string) error {
	url := fmt.Sprintf("%s/api/v1/services/%s/health", c.registryURL, serviceID)
	
	payload := map[string]string{
		"health": health,
	}
	
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal health update: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update health: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("health update failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	
	return nil
}