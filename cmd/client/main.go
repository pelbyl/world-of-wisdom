package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"world-of-wisdom/internal/client"
)

func main() {
	var (
		server   = flag.String("server", "localhost:8080", "Server address")
		attempts = flag.Int("attempts", 1, "Number of quote requests")
		timeout  = flag.Duration("timeout", 30*time.Second, "Request timeout")
	)
	flag.Parse()

	c := client.NewClient(*server, *timeout)

	log.Printf("Connecting to server %s, requesting %d quote(s)", *server, *attempts)

	if *attempts == 1 {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Fatalf("Failed to get quote: %v", err)
		}
		fmt.Printf("\nWord of Wisdom: %s\n", quote)
	} else {
		quotes := c.RequestMultipleQuotes(*attempts)
		fmt.Printf("\nReceived %d quotes from the Word of Wisdom server\n", len(quotes))
	}
}
