package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"world-of-wisdom/pkg/logger"
	"world-of-wisdom/pkg/pow"
)

type Client struct {
	serverAddr string
	timeout    time.Duration
	maxRetries int
	retryDelay time.Duration
	encoder    *pow.ChallengeEncoder
}

func NewClient(serverAddr string, timeout time.Duration) *Client {
	return &Client{
		serverAddr: serverAddr,
		timeout:    timeout,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
		encoder:    pow.NewChallengeEncoder(pow.FormatBinary), // Default to binary
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

	challengeData := scanner.Bytes()
	log.Printf("Received challenge data: %d bytes", len(challengeData))

	// Auto-detect format and decode challenge
	format := c.encoder.AutoDetectFormat(challengeData)
	log.Printf("Detected challenge format: %s", format)
	
	secureChallenge, err := c.encoder.Decode(challengeData, format, "")
	if err != nil {
		return "", fmt.Errorf("failed to decode %s challenge: %w", format, err)
	}

	log.Printf("Decoded secure challenge: Algorithm=%s, Difficulty=%d, ExpiresAt=%d", 
		secureChallenge.Algorithm, secureChallenge.Difficulty, secureChallenge.ExpiresAt)

	// Solve the challenge
	var solution string
	start := time.Now()

	if secureChallenge.Algorithm == "sha256" {
		// Solve SHA-256 challenge
		challenge := &pow.Challenge{
			Seed:       secureChallenge.Seed,
			Difficulty: secureChallenge.Difficulty,
		}
		solution, err = pow.SolveChallenge(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to solve SHA-256 challenge: %w", err)
		}
	} else if secureChallenge.Algorithm == "argon2" {
		// Solve Argon2 challenge
		challenge := &pow.Argon2Challenge{
			Seed:       secureChallenge.Seed,
			Difficulty: secureChallenge.Difficulty,
		}
		if secureChallenge.Argon2Params != nil {
			challenge.Time = secureChallenge.Argon2Params.Time
			challenge.Memory = secureChallenge.Argon2Params.Memory
			challenge.Threads = secureChallenge.Argon2Params.Threads
			challenge.KeyLen = secureChallenge.Argon2Params.KeyLength
		}
		solution, err = pow.SolveArgon2Challenge(challenge)
		if err != nil {
			return "", fmt.Errorf("failed to solve Argon2 challenge: %w", err)
		}
	} else {
		return "", fmt.Errorf("unsupported algorithm: %s", secureChallenge.Algorithm)
	}
	elapsed := time.Since(start)

	log.Printf("Solved challenge in %v, sending solution: %s", elapsed, logger.MaskSensitive(solution))

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

// Legacy parsing functions removed - only JSON format supported

func (c *Client) RequestMultipleQuotes(count int) []string {
	quotes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		quote, err := c.RequestQuote()
		if err != nil {
			log.Printf("Failed to get quote %d/%d: %v", i+1, count, err)
			continue
		}
		quotes = append(quotes, quote)
		log.Printf("Quote %d/%d: %s", i+1, count, logger.SanitizeForLog(quote))

		if i < count-1 {
			time.Sleep(time.Second)
		}
	}

	return quotes
}
