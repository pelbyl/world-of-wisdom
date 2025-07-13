package pow

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Challenge represents a challenge using Argon2
type Argon2Challenge struct {
	Seed       string
	Difficulty int
	// Argon2 parameters
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

// GenerateArgon2Challenge creates a new Argon2-based challenge
func GenerateArgon2Challenge(difficulty int) (*Argon2Challenge, error) {
	if difficulty < 1 || difficulty > 6 {
		return nil, fmt.Errorf("difficulty must be between 1 and 6, got %d", difficulty)
	}

	seedBytes := make([]byte, 16)
	if _, err := rand.Read(seedBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random seed: %w", err)
	}

	// Scale Argon2 parameters based on difficulty
	// Higher difficulty = more memory and iterations
	memory := uint32(64 * 1024) // 64 MB base
	time := uint32(1)           // 1 iteration base
	
	switch difficulty {
	case 1:
		memory = 32 * 1024  // 32 MB
		time = 1
	case 2:
		memory = 64 * 1024  // 64 MB
		time = 1
	case 3:
		memory = 128 * 1024 // 128 MB
		time = 2
	case 4:
		memory = 256 * 1024 // 256 MB
		time = 2
	case 5:
		memory = 512 * 1024 // 512 MB
		time = 3
	case 6:
		memory = 1024 * 1024 // 1 GB
		time = 3
	}

	return &Argon2Challenge{
		Seed:       hex.EncodeToString(seedBytes),
		Difficulty: difficulty,
		Time:       time,
		Memory:     memory,
		Threads:    4,
		KeyLen:     32,
	}, nil
}

func (c *Argon2Challenge) String() string {
	return fmt.Sprintf("Solve Argon2 PoW: %s with %d leading zeros", c.Seed, c.Difficulty)
}

// VerifyArgon2PoW verifies an Argon2-based proof of work
func VerifyArgon2PoW(challenge *Argon2Challenge, nonce string) bool {
	if challenge.Difficulty < 1 || challenge.Difficulty > 6 {
		return false
	}

	// Combine seed and nonce
	data := []byte(challenge.Seed + nonce)
	
	// Use empty salt for PoW (deterministic)
	salt := []byte{}
	
	// Generate Argon2 hash
	hash := argon2.IDKey(data, salt, challenge.Time, challenge.Memory, challenge.Threads, challenge.KeyLen)
	hashHex := hex.EncodeToString(hash)

	// Check for required number of leading zeros
	requiredPrefix := strings.Repeat("0", challenge.Difficulty)
	return strings.HasPrefix(hashHex, requiredPrefix)
}

// SolveArgon2Challenge attempts to solve an Argon2 challenge
func SolveArgon2Challenge(challenge *Argon2Challenge) (string, error) {
	for nonce := 0; ; nonce++ {
		nonceStr := strconv.Itoa(nonce)
		if VerifyArgon2PoW(challenge, nonceStr) {
			return nonceStr, nil
		}
		
		// Increase max attempts for higher difficulties
		maxAttempts := 1000000 * challenge.Difficulty
		if nonce > maxAttempts {
			return "", fmt.Errorf("solution not found after %d attempts", nonce)
		}
	}
}

// Legacy compatibility functions to maintain backward compatibility
func GenerateChallengeWithAlgorithm(difficulty int, algorithm string) (interface{}, error) {
	switch algorithm {
	case "sha256":
		return GenerateChallenge(difficulty)
	case "argon2":
		return GenerateArgon2Challenge(difficulty)
	default:
		return nil, fmt.Errorf("unknown algorithm: %s", algorithm)
	}
}