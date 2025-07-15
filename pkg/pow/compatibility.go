package pow

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ChallengeCompatibility provides backward compatibility between legacy and secure challenges
type ChallengeCompatibility struct {
	signingKey         []byte
	defaultAlgorithm   string
	defaultDifficulty  int
	enableSecureMode   bool
	enableLegacyMode   bool
}

// NewChallengeCompatibility creates a new compatibility layer
func NewChallengeCompatibility(signingKey []byte, defaultAlgorithm string, defaultDifficulty int) *ChallengeCompatibility {
	return &ChallengeCompatibility{
		signingKey:         signingKey,
		defaultAlgorithm:   defaultAlgorithm,
		defaultDifficulty:  defaultDifficulty,
		enableSecureMode:   true,
		enableLegacyMode:   true,
	}
}

// GenerateCompatibleChallenge generates a challenge in the requested format
func (cc *ChallengeCompatibility) GenerateCompatibleChallenge(clientID string, difficulty int, algorithm string, preferSecure bool) (string, error) {
	if preferSecure && cc.enableSecureMode {
		return cc.generateSecureChallenge(clientID, difficulty, algorithm)
	}
	
	if cc.enableLegacyMode {
		return cc.generateLegacyChallenge(difficulty, algorithm)
	}
	
	return "", fmt.Errorf("no compatible challenge format available")
}

// generateSecureChallenge creates a JSON-formatted secure challenge
func (cc *ChallengeCompatibility) generateSecureChallenge(clientID string, difficulty int, algorithm string) (string, error) {
	challenge, err := GenerateSecureChallenge(difficulty, algorithm, clientID, cc.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure challenge: %w", err)
	}
	
	jsonData, err := json.Marshal(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to serialize secure challenge: %w", err)
	}
	
	return string(jsonData), nil
}

// generateLegacyChallenge creates a text-formatted legacy challenge
func (cc *ChallengeCompatibility) generateLegacyChallenge(difficulty int, algorithm string) (string, error) {
	switch algorithm {
	case "sha256":
		challenge, err := GenerateChallenge(difficulty)
		if err != nil {
			return "", fmt.Errorf("failed to generate SHA-256 challenge: %w", err)
		}
		return challenge.String(), nil
		
	case "argon2":
		challenge, err := GenerateArgon2Challenge(difficulty)
		if err != nil {
			return "", fmt.Errorf("failed to generate Argon2 challenge: %w", err)
		}
		return challenge.String(), nil
		
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", algorithm)
	}
}

// ValidateCompatibleSolution validates a solution regardless of challenge format
func (cc *ChallengeCompatibility) ValidateCompatibleSolution(challengeStr, solution string) error {
	// Detect challenge format
	if strings.HasPrefix(challengeStr, "{") {
		// JSON format - secure challenge
		return cc.validateSecureSolution(challengeStr, solution)
	} else {
		// Legacy text format
		return cc.validateLegacySolution(challengeStr, solution)
	}
}

// validateSecureSolution validates a solution for a secure challenge
func (cc *ChallengeCompatibility) validateSecureSolution(challengeStr, solution string) error {
	// Parse JSON challenge
	var challenge SecureChallenge
	if err := json.Unmarshal([]byte(challengeStr), &challenge); err != nil {
		return fmt.Errorf("failed to parse JSON challenge: %w", err)
	}
	
	// Validate the solution
	return VerifySecurePoW(&challenge, solution, cc.signingKey)
}

// validateLegacySolution validates a solution for a legacy challenge
func (cc *ChallengeCompatibility) validateLegacySolution(challengeStr, solution string) error {
	if strings.Contains(challengeStr, "Argon2") {
		// Parse Argon2 challenge
		seed, difficulty, err := parseArgon2ChallengeLegacy(challengeStr)
		if err != nil {
			return fmt.Errorf("failed to parse Argon2 challenge: %w", err)
		}
		
		challenge, err := GenerateArgon2Challenge(difficulty)
		if err != nil {
			return fmt.Errorf("failed to create Argon2 challenge: %w", err)
		}
		challenge.Seed = seed
		
		if !VerifyArgon2PoW(challenge, solution) {
			return fmt.Errorf("invalid Argon2 solution")
		}
	} else {
		// Parse SHA-256 challenge
		seed, difficulty, err := parseChallengeLegacy(challengeStr)
		if err != nil {
			return fmt.Errorf("failed to parse SHA-256 challenge: %w", err)
		}
		
		if !VerifyPoW(seed, solution, difficulty) {
			return fmt.Errorf("invalid SHA-256 solution")
		}
	}
	
	return nil
}

// parseChallengeLegacy parses legacy SHA-256 challenge format
func parseChallengeLegacy(challenge string) (seed string, difficulty int, err error) {
	// Extract seed and difficulty from "Solve PoW: [seed] with prefix [zeros]"
	parts := strings.Fields(challenge)
	if len(parts) < 6 {
		return "", 0, fmt.Errorf("invalid challenge format")
	}
	
	seed = parts[2]
	prefix := parts[5]
	difficulty = len(prefix)
	
	return seed, difficulty, nil
}

// parseArgon2ChallengeLegacy parses legacy Argon2 challenge format
func parseArgon2ChallengeLegacy(challenge string) (seed string, difficulty int, err error) {
	// Extract seed and difficulty from "Solve Argon2 PoW: [seed] with [n] leading zeros"
	parts := strings.Fields(challenge)
	if len(parts) < 7 {
		return "", 0, fmt.Errorf("invalid Argon2 challenge format")
	}
	
	seed = parts[3]
	_, err = fmt.Sscanf(parts[5], "%d", &difficulty)
	if err != nil {
		return "", 0, fmt.Errorf("failed to parse difficulty: %w", err)
	}
	
	return seed, difficulty, nil
}

// SetSecureMode enables or disables secure challenge mode
func (cc *ChallengeCompatibility) SetSecureMode(enabled bool) {
	cc.enableSecureMode = enabled
}

// SetLegacyMode enables or disables legacy challenge mode
func (cc *ChallengeCompatibility) SetLegacyMode(enabled bool) {
	cc.enableLegacyMode = enabled
}

// GetSupportedFormats returns the supported challenge formats
func (cc *ChallengeCompatibility) GetSupportedFormats() []string {
	formats := []string{}
	
	if cc.enableSecureMode {
		formats = append(formats, "secure")
	}
	
	if cc.enableLegacyMode {
		formats = append(formats, "legacy")
	}
	
	return formats
}

// MigrateChallenge converts between challenge formats
func (cc *ChallengeCompatibility) MigrateChallenge(challengeStr string, targetFormat string) (string, error) {
	isSecure := strings.HasPrefix(challengeStr, "{")
	
	if targetFormat == "secure" && !isSecure {
		// Convert legacy to secure
		return cc.convertLegacyToSecure(challengeStr)
	}
	
	if targetFormat == "legacy" && isSecure {
		// Convert secure to legacy
		return cc.convertSecureToLegacy(challengeStr)
	}
	
	// Already in target format
	return challengeStr, nil
}

// convertLegacyToSecure converts a legacy challenge to secure format
func (cc *ChallengeCompatibility) convertLegacyToSecure(challengeStr string) (string, error) {
	var seed string
	var difficulty int
	var algorithm string
	var err error
	
	if strings.Contains(challengeStr, "Argon2") {
		seed, difficulty, err = parseArgon2ChallengeLegacy(challengeStr)
		algorithm = "argon2"
	} else {
		seed, difficulty, err = parseChallengeLegacy(challengeStr)
		algorithm = "sha256"
	}
	
	if err != nil {
		return "", fmt.Errorf("failed to parse legacy challenge: %w", err)
	}
	
	// Generate secure challenge with same parameters
	challenge, err := GenerateSecureChallenge(difficulty, algorithm, "converted", cc.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure challenge: %w", err)
	}
	
	// Use the original seed
	challenge.Seed = seed
	
	// Re-sign with the modified seed
	if err := challenge.Sign(cc.signingKey); err != nil {
		return "", fmt.Errorf("failed to sign converted challenge: %w", err)
	}
	
	jsonData, err := json.Marshal(challenge)
	if err != nil {
		return "", fmt.Errorf("failed to serialize converted challenge: %w", err)
	}
	
	return string(jsonData), nil
}

// convertSecureToLegacy converts a secure challenge to legacy format
func (cc *ChallengeCompatibility) convertSecureToLegacy(challengeStr string) (string, error) {
	var challenge SecureChallenge
	if err := json.Unmarshal([]byte(challengeStr), &challenge); err != nil {
		return "", fmt.Errorf("failed to parse secure challenge: %w", err)
	}
	
	switch challenge.Algorithm {
	case "sha256":
		basicChallenge := &Challenge{
			Seed:       challenge.Seed,
			Difficulty: challenge.Difficulty,
		}
		return basicChallenge.String(), nil
		
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
		return argon2Challenge.String(), nil
		
	default:
		return "", fmt.Errorf("unsupported algorithm: %s", challenge.Algorithm)
	}
}

// ClientCapabilities represents what formats a client supports
type ClientCapabilities struct {
	SupportsSecure bool `json:"supports_secure"`
	SupportsLegacy bool `json:"supports_legacy"`
	ClientID       string `json:"client_id"`
	Version        string `json:"version"`
}

// DetectClientCapabilities attempts to detect client capabilities
func (cc *ChallengeCompatibility) DetectClientCapabilities(clientID string) ClientCapabilities {
	// This would be implemented with proper client detection logic
	// For now, assume all clients support legacy format
	return ClientCapabilities{
		SupportsSecure: false,
		SupportsLegacy: true,
		ClientID:       clientID,
		Version:        "1.0",
	}
}

// RecommendFormat recommends the best format for a client
func (cc *ChallengeCompatibility) RecommendFormat(caps ClientCapabilities) string {
	if caps.SupportsSecure && cc.enableSecureMode {
		return "secure"
	}
	
	if caps.SupportsLegacy && cc.enableLegacyMode {
		return "legacy"
	}
	
	return ""
}