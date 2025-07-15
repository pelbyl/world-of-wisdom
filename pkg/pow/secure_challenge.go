package pow

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SecureChallenge represents an enhanced challenge with HMAC signature and time-based expiration
type SecureChallenge struct {
	// Core challenge data
	Version    uint8  `json:"v"`           // Protocol version
	Seed       string `json:"seed"`        
	Difficulty int    `json:"difficulty"`
	Algorithm  string `json:"algorithm"`   // "argon2" or "sha256"
	
	// Argon2 specific parameters (when algorithm="argon2")
	Argon2Params *Argon2Params `json:"argon2_params,omitempty"`
	
	// Security metadata
	ClientID   string `json:"client_id"`   // Track per-client
	Timestamp  int64  `json:"timestamp"`
	ExpiresAt  int64  `json:"expires_at"`
	Nonce      string `json:"nonce"`       // Prevent replay
	
	// Signature (always last for easy parsing)
	Signature  string `json:"signature"`   // Base64 encoded HMAC
}

// Argon2Params holds Argon2 specific parameters
type Argon2Params struct {
	Time      uint32 `json:"t"`
	Memory    uint32 `json:"m"`
	Threads   uint8  `json:"p"`
	KeyLength uint32 `json:"l"`
}

// HMACSignature handles HMAC signing and verification
type HMACSignature struct {
	key []byte
}

// NewHMACSignature creates a new HMAC signature instance
func NewHMACSignature(key []byte) *HMACSignature {
	return &HMACSignature{key: key}
}

// Sign creates an HMAC signature for the given data
func (s *HMACSignature) Sign(data []byte) []byte {
	h := hmac.New(sha256.New, s.key)
	h.Write(data)
	return h.Sum(nil)
}

// Verify validates an HMAC signature
func (s *HMACSignature) Verify(data, signature []byte) bool {
	expected := s.Sign(data)
	return hmac.Equal(expected, signature)
}

// GenerateSecureChallenge creates a new secure challenge with HMAC signature
func GenerateSecureChallenge(difficulty int, algorithm string, clientID string, signingKey []byte) (*SecureChallenge, error) {
	if difficulty < 1 || difficulty > 6 {
		return nil, fmt.Errorf("difficulty must be between 1 and 6, got %d", difficulty)
	}

	// Generate random seed
	seedBytes := make([]byte, 16)
	if _, err := rand.Read(seedBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random seed: %w", err)
	}

	// Generate random nonce for replay prevention
	nonceBytes := make([]byte, 8)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random nonce: %w", err)
	}

	now := time.Now()
	expiresAt := now.Add(5 * time.Minute)

	challenge := &SecureChallenge{
		Version:    1,
		Seed:       hex.EncodeToString(seedBytes),
		Difficulty: difficulty,
		Algorithm:  algorithm,
		ClientID:   clientID,
		Timestamp:  now.UnixMicro(),
		ExpiresAt:  expiresAt.UnixMicro(),
		Nonce:      hex.EncodeToString(nonceBytes),
	}

	// Set Argon2 parameters if needed
	if algorithm == "argon2" {
		challenge.Argon2Params = &Argon2Params{
			Time:      1,
			Memory:    64 * 1024, // 64MB
			Threads:   4,
			KeyLength: 32,
		}
	}

	// Create signature
	if err := challenge.Sign(signingKey); err != nil {
		return nil, fmt.Errorf("failed to sign challenge: %w", err)
	}

	return challenge, nil
}

// Sign creates an HMAC signature for the challenge
func (c *SecureChallenge) Sign(key []byte) error {
	// Create a copy without signature for signing
	temp := *c
	temp.Signature = ""
	
	// Marshal to JSON for consistent signing
	data, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("failed to marshal challenge for signing: %w", err)
	}

	// Create HMAC signature
	signer := NewHMACSignature(key)
	signature := signer.Sign(data)
	c.Signature = base64.StdEncoding.EncodeToString(signature)

	return nil
}

// Verify validates the challenge's HMAC signature
func (c *SecureChallenge) Verify(key []byte) error {
	if c.Signature == "" {
		return fmt.Errorf("challenge has no signature")
	}

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Create a copy without signature for verification
	temp := *c
	temp.Signature = ""
	
	// Marshal to JSON for consistent verification
	data, err := json.Marshal(temp)
	if err != nil {
		return fmt.Errorf("failed to marshal challenge for verification: %w", err)
	}

	// Verify HMAC signature
	signer := NewHMACSignature(key)
	if !signer.Verify(data, signature) {
		return fmt.Errorf("invalid HMAC signature")
	}

	return nil
}

// IsExpired checks if the challenge has expired
func (c *SecureChallenge) IsExpired() bool {
	return time.Now().UnixMicro() > c.ExpiresAt
}

// IsValid performs comprehensive validation of the challenge
func (c *SecureChallenge) IsValid(key []byte) error {
	// Check version
	if c.Version != 1 {
		return fmt.Errorf("unsupported challenge version: %d", c.Version)
	}

	// Check algorithm
	if c.Algorithm != "sha256" && c.Algorithm != "argon2" {
		return fmt.Errorf("unsupported algorithm: %s", c.Algorithm)
	}

	// Check difficulty
	if c.Difficulty < 1 || c.Difficulty > 6 {
		return fmt.Errorf("invalid difficulty: %d", c.Difficulty)
	}

	// Check expiration
	if c.IsExpired() {
		return fmt.Errorf("challenge has expired")
	}

	// Verify signature
	if err := c.Verify(key); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// String returns a human-readable representation of the challenge
func (c *SecureChallenge) String() string {
	var prefix string
	if c.Algorithm == "sha256" {
		prefix = strings.Repeat("0", c.Difficulty)
		return fmt.Sprintf("Solve PoW: %s with prefix %s (expires: %s)", 
			c.Seed, prefix, time.UnixMicro(c.ExpiresAt).Format(time.RFC3339))
	}
	
	return fmt.Sprintf("Solve Argon2 PoW: %s (difficulty: %d, expires: %s)", 
		c.Seed, c.Difficulty, time.UnixMicro(c.ExpiresAt).Format(time.RFC3339))
}

// VerifySecurePoW validates a proof-of-work solution for a secure challenge
func VerifySecurePoW(challenge *SecureChallenge, solution string, signingKey []byte) error {
	// First validate the challenge itself
	if err := challenge.IsValid(signingKey); err != nil {
		return fmt.Errorf("invalid challenge: %w", err)
	}

	// Verify the proof-of-work based on algorithm
	switch challenge.Algorithm {
	case "sha256":
		if !VerifyPoW(challenge.Seed, solution, challenge.Difficulty) {
			return fmt.Errorf("invalid SHA-256 proof-of-work")
		}
	case "argon2":
		if challenge.Argon2Params == nil {
			return fmt.Errorf("missing Argon2 parameters")
		}
		
		argon2Challenge := &Argon2Challenge{
			Seed:       challenge.Seed,
			Difficulty: challenge.Difficulty,
			Time:       challenge.Argon2Params.Time,
			Memory:     challenge.Argon2Params.Memory,
			Threads:    challenge.Argon2Params.Threads,
			KeyLen:     challenge.Argon2Params.KeyLength,
		}
		
		if !VerifyArgon2PoW(argon2Challenge, solution) {
			return fmt.Errorf("invalid Argon2 proof-of-work")
		}
	default:
		return fmt.Errorf("unsupported algorithm: %s", challenge.Algorithm)
	}

	return nil
}

// SolveSecureChallenge attempts to solve a secure challenge
func SolveSecureChallenge(challenge *SecureChallenge, signingKey []byte) (string, error) {
	// Validate challenge first
	if err := challenge.IsValid(signingKey); err != nil {
		return "", fmt.Errorf("invalid challenge: %w", err)
	}

	// Solve based on algorithm
	switch challenge.Algorithm {
	case "sha256":
		// Use existing SHA-256 solver
		basicChallenge := &Challenge{
			Seed:       challenge.Seed,
			Difficulty: challenge.Difficulty,
		}
		return SolveChallenge(basicChallenge)
		
	case "argon2":
		if challenge.Argon2Params == nil {
			return "", fmt.Errorf("missing Argon2 parameters")
		}
		
		argon2Challenge := &Argon2Challenge{
			Seed:       challenge.Seed,
			Difficulty: challenge.Difficulty,
			Time:       challenge.Argon2Params.Time,
			Memory:     challenge.Argon2Params.Memory,
			Threads:    challenge.Argon2Params.Threads,
			KeyLen:     challenge.Argon2Params.KeyLength,
		}
		
		return SolveArgon2Challenge(argon2Challenge)
		
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", challenge.Algorithm)
	}
}