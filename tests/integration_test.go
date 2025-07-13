package tests

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"world-of-wisdom/internal/client"
	"world-of-wisdom/internal/server"
)

func TestFullIntegration(t *testing.T) {
	// Start test server
	cfg := server.Config{
		Port:         ":0", // Let OS choose port
		Difficulty:   2,
		Timeout:      10 * time.Second,
		AdaptiveMode: true,
		MetricsPort:  ":0", // Let OS choose port
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Get the actual port assigned
	serverAddr := srv.Addr()

	go func() {
		if err := srv.Start(); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Test client connection and PoW solving
	c := client.NewClient(serverAddr, 5*time.Second)

	quote, err := c.RequestQuote()
	if err != nil {
		t.Fatalf("Failed to get quote: %v", err)
	}

	if quote == "" {
		t.Error("Received empty quote")
	}

	if !strings.Contains(quote, "-") {
		t.Error("Quote doesn't contain author attribution")
	}

	// Test multiple requests
	quotes := c.RequestMultipleQuotes(3)
	if len(quotes) != 3 {
		t.Errorf("Expected 3 quotes, got %d", len(quotes))
	}

	// Cleanup
	srv.Shutdown()
}

func TestDifficultyAdaptation(t *testing.T) {
	cfg := server.Config{
		Port:         ":0",
		Difficulty:   1,
		Timeout:      10 * time.Second,
		AdaptiveMode: true,
		MetricsPort:  "",
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	serverAddr := srv.Addr()

	go func() {
		srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Rapid requests to trigger difficulty adjustment
	c := client.NewClient(serverAddr, 5*time.Second)

	for i := 0; i < 15; i++ {
		_, err := c.RequestQuote()
		if err != nil {
			t.Logf("Request %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	stats := srv.GetStats()
	if stats["adaptive_mode"].(bool) != true {
		t.Error("Adaptive mode should be enabled")
	}

	srv.Shutdown()
}

func TestMetricsEndpoint(t *testing.T) {
	cfg := server.Config{
		Port:         ":0",
		Difficulty:   2,
		Timeout:      10 * time.Second,
		AdaptiveMode: false,
		MetricsPort:  ":9091", // Different port to avoid conflicts
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	serverAddr := srv.Addr()

	go func() {
		srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Make some requests to generate metrics
	c := client.NewClient(serverAddr, 5*time.Second)
	for i := 0; i < 3; i++ {
		c.RequestQuote()
	}

	// Check metrics endpoint
	resp, err := http.Get("http://localhost:9091/metrics")
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check for specific metrics
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}
	content := string(body)

	expectedMetrics := []string{
		"wisdom_connections_total",
		"wisdom_puzzles_solved_total",
		"wisdom_current_difficulty",
		"wisdom_solve_time_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(content, metric) {
			t.Errorf("Metrics endpoint missing: %s", metric)
		}
	}

	srv.Shutdown()
}

func TestConcurrentClients(t *testing.T) {
	cfg := server.Config{
		Port:         ":0",
		Difficulty:   2,
		Timeout:      10 * time.Second,
		AdaptiveMode: true,
		MetricsPort:  "",
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	serverAddr := srv.Addr()

	go func() {
		srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Test concurrent client connections
	results := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func() {
			c := client.NewClient(serverAddr, 5*time.Second)
			_, err := c.RequestQuote()
			results <- err
		}()
	}

	// Collect results
	for i := 0; i < 10; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent client %d failed: %v", i, err)
		}
	}

	srv.Shutdown()
}

func TestInvalidPoWRejection(t *testing.T) {
	cfg := server.Config{
		Port:         ":0",
		Difficulty:   2,
		Timeout:      10 * time.Second,
		AdaptiveMode: false,
		MetricsPort:  "",
	}

	srv, err := server.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	serverAddr := srv.Addr()

	go func() {
		srv.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	// Test direct TCP connection with invalid PoW
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Read challenge
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read challenge: %v", err)
	}

	challenge := string(buf[:n])
	if !strings.Contains(challenge, "Solve PoW:") {
		t.Errorf("Invalid challenge format: %s", challenge)
	}

	// Send invalid solution
	_, err = conn.Write([]byte("invalid_nonce\n"))
	if err != nil {
		t.Fatalf("Failed to send response: %v", err)
	}

	// Read response
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	response := string(buf[:n])
	if !strings.Contains(response, "Error:") {
		t.Errorf("Expected error for invalid PoW, got: %s", response)
	}

	srv.Shutdown()
}
