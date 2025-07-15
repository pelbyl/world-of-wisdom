package client

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"world-of-wisdom/pkg/pow"
)

// SecureClient supports JSON-based secure challenges only
type SecureClient struct {
	*Client
	signingKey     []byte
	supportsSecure bool
	clientID       string
}

// NewSecureClient creates a new client with support for secure challenges
func NewSecureClient(serverAddr string, timeout time.Duration, signingKey []byte, clientID string) *SecureClient {
	return &SecureClient{
		Client:         NewClient(serverAddr, timeout),
		signingKey:     signingKey,
		supportsSecure: true,
		clientID:       clientID,
	}
}

// RequestQuoteSecure attempts to get a quote using secure protocol first, falling back to legacy
func (sc *SecureClient) RequestQuoteSecure() (string, error) {
	return sc.requestQuoteWithRetry(sc.maxRetries)
}

// Override the base client's attemptRequestQuote to handle both formats
func (sc *SecureClient) attemptRequestQuote() (string, error) {
	conn, err := net.DialTimeout("tcp", sc.serverAddr, sc.timeout)
	if err != nil {
		return "", fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(sc.timeout))

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to receive challenge from server")
	}

	challengeData := scanner.Bytes()
	log.Printf("Received challenge data: %d bytes", len(challengeData))

	// Auto-detect format and handle accordingly
	format := sc.encoder.AutoDetectFormat(challengeData)
	log.Printf("Detected challenge format: %s", format)
	
	return sc.handleSecureChallenge(conn, challengeData, format, scanner)
}

// handleSecureChallenge processes secure challenges in any supported format
func (sc *SecureClient) handleSecureChallenge(conn net.Conn, challengeData []byte, format pow.ChallengeFormat, scanner *bufio.Scanner) (string, error) {
	// Decode challenge using detected format
	challenge, err := sc.encoder.Decode(challengeData, format, sc.clientID)
	if err != nil {
		return "", fmt.Errorf("failed to decode %s challenge: %w", format, err)
	}

	// Client ID is already set by decoder if needed

	log.Printf("Parsed secure challenge: Algorithm=%s, Difficulty=%d, ExpiresAt=%d", 
		challenge.Algorithm, challenge.Difficulty, challenge.ExpiresAt)

	// Validate challenge if we have signing key
	if sc.signingKey != nil {
		if err := challenge.IsValid(sc.signingKey); err != nil {
			return "", fmt.Errorf("invalid secure challenge: %w", err)
		}
		log.Printf("Challenge signature validated successfully")
	}

	// Solve the challenge
	start := time.Now()
	solution, err := pow.SolveSecureChallenge(challenge, sc.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to solve secure challenge: %w", err)
	}
	elapsed := time.Since(start)

	log.Printf("Solved secure challenge in %v, sending solution: %s", elapsed, solution)

	// Send solution
	_, err = conn.Write([]byte(solution + "\n"))
	if err != nil {
		return "", fmt.Errorf("failed to send solution: %w", err)
	}

	// Read response
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to receive response from server")
	}

	response := scanner.Text()
	if strings.HasPrefix(response, "Error:") {
		return "", fmt.Errorf("server error: %s", response)
	}

	return response, nil
}

// Legacy challenge handling removed - only JSON format supported

// SupportsSecureChallenges returns true if client supports secure challenges
func (sc *SecureClient) SupportsSecureChallenges() bool {
	return sc.supportsSecure && sc.signingKey != nil
}

// SetSigningKey updates the signing key for challenge verification
func (sc *SecureClient) SetSigningKey(key []byte) {
	sc.signingKey = key
}

// GetClientID returns the client ID
func (sc *SecureClient) GetClientID() string {
	return sc.clientID
}

// SetClientID updates the client ID
func (sc *SecureClient) SetClientID(id string) {
	sc.clientID = id
}

// RequestMultipleQuotesSecure gets multiple quotes using secure protocol
func (sc *SecureClient) RequestMultipleQuotesSecure(count int) []string {
	quotes := make([]string, 0, count)

	for i := 0; i < count; i++ {
		quote, err := sc.RequestQuoteSecure()
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

// BenchmarkSolveTime measures the time to solve challenges
func (sc *SecureClient) BenchmarkSolveTime(iterations int) (time.Duration, error) {
	total := time.Duration(0)
	successful := 0

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := sc.RequestQuoteSecure()
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("Benchmark iteration %d failed: %v", i+1, err)
			continue
		}

		total += elapsed
		successful++
	}

	if successful == 0 {
		return 0, fmt.Errorf("no successful iterations")
	}

	return total / time.Duration(successful), nil
}

// TestConnectivity tests if the client can connect to the server
func (sc *SecureClient) TestConnectivity() error {
	conn, err := net.DialTimeout("tcp", sc.serverAddr, sc.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	log.Printf("Successfully connected to server at %s", sc.serverAddr)
	return nil
}

// ClientStats holds statistics about client performance
type ClientStats struct {
	TotalRequests     int           `json:"total_requests"`
	SuccessfulRequests int           `json:"successful_requests"`
	FailedRequests    int           `json:"failed_requests"`
	AverageSolveTime  time.Duration `json:"average_solve_time"`
	SecureChallenges  int           `json:"secure_challenges"`
	LegacyChallenges  int           `json:"legacy_challenges"`
}

// GetStats returns client performance statistics
func (sc *SecureClient) GetStats() ClientStats {
	// This would be implemented with proper statistics collection
	return ClientStats{
		TotalRequests:     0,
		SuccessfulRequests: 0,
		FailedRequests:    0,
		AverageSolveTime:  0,
		SecureChallenges:  0,
		LegacyChallenges:  0,
	}
}