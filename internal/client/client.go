package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"regexp"
	"strings"
	"time"

	"world-of-wisdom/pkg/pow"
)

type Client struct {
	serverAddr string
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration
}

func NewClient(serverAddr string, timeout time.Duration) *Client {
	return &Client{
		serverAddr: serverAddr,
		timeout:    timeout,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
	}
}

func (c *Client) GetServer() string {
	return c.serverAddr
}

func (c *Client) RequestQuote() (string, error) {
	return c.requestQuoteWithRetry(c.maxRetries)
}

func (c *Client) requestQuoteWithRetry(retriesLeft int) (string, error) {
	quote, err := c.attemptRequestQuote()
	if err != nil {
		if retriesLeft > 0 {
			log.Printf("Request failed: %v. Retrying in %v... (%d retries left)", err, c.retryDelay, retriesLeft)
			time.Sleep(c.retryDelay)
			return c.requestQuoteWithRetry(retriesLeft - 1)
		}
		return "", fmt.Errorf("failed after %d retries: %w", c.maxRetries, err)
	}
	return quote, nil
}

func (c *Client) attemptRequestQuote() (string, error) {
	conn, err := net.DialTimeout("tcp", c.serverAddr, c.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(c.timeout))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to receive challenge from server")
	}

	challengeStr := scanner.Text()
	log.Printf("Received challenge: %s", challengeStr)

	// Determine algorithm based on challenge format
	var solution string
	start := time.Now()

	if strings.Contains(challengeStr, "Argon2") {
		// Parse Argon2 challenge
		seed, difficulty, err := parseArgon2Challenge(challengeStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse Argon2 challenge: %w", err)
		}

		challenge, err := pow.GenerateArgon2Challenge(difficulty)
		if err != nil {
			return "", fmt.Errorf("failed to create Argon2 challenge: %w", err)
		}
		challenge.Seed = seed

		solution, err = pow.SolveArgon2Challenge(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to solve Argon2 challenge: %w", err)
		}
	} else {
		// Parse SHA-256 challenge
		seed, difficulty, err := parseChallenge(challengeStr)
		if err != nil {
			return "", fmt.Errorf("failed to parse challenge: %w", err)
		}

		challenge := &pow.Challenge{
			Seed:       seed,
			Difficulty: difficulty,
		}

		solution, err = pow.SolveChallenge(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to solve challenge: %w", err)
		}
	}
	elapsed := time.Since(start)

	log.Printf("Solved challenge in %v, sending solution: %s", elapsed, solution)

	_, err = conn.Write([]byte(solution + "\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send solution: %w", err)
	}

	if !scanner.Scan() {
		return "", fmt.Errorf("failed to receive response from server")
	}

	response := scanner.Text()

	if strings.HasPrefix(response, "Error:") {
		return "", fmt.Errorf("server error: %s", response)
	}

	return response, nil
}

// SetRetryConfig allows customizing retry behavior
func (c *Client) SetRetryConfig(maxRetries int, retryDelay time.Duration) {
	c.maxRetries = maxRetries
	c.retryDelay = retryDelay
}

func parseChallenge(challenge string) (seed string, difficulty int, err error) {
	re := regexp.MustCompile(`Solve PoW: ([a-f0-9]+) with prefix (0+)`)
	matches := re.FindStringSubmatch(challenge)

	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid challenge format")
	}

	seed = matches[1]
	difficulty = len(matches[2])

	return seed, difficulty, nil
}

func parseArgon2Challenge(challenge string) (seed string, difficulty int, err error) {
	re := regexp.MustCompile(`Solve Argon2 PoW: ([a-f0-9]+) with (\d+) leading zeros`)
	matches := re.FindStringSubmatch(challenge)

	if len(matches) != 3 {
		return "", 0, fmt.Errorf("invalid Argon2 challenge format")
	}

	seed = matches[1]
	_, err = fmt.Sscanf(matches[2], "%d", &difficulty)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse difficulty: %w", err)
	}

	return seed, difficulty, nil
}

func (c *Client) RequestMultipleQuotes(count int) []string {
	quotes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Printf("Failed to get quote %d/%d: %v", i+1, count, err)
			continue
		}
		quotes = append(quotes, quote)
		log.Printf("Quote %d/%d: %s", i+1, count, quote)

		if i < count-1 {
			time.Sleep(time.Second)
		}
	}

	return quotes
}
