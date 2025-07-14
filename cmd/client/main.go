package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"world-of-wisdom/internal/client"
)

// ClientConfig holds configuration for different client types
type ClientConfig struct {
	ClientType        string
	ClientName        string
	SolveDelayMS      int
	MaxAttempts       int
	ConnectionDelayMS int
	AttackMode        bool
	FloodConnections  bool
}

func main() {
	var (
		server   = flag.String("server", getEnv("SERVER_HOST", "server")+":"+getEnv("SERVER_PORT", "8080"), "Server address")
		attempts = flag.Int("attempts", 1, "Number of quote requests")
		timeout  = flag.Duration("timeout", 30*time.Second, "Request timeout")
	)
	flag.Parse()

	// Load configuration from environment variables
	config := loadClientConfig()

	// Override server from environment if provided
	if serverHost := os.Getenv("SERVER_HOST"); serverHost != "" {
		serverPort := os.Getenv("SERVER_PORT")
		if serverPort == "" {
			serverPort = "8080"
		}
		*server = fmt.Sprintf("%s:%s", serverHost, serverPort)
	}

	log.Printf("🚀 Starting %s client (%s)", config.ClientType, config.ClientName)
	log.Printf("📡 Connecting to server %s", *server)
	log.Printf("⚙️ Config: SolveDelay=%dms, MaxAttempts=%d, ConnectionDelay=%dms, AttackMode=%v",
		config.SolveDelayMS, config.MaxAttempts, config.ConnectionDelayMS, config.AttackMode)

	c := client.NewClient(*server, *timeout)

	// Set client behavior based on type
	switch config.ClientType {
	case "fast":
		runFastClient(c, config)
	case "normal":
		runNormalClient(c, config)
	case "slow":
		runSlowClient(c, config)
	case "attacker":
		runAttackerClient(c, config)
	default:
		// Default behavior for legacy usage
		runDefaultClient(c, *attempts)
	}
}

func loadClientConfig() ClientConfig {
	config := ClientConfig{
		ClientType:        getEnv("CLIENT_TYPE", "normal"),
		ClientName:        getEnv("CLIENT_NAME", "default-client"),
		SolveDelayMS:      getEnvInt("SOLVE_DELAY_MS", 1000),
		MaxAttempts:       getEnvInt("MAX_ATTEMPTS", 10),
		ConnectionDelayMS: getEnvInt("CONNECTION_DELAY_MS", 1000),
		AttackMode:        getEnvBool("ATTACK_MODE", false),
		FloodConnections:  getEnvBool("FLOOD_CONNECTIONS", false),
	}
	return config
}

func runFastClient(c *client.Client, config ClientConfig) {
	log.Printf("⚡ Running as FAST client - quick puzzle solving")

	for {
		// Fast clients solve puzzles quickly and make frequent requests
		quote, err := c.RequestQuote()
		if err != nil {
			log.Printf("❌ Failed to get quote: %v", err)
		} else {
			log.Printf("✅ Quote received: %.50s...", quote)
		}

		// Short delay between requests
		time.Sleep(time.Duration(config.SolveDelayMS) * time.Millisecond)

		// Random jitter to avoid synchronization
		jitter := rand.Intn(config.SolveDelayMS / 2)
		time.Sleep(time.Duration(jitter) * time.Millisecond)
	}
}

func runNormalClient(c *client.Client, config ClientConfig) {
	log.Printf("🔄 Running as NORMAL client - standard puzzle solving")

	for {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Printf("❌ Failed to get quote: %v", err)
		} else {
			log.Printf("✅ Quote received: %.50s...", quote)
		}

		// Normal delay between requests
		time.Sleep(time.Duration(config.SolveDelayMS) * time.Millisecond)

		// Random jitter
		jitter := rand.Intn(config.SolveDelayMS)
		time.Sleep(time.Duration(jitter) * time.Millisecond)
	}
}

func runSlowClient(c *client.Client, config ClientConfig) {
	log.Printf("🐌 Running as SLOW client - slower puzzle solving")

	for {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Printf("❌ Failed to get quote: %v", err)
		} else {
			log.Printf("✅ Quote received: %.50s...", quote)
		}

		// Longer delay between requests
		time.Sleep(time.Duration(config.SolveDelayMS) * time.Millisecond)

		// Larger random jitter
		jitter := rand.Intn(config.SolveDelayMS * 2)
		time.Sleep(time.Duration(jitter) * time.Millisecond)
	}
}

func runAttackerClient(c *client.Client, config ClientConfig) {
	log.Printf("🔥 Running as ATTACKER client - attempting to overwhelm server")

	if config.FloodConnections {
		// Flood with connections
		log.Printf("🌊 FLOOD MODE: Creating multiple rapid connections")
		for i := 0; i < config.MaxAttempts; i++ {
			go func(id int) {
				attackerClient := client.NewClient(c.GetServer(), 5*time.Second)
				for {
					// Make rapid requests with minimal delay
					_, err := attackerClient.RequestQuote()
					if err != nil {
						log.Printf("🔥 Attacker-%d failed: %v", id, err)
						// Brief pause before retrying
						time.Sleep(100 * time.Millisecond)
					}

					// Very short delay between attack requests
					time.Sleep(time.Duration(config.SolveDelayMS) * time.Millisecond)
				}
			}(i)
		}

		// Keep main goroutine alive
		select {}
	} else {
		// Single connection rapid requests
		log.Printf("⚡ RAPID MODE: Sending rapid requests on single connection")
		for {
			_, err := c.RequestQuote()
			if err != nil {
				log.Printf("🔥 Attack failed: %v", err)
			}

			// Minimal delay between requests
			time.Sleep(time.Duration(config.SolveDelayMS) * time.Millisecond)
		}
	}
}

func runDefaultClient(c *client.Client, attempts int) {
	log.Printf("📝 Running in DEFAULT mode - legacy behavior")

	if attempts == 1 {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Fatalf("Failed to get quote: %v", err)
		}
		fmt.Printf("\nWord of Wisdom: %s\n", quote)
	} else {
		quotes := c.RequestMultipleQuotes(attempts)
		fmt.Printf("\nReceived %d quotes from the Word of Wisdom server\n", len(quotes))
	}
}

// Helper functions for environment variables
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
