package pow

import (
	"fmt"
	"time"
)

// ValidationPipeline provides fast, multi-stage validation of proof-of-work solutions
type ValidationPipeline struct {
	signingKey []byte
	
	// Caching for performance (using simple maps for now)
	hmacCache      map[string]bool
	challengeCache map[string]*SecureChallenge
	
	// Rate limiting state
	rateLimitMap map[string]*RateLimitState
	
	// Configuration
	maxCacheSize    int
	rateLimitWindow time.Duration
	maxRequestsPerWindow int
}

// RateLimitState tracks rate limiting per client
type RateLimitState struct {
	requests   int
	windowStart time.Time
}

// Solution represents a proof-of-work solution submission
type Solution struct {
	ChallengeID string          `json:"challenge_id"`
	Challenge   *SecureChallenge `json:"challenge"`
	Nonce       string          `json:"nonce"`
	ClientID    string          `json:"client_id"`
	Timestamp   int64           `json:"timestamp"`
	SolveTime   time.Duration   `json:"solve_time"`
}

// ValidationResult contains the result of validation
type ValidationResult struct {
	Valid        bool          `json:"valid"`
	Error        error         `json:"error,omitempty"`
	Stage        string        `json:"stage"`
	Duration     time.Duration `json:"duration"`
	ClientID     string        `json:"client_id"`
}

// ValidationError represents different types of validation errors
type ValidationError struct {
	Stage   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed at stage %s: %s", e.Stage, e.Message)
}

// NewValidationPipeline creates a new validation pipeline
func NewValidationPipeline(signingKey []byte) *ValidationPipeline {
	return &ValidationPipeline{
		signingKey:           signingKey,
		hmacCache:            make(map[string]bool),
		challengeCache:       make(map[string]*SecureChallenge),
		rateLimitMap:         make(map[string]*RateLimitState),
		maxCacheSize:         1000,
		rateLimitWindow:      time.Minute,
		maxRequestsPerWindow: 60, // 1 request per second average
	}
}

// Validate performs fast multi-stage validation of a solution
func (v *ValidationPipeline) Validate(solution *Solution) *ValidationResult {
	start := time.Now()
	
	// Step 0: Rate limiting check (fail-fastest)
	if err := v.checkRateLimit(solution.ClientID); err != nil {
		return &ValidationResult{
			Valid:    false,
			Error:    &ValidationError{Stage: "rate_limit", Message: err.Error()},
			Stage:    "rate_limit",
			Duration: time.Since(start),
			ClientID: solution.ClientID,
		}
	}
	
	// Step 1: Format validation (fail-fast)
	if err := v.validateFormat(solution); err != nil {
		return &ValidationResult{
			Valid:    false,
			Error:    &ValidationError{Stage: "format", Message: err.Error()},
			Stage:    "format",
			Duration: time.Since(start),
			ClientID: solution.ClientID,
		}
	}
	
	// Step 2: Timestamp check (prevent old/future challenges)
	if err := v.validateTimestamp(solution); err != nil {
		return &ValidationResult{
			Valid:    false,
			Error:    &ValidationError{Stage: "timestamp", Message: err.Error()},
			Stage:    "timestamp",
			Duration: time.Since(start),
			ClientID: solution.ClientID,
		}
	}
	
	// Step 3: Signature verification (with caching)
	if err := v.verifySignature(solution); err != nil {
		return &ValidationResult{
			Valid:    false,
			Error:    &ValidationError{Stage: "signature", Message: err.Error()},
			Stage:    "signature",
			Duration: time.Since(start),
			ClientID: solution.ClientID,
		}
	}
	
	// Step 4: PoW verification (most expensive)
	if err := v.verifyPoW(solution); err != nil {
		return &ValidationResult{
			Valid:    false,
			Error:    &ValidationError{Stage: "pow", Message: err.Error()},
			Stage:    "pow",
			Duration: time.Since(start),
			ClientID: solution.ClientID,
		}
	}
	
	return &ValidationResult{
		Valid:    true,
		Stage:    "complete",
		Duration: time.Since(start),
		ClientID: solution.ClientID,
	}
}

// checkRateLimit implements per-client rate limiting
func (v *ValidationPipeline) checkRateLimit(clientID string) error {
	now := time.Now()
	
	state, exists := v.rateLimitMap[clientID]
	if !exists {
		v.rateLimitMap[clientID] = &RateLimitState{
			requests:    1,
			windowStart: now,
		}
		return nil
	}
	
	// Check if we need to reset the window
	if now.Sub(state.windowStart) > v.rateLimitWindow {
		state.requests = 1
		state.windowStart = now
		return nil
	}
	
	// Check if we've exceeded the limit
	if state.requests >= v.maxRequestsPerWindow {
		return fmt.Errorf("rate limit exceeded: %d requests in %v", 
			state.requests, v.rateLimitWindow)
	}
	
	state.requests++
	return nil
}

// validateFormat checks the basic format and structure of the solution
func (v *ValidationPipeline) validateFormat(solution *Solution) error {
	if solution == nil {
		return fmt.Errorf("solution is nil")
	}
	
	if solution.Challenge == nil {
		return fmt.Errorf("challenge is nil")
	}
	
	if solution.Nonce == "" {
		return fmt.Errorf("nonce is empty")
	}
	
	if solution.ClientID == "" {
		return fmt.Errorf("client ID is empty")
	}
	
	if solution.ChallengeID == "" {
		return fmt.Errorf("challenge ID is empty")
	}
	
	// Basic challenge format validation
	if solution.Challenge.Version != 1 {
		return fmt.Errorf("unsupported challenge version: %d", solution.Challenge.Version)
	}
	
	if solution.Challenge.Algorithm != "sha256" && solution.Challenge.Algorithm != "argon2" {
		return fmt.Errorf("unsupported algorithm: %s", solution.Challenge.Algorithm)
	}
	
	if solution.Challenge.Difficulty < 1 || solution.Challenge.Difficulty > 6 {
		return fmt.Errorf("invalid difficulty: %d", solution.Challenge.Difficulty)
	}
	
	return nil
}

// validateTimestamp checks if the challenge is within valid time bounds
func (v *ValidationPipeline) validateTimestamp(solution *Solution) error {
	now := time.Now().UnixMicro()
	
	// Check if challenge has expired
	if solution.Challenge.ExpiresAt < now {
		return fmt.Errorf("challenge has expired")
	}
	
	// Check if challenge is from the future (allow 1 minute clock skew)
	maxFuture := now + (1 * time.Minute).Microseconds()
	if solution.Challenge.Timestamp > maxFuture {
		return fmt.Errorf("challenge timestamp is too far in the future")
	}
	
	// Check if challenge is too old (beyond reasonable solve time)
	minAge := now - (10 * time.Minute).Microseconds()
	if solution.Challenge.Timestamp < minAge {
		return fmt.Errorf("challenge timestamp is too old")
	}
	
	return nil
}

// verifySignature verifies the challenge signature with caching
func (v *ValidationPipeline) verifySignature(solution *Solution) error {
	challengeID := solution.ChallengeID
	
	// Check cache first
	if cached, exists := v.hmacCache[challengeID]; exists {
		if !cached {
			return fmt.Errorf("signature verification failed (cached)")
		}
		return nil
	}
	
	// Verify signature using constant-time comparison
	err := solution.Challenge.Verify(v.signingKey)
	
	// Cache the result (with simple cache size management)
	if len(v.hmacCache) >= v.maxCacheSize {
		// Simple cache cleanup - remove 10% of entries
		count := 0
		for k := range v.hmacCache {
			delete(v.hmacCache, k)
			count++
			if count >= v.maxCacheSize/10 {
				break
			}
		}
	}
	
	v.hmacCache[challengeID] = (err == nil)
	
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	
	return nil
}

// verifyPoW verifies the proof-of-work solution
func (v *ValidationPipeline) verifyPoW(solution *Solution) error {
	return VerifySecurePoW(solution.Challenge, solution.Nonce, v.signingKey)
}

// ClearCache clears the validation caches
func (v *ValidationPipeline) ClearCache() {
	v.hmacCache = make(map[string]bool)
	v.challengeCache = make(map[string]*SecureChallenge)
}

// GetCacheStats returns statistics about the validation cache
func (v *ValidationPipeline) GetCacheStats() map[string]int {
	return map[string]int{
		"hmac_cache_size":      len(v.hmacCache),
		"challenge_cache_size": len(v.challengeCache),
		"rate_limit_entries":   len(v.rateLimitMap),
	}
}

// SetRateLimitConfig updates the rate limiting configuration
func (v *ValidationPipeline) SetRateLimitConfig(window time.Duration, maxRequests int) {
	v.rateLimitWindow = window
	v.maxRequestsPerWindow = maxRequests
}


// BatchValidate validates multiple solutions concurrently
func (v *ValidationPipeline) BatchValidate(solutions []*Solution) []*ValidationResult {
	results := make([]*ValidationResult, len(solutions))
	
	// Simple parallel validation (could be optimized with worker pools)
	resultChan := make(chan struct {
		index  int
		result *ValidationResult
	}, len(solutions))
	
	for i, solution := range solutions {
		go func(idx int, sol *Solution) {
			result := v.Validate(sol)
			resultChan <- struct {
				index  int
				result *ValidationResult
			}{index: idx, result: result}
		}(i, solution)
	}
	
	// Collect results
	for range len(solutions) {
		res := <-resultChan
		results[res.index] = res.result
	}
	
	return results
}

// ValidationMetrics provides metrics about validation performance
type ValidationMetrics struct {
	TotalValidations     int64         `json:"total_validations"`
	SuccessfulValidations int64         `json:"successful_validations"`
	FailedValidations    int64         `json:"failed_validations"`
	AverageValidationTime time.Duration `json:"average_validation_time"`
	CacheHitRate         float64       `json:"cache_hit_rate"`
	RateLimitHits        int64         `json:"rate_limit_hits"`
}

// GetMetrics returns validation metrics (basic implementation)
func (v *ValidationPipeline) GetMetrics() *ValidationMetrics {
	// This would be implemented with proper metrics collection
	return &ValidationMetrics{
		TotalValidations:     0,
		SuccessfulValidations: 0,
		FailedValidations:    0,
		AverageValidationTime: 0,
		CacheHitRate:         0.0,
		RateLimitHits:        0,
	}
}