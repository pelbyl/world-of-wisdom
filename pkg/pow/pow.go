package pow

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type Challenge struct {
	Seed       string
	Difficulty int
}

func GenerateChallenge(difficulty int) (*Challenge, error) {
	if difficulty < 1 || difficulty > 6 {
		return nil, fmt.Errorf("difficulty must be between 1 and 6, got %d", difficulty)
	}

	seedBytes := make([]byte, 16)
	if _, err := rand.Read(seedBytes); err != nil {
		return nil, fmt.Errorf("failed to generate random seed: %w", err)
	}

	return &Challenge{
		Seed:       hex.EncodeToString(seedBytes),
		Difficulty: difficulty,
	}, nil
}

func (c *Challenge) String() string {
	return fmt.Sprintf("Solve PoW: %s with prefix %s", c.Seed, strings.Repeat("0", c.Difficulty))
}

func VerifyPoW(seed, nonce string, difficulty int) bool {
	if difficulty < 1 || difficulty > 6 {
		return false
	}

	data := seed + nonce
	hash := sha256.Sum256([]byte(data))
	hashHex := hex.EncodeToString(hash[:])

	requiredPrefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hashHex, requiredPrefix)
}

func SolveChallenge(challenge *Challenge) (string, error) {
	for nonce := 0; ; nonce++ {
		nonceStr := strconv.Itoa(nonce)
		if VerifyPoW(challenge.Seed, nonceStr, challenge.Difficulty) {
			return nonceStr, nil
		}
		
		if nonce > 100000000 {
			return "", fmt.Errorf("solution not found after %d attempts", nonce)
		}
	}
}