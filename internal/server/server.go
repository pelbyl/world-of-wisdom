package server

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"world-of-wisdom/internal/behavior"
	generated "world-of-wisdom/internal/database/generated"
	"world-of-wisdom/pkg/metrics"
	"world-of-wisdom/pkg/pow"
	"world-of-wisdom/pkg/wisdom"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Server struct {
	listener      net.Listener
	quoteProvider *wisdom.QuoteProvider
	difficulty    int
	timeout       time.Duration
	mu            sync.RWMutex
	activeConns   sync.WaitGroup
	shutdownChan  chan struct{}

	// Database components
	dbpool  *pgxpool.Pool
	queries *generated.Queries

	// Adaptive difficulty tracking
	solveTimes     []time.Duration
	connectionRate int64
	lastAdjustment time.Time
	adaptiveMode   bool

	// PoW algorithm selection
	algorithm string // "sha256" or "argon2"

	// Client behavior tracking
	behaviorTracker *behavior.Tracker
	
	// HMAC key management for secure challenges
	keyManager *pow.KeyManager
	
	// Challenge protocol format
	challengeFormat pow.ChallengeFormat // "json" or "binary"
	challengeEncoder *pow.ChallengeEncoder
}

type Config struct {
	Port            string
	Difficulty      int
	Timeout         time.Duration
	AdaptiveMode    bool
	MetricsPort     string
	Algorithm       string // "sha256" or "argon2"
	DatabaseURL     string
	ChallengeFormat string // "json" or "binary"
}

func NewServer(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.Port, err)
	}

	// Connect to database
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test database connection
	if err := dbpool.Ping(ctx); err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("✅ TCP Server connected to database")

	// Start metrics server if port specified
	if cfg.MetricsPort != "" {
		metrics.StartMetricsServer(cfg.MetricsPort)
		log.Printf("Metrics server started on %s", cfg.MetricsPort)
	}

	// Initialize metrics
	metrics.UpdateCurrentDifficulty(cfg.Difficulty)

	// Default to argon2 if not specified
	algorithm := cfg.Algorithm
	if algorithm == "" {
		algorithm = "argon2"
	}
	if algorithm != "sha256" && algorithm != "argon2" {
		return nil, fmt.Errorf("invalid algorithm: %s (must be sha256 or argon2)", algorithm)
	}

	// Initialize key manager for HMAC signing
	keyPath := os.Getenv("WOW_KEY_PATH")
	if keyPath == "" {
		keyPath = "wow-hmac-keys.json"
	}
	keyManager, err := pow.NewKeyManager(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize key manager: %w", err)
	}
	log.Printf("✅ HMAC key manager initialized with persistent storage")

	// Default to binary format if not specified
	challengeFormat := pow.ChallengeFormat(cfg.ChallengeFormat)
	if challengeFormat == "" {
		challengeFormat = pow.FormatBinary
	}
	if challengeFormat != pow.FormatJSON && challengeFormat != pow.FormatBinary {
		return nil, fmt.Errorf("invalid challenge format: %s (must be json or binary)", challengeFormat)
	}

	return &Server{
		listener:         listener,
		quoteProvider:    wisdom.NewQuoteProvider(),
		difficulty:       cfg.Difficulty,
		timeout:          cfg.Timeout,
		shutdownChan:     make(chan struct{}),
		dbpool:           dbpool,
		queries:          generated.New(),
		solveTimes:       make([]time.Duration, 0, 100),
		lastAdjustment:   time.Now(),
		adaptiveMode:     cfg.AdaptiveMode,
		algorithm:        algorithm,
		behaviorTracker:  behavior.NewTracker(dbpool),
		keyManager:       keyManager,
		challengeFormat:  challengeFormat,
		challengeEncoder: pow.NewChallengeEncoder(challengeFormat),
	}, nil
}

func (s *Server) Start() error {
	log.Printf("Server listening on %s with difficulty %d (format: %s)", s.listener.Addr(), s.difficulty, s.challengeFormat)

	// Start periodic behavior stats logging
	go s.logBehaviorStats()

	for {
		select {
		case <-s.shutdownChan:
			return nil
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.shutdownChan:
					return nil
				default:
					log.Printf("Failed to accept connection: %v", err)
					continue
				}
			}

			s.activeConns.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.activeConns.Done()
	defer conn.Close()

	startTime := time.Now()
	clientAddr := conn.RemoteAddr().String()
	clientID := s.generateClientID(clientAddr)
	log.Printf("New connection from %s (Client ID: %s)", clientAddr, clientID)
	
	// Context for database operations
	ctx := context.Background()
	
	// Track connection record for cleanup
	var connectionRecord generated.Connection
	// Track client behavior for this connection
	var clientBehavior *behavior.ClientBehavior
	defer func() {
		// Record disconnection in behavior tracker
		if clientBehavior != nil && clientBehavior.ConnectionTimestampID != (pgtype.UUID{}) {
			err := s.behaviorTracker.RecordDisconnection(ctx, clientBehavior.ConnectionTimestampID, 
				connectionRecord.ID != (pgtype.UUID{}))
			if err != nil {
				log.Printf("Failed to record disconnection: %v", err)
			}
		}
		
		// Always mark connection as disconnected when handler exits
		if connectionRecord.ID != (pgtype.UUID{}) {
			s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusDisconnected)
			s.logActivity(ctx, "info", fmt.Sprintf("Client disconnected: %s", clientAddr), map[string]interface{}{
				"client_id": clientID,
				"event":     "connection_closed",
			})
			log.Printf("Client %s disconnected (cleanup)", clientAddr)
		}
	}()

	// Log new connection
	s.logActivity(context.Background(), "info", fmt.Sprintf("New connection from %s", clientAddr), map[string]interface{}{
		"client_id":   clientID,
		"remote_addr": clientAddr,
		"event":       "connection_established",
	})

	// Parse remote address
	remoteAddr, err := netip.ParseAddr(strings.Split(clientAddr, ":")[0])
	if err != nil {
		log.Printf("Failed to parse remote address %s: %v", clientAddr, err)
		return
	}

	// Get previous behavior if exists
	prevBehavior, _ := s.behaviorTracker.GetClientBehavior(ctx, remoteAddr)
	prevDifficulty := prevBehavior.Difficulty
	prevConnectionCount := prevBehavior.ConnectionCount
	
	// Track client behavior and get per-client difficulty
	clientBehavior, err = s.behaviorTracker.RecordConnection(ctx, remoteAddr)
	if err != nil {
		log.Printf("Failed to track client behavior: %v", err)
		// Fall back to global difficulty
		clientBehavior = &behavior.ClientBehavior{
			IP:         remoteAddr,
			Difficulty: s.getDifficulty(),
		}
	}
	
	// Log connection with behavior context
	if prevConnectionCount > 0 {
		s.logActivity(ctx, "info", fmt.Sprintf("Client %s reconnected (connection #%d)", remoteAddr.String(), clientBehavior.ConnectionCount), map[string]interface{}{
			"ip":                remoteAddr.String(),
			"connection_count":  clientBehavior.ConnectionCount,
			"failure_rate":      fmt.Sprintf("%.2f%%", clientBehavior.FailureRate*100),
			"avg_solve_time_ms": clientBehavior.AvgSolveTime.Milliseconds(),
			"reconnect_rate":    fmt.Sprintf("%.2f%%", clientBehavior.ReconnectRate*100),
			"reputation_score":  clientBehavior.ReputationScore,
			"event":             "client_reconnected",
		})
		
		// Log difficulty change on reconnection
		if prevDifficulty != clientBehavior.Difficulty {
			s.logActivity(ctx, "warning", fmt.Sprintf("Client %s difficulty changed from %d to %d on reconnection", remoteAddr.String(), prevDifficulty, clientBehavior.Difficulty), map[string]interface{}{
				"ip":             remoteAddr.String(),
				"old_difficulty": prevDifficulty,
				"new_difficulty": clientBehavior.Difficulty,
				"reason":         "reconnection_pattern",
				"event":          "difficulty_adjusted",
			})
		}
	} else {
		// First connection
		s.logActivity(ctx, "info", fmt.Sprintf("New client %s connected with initial difficulty %d", remoteAddr.String(), clientBehavior.Difficulty), map[string]interface{}{
			"ip":                 remoteAddr.String(),
			"initial_difficulty": clientBehavior.Difficulty,
			"reputation_score":   clientBehavior.ReputationScore,
			"event":              "new_client_connected",
		})
	}

	// Create connection record in database
	connectionRecord, err = s.logConnection(ctx, clientID, remoteAddr, s.algorithm)
	if err != nil {
		log.Printf("Failed to log connection: %v", err)
		// Continue anyway - don't fail the connection due to DB issues
	}

	// Record connection metrics
	metrics.RecordConnection("accepted")
	conn.SetDeadline(time.Now().Add(s.timeout))

	// Track connection rate for adaptive difficulty
	s.trackConnection()

	// Use per-client difficulty
	difficulty := clientBehavior.Difficulty
	log.Printf("Client %s assigned difficulty %d (reputation: %.1f, suspicious: %.1f)", 
		clientAddr, difficulty, clientBehavior.ReputationScore, clientBehavior.SuspiciousScore)
	
	// Log if client is flagged as aggressive
	if difficulty >= 5 {
		s.logActivity(ctx, "warning", fmt.Sprintf("High difficulty assigned to potential DDoS client: %s", remoteAddr.String()), map[string]interface{}{
			"ip":                remoteAddr.String(),
			"difficulty":        difficulty,
			"reputation_score":  clientBehavior.ReputationScore,
			"suspicious_score":  clientBehavior.SuspiciousScore,
			"event":             "high_difficulty_assigned",
		})
	}

	// Generate secure JSON challenge
	var secureChallenge *pow.SecureChallenge
	var challengeSeed string
	var verifySolution func(string) bool

	// Use secure challenge generation with key manager
	secureChallenge, err = pow.GenerateSecureChallengeWithKeyManager(difficulty, s.algorithm, clientID, s.keyManager)
	if err != nil {
		log.Printf("Failed to generate secure challenge: %v", err)
		conn.Write([]byte("Error: Failed to generate challenge\n"))
		s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
		return
	}
	
	challengeSeed = secureChallenge.Seed
	
	// Set up verification function based on algorithm
	if s.algorithm == "sha256" {
		verifySolution = func(response string) bool {
			return pow.VerifyPoW(secureChallenge.Seed, response, secureChallenge.Difficulty)
		}
	} else {
		// For Argon2, we need to verify with the secure challenge
		verifySolution = func(response string) bool {
			// Create an Argon2Challenge from the SecureChallenge for verification
			argon2Challenge := &pow.Argon2Challenge{
				Seed:       secureChallenge.Seed,
				Difficulty: secureChallenge.Difficulty,
			}
			if secureChallenge.Argon2Params != nil {
				argon2Challenge.Time = secureChallenge.Argon2Params.Time
				argon2Challenge.Memory = secureChallenge.Argon2Params.Memory
				argon2Challenge.Threads = secureChallenge.Argon2Params.Threads
				argon2Challenge.KeyLen = secureChallenge.Argon2Params.KeyLength
			}
			return pow.VerifyArgon2PoW(argon2Challenge, response)
		}
	}
	
	// Encode challenge using configured format
	challengeData, err := s.challengeEncoder.Encode(secureChallenge, s.challengeFormat)
	if err != nil {
		log.Printf("Failed to encode challenge: %v", err)
		conn.Write([]byte("Error: Failed to generate challenge\n"))
		s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
		return
	}
	
	log.Printf("Sending %s challenge to %s (size: %d bytes)", s.challengeFormat, clientAddr, len(challengeData))

	// Log challenge to database
	challengeRecord, err := s.logChallenge(ctx, challengeSeed, int32(difficulty), s.algorithm, clientID)
	if err != nil {
		log.Printf("Failed to log challenge: %v", err)
		// Continue anyway
	}

	// Update connection status to solving
	s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusSolving)

	_, err = conn.Write(append(challengeData, '\n'))
	if err != nil {
		log.Printf("Failed to send challenge to %s: %v", clientAddr, err)
		s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
		return
	}

	solveStart := time.Now()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		log.Printf("Client %s disconnected or timed out", clientAddr)
		
		// Log disconnection
		s.logActivity(ctx, "warning", fmt.Sprintf("Client disconnected: %s", clientAddr), map[string]interface{}{
			"client_id": clientID,
			"event":     "client_disconnected",
			"reason":    "timeout_or_disconnect",
		})
		
		// Record expired challenge as failed attempt for behavior tracking
		solveTime := time.Since(solveStart)
		err = s.behaviorTracker.RecordChallengeResult(ctx, remoteAddr, false, solveTime)
		if err != nil {
			log.Printf("Failed to record expired challenge result: %v", err)
		}
		
		s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusDisconnected)
		if challengeRecord.ID != (pgtype.UUID{}) {
			s.updateChallengeStatus(ctx, challengeRecord.ID, generated.ChallengeStatusExpired)
		}
		return
	}

	response := strings.TrimSpace(scanner.Text())
	solveTime := time.Since(solveStart)

	if verifySolution(response) {
		log.Printf("Client %s solved the %s challenge in %v", clientAddr, s.algorithm, solveTime)
		s.recordSolveTime(solveTime)

		// Get current reputation before update
		oldBehavior, _ := s.behaviorTracker.GetClientBehavior(ctx, remoteAddr)
		oldReputation := oldBehavior.ReputationScore
		
		// Update client behavior with successful challenge
		err = s.behaviorTracker.RecordChallengeResult(ctx, remoteAddr, true, solveTime)
		if err != nil {
			log.Printf("Failed to record challenge result: %v", err)
		}
		
		// Get new behavior to check changes
		newBehavior, _ := s.behaviorTracker.GetClientBehavior(ctx, remoteAddr)
		newReputation := newBehavior.ReputationScore
		newDifficulty := newBehavior.Difficulty
		
		// Log reputation change
		if oldReputation != newReputation {
			s.logActivity(ctx, "info", fmt.Sprintf("Client %s reputation increased from %.1f to %.1f after successful challenge", remoteAddr.String(), oldReputation, newReputation), map[string]interface{}{
				"ip":                remoteAddr.String(),
				"old_reputation":    oldReputation,
				"new_reputation":    newReputation,
				"change":            newReputation - oldReputation,
				"event":             "reputation_increased",
			})
		}
		
		// Log difficulty change if it occurred
		if difficulty != newDifficulty {
			s.logActivity(ctx, "info", fmt.Sprintf("Client %s difficulty changed from %d to %d after successful challenge", remoteAddr.String(), difficulty, newDifficulty), map[string]interface{}{
				"ip":                remoteAddr.String(),
				"old_difficulty":    difficulty,
				"new_difficulty":    newDifficulty,
				"event":             "difficulty_changed",
			})
		}

		// Log successful solution
		s.logActivity(ctx, "success", fmt.Sprintf("Challenge solved by %s", clientAddr), map[string]interface{}{
			"client_id":   clientID,
			"solve_time":  solveTime.Milliseconds(),
			"difficulty":  difficulty,
			"algorithm":   s.algorithm,
			"event":       "challenge_solved",
		})

		// Log successful solution to database
		if challengeRecord.ID != (pgtype.UUID{}) {
			s.logSolution(ctx, challengeRecord.ID, response, true, solveTime)
			s.updateChallengeStatus(ctx, challengeRecord.ID, generated.ChallengeStatusCompleted)
		}

		// Record metrics
		metrics.RecordPuzzleSolved(difficulty, solveTime)
		metrics.RecordProcessingTime("success", time.Since(startTime))

		quote := s.quoteProvider.GetRandomQuote()
		conn.Write([]byte(quote + "\n"))
	} else {
		log.Printf("Client %s failed the %s challenge", clientAddr, s.algorithm)

		// Get current reputation before update
		oldBehavior, _ := s.behaviorTracker.GetClientBehavior(ctx, remoteAddr)
		oldReputation := oldBehavior.ReputationScore
		
		// Update client behavior with failed challenge
		err = s.behaviorTracker.RecordChallengeResult(ctx, remoteAddr, false, solveTime)
		if err != nil {
			log.Printf("Failed to record challenge result: %v", err)
		}
		
		// Get new behavior to check changes
		newBehavior, _ := s.behaviorTracker.GetClientBehavior(ctx, remoteAddr)
		newReputation := newBehavior.ReputationScore
		newDifficulty := newBehavior.Difficulty
		
		// Log reputation decrease
		if oldReputation != newReputation {
			s.logActivity(ctx, "warning", fmt.Sprintf("Client %s reputation decreased from %.1f to %.1f after failed challenge", remoteAddr.String(), oldReputation, newReputation), map[string]interface{}{
				"ip":                remoteAddr.String(),
				"old_reputation":    oldReputation,
				"new_reputation":    newReputation,
				"change":            newReputation - oldReputation,
				"event":             "reputation_decreased",
			})
		}
		
		// Log difficulty change if it occurred
		if difficulty != newDifficulty {
			s.logActivity(ctx, "warning", fmt.Sprintf("Client %s difficulty increased from %d to %d after failed challenge", remoteAddr.String(), difficulty, newDifficulty), map[string]interface{}{
				"ip":                remoteAddr.String(),
				"old_difficulty":    difficulty,
				"new_difficulty":    newDifficulty,
				"event":             "difficulty_increased",
			})
		}

		// Log failed challenge
		s.logActivity(ctx, "warning", fmt.Sprintf("Challenge failed by %s", clientAddr), map[string]interface{}{
			"client_id":   clientID,
			"solve_time":  solveTime.Milliseconds(),
			"difficulty":  difficulty,
			"algorithm":   s.algorithm,
			"event":       "challenge_failed",
		})

		// Log failed solution to database
		if challengeRecord.ID != (pgtype.UUID{}) {
			s.logSolution(ctx, challengeRecord.ID, response, false, solveTime)
			s.updateChallengeStatus(ctx, challengeRecord.ID, generated.ChallengeStatusFailed)
		}

		// Record metrics
		metrics.RecordPuzzleFailed(difficulty)
		metrics.RecordProcessingTime("failed", time.Since(startTime))

		conn.Write([]byte("Error: Invalid proof of work\n"))
	}
	
	// Connection status will be updated by defer
}

func (s *Server) getDifficulty() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.difficulty
}

func (s *Server) SetDifficulty(difficulty int) error {
	if difficulty < 1 || difficulty > 6 {
		return fmt.Errorf("difficulty must be between 1 and 6")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.difficulty = difficulty
	log.Printf("Difficulty updated to %d", difficulty)
	return nil
}

func (s *Server) Shutdown() error {
	log.Println("Shutting down server...")
	close(s.shutdownChan)

	err := s.listener.Close()
	if err != nil {
		return fmt.Errorf("failed to close listener: %w", err)
	}

	done := make(chan struct{})
	go func() {
		s.activeConns.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All connections closed")
	case <-time.After(10 * time.Second):
		log.Println("Timeout waiting for connections to close")
	}

	// Close database connection pool
	if s.dbpool != nil {
		s.dbpool.Close()
		log.Println("✅ Database connection pool closed")
	}

	return nil
}

func (s *Server) trackConnection() {
	if !s.adaptiveMode {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.connectionRate++
}

func (s *Server) recordSolveTime(solveTime time.Duration) {
	if !s.adaptiveMode {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.solveTimes = append(s.solveTimes, solveTime)

	// Keep only last 50 solve times
	if len(s.solveTimes) > 50 {
		s.solveTimes = s.solveTimes[len(s.solveTimes)-50:]
	}

	// Adjust difficulty every 10 solutions or every 30 seconds
	if len(s.solveTimes) >= 10 || time.Since(s.lastAdjustment) > 30*time.Second {
		s.adjustDifficulty()
	}
}

func (s *Server) adjustDifficulty() {
	if len(s.solveTimes) == 0 {
		return
	}

	// Calculate average solve time
	var total time.Duration
	for _, t := range s.solveTimes {
		total += t
	}
	avgSolveTime := total / time.Duration(len(s.solveTimes))

	oldDifficulty := s.difficulty

	// Adaptive difficulty rules:
	// - If avg solve time < 1s: increase difficulty
	// - If avg solve time > 5s: decrease difficulty
	// - If connection rate is high (>20/min): increase difficulty

	connectionRatePerMinute := float64(s.connectionRate) / time.Since(s.lastAdjustment).Minutes()

	if avgSolveTime < time.Second || connectionRatePerMinute > 20 {
		if s.difficulty < 6 {
			s.difficulty++
		}
	} else if avgSolveTime > 5*time.Second && connectionRatePerMinute < 5 {
		if s.difficulty > 1 {
			s.difficulty--
		}
	}

	if s.difficulty != oldDifficulty {
		direction := "increase"
		if s.difficulty < oldDifficulty {
			direction = "decrease"
		}

		log.Printf("Adaptive difficulty: %d -> %d (avg solve: %v, rate: %.1f/min)",
			oldDifficulty, s.difficulty, avgSolveTime, connectionRatePerMinute)

		// Record metrics
		metrics.RecordDifficultyAdjustment(direction)
		metrics.UpdateCurrentDifficulty(s.difficulty)
	}

	// Reset tracking
	s.solveTimes = s.solveTimes[:0]
	s.connectionRate = 0
	s.lastAdjustment = time.Now()
}

func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var avgSolveTime time.Duration
	if len(s.solveTimes) > 0 {
		var total time.Duration
		for _, t := range s.solveTimes {
			total += t
		}
		avgSolveTime = total / time.Duration(len(s.solveTimes))
	}

	connectionRatePerMinute := float64(s.connectionRate) / time.Since(s.lastAdjustment).Minutes()

	return map[string]interface{}{
		"difficulty":         s.difficulty,
		"adaptive_mode":      s.adaptiveMode,
		"avg_solve_time_ms":  avgSolveTime.Milliseconds(),
		"connection_rate":    connectionRatePerMinute,
		"recent_solve_count": len(s.solveTimes),
		"last_adjustment":    s.lastAdjustment.Unix(),
	}
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *Server) logBehaviorStats() {
	ticker := time.NewTicker(60 * time.Second) // Log every 60 seconds
	defer ticker.Stop()

	for {
		select {
		case <-s.shutdownChan:
			return
		case <-ticker.C:
			ctx := context.Background()
			
			// Get aggressive clients
			aggressiveClients, err := s.behaviorTracker.GetAggressiveClients(ctx, 10)
			if err != nil {
				log.Printf("Failed to get aggressive clients: %v", err)
				continue
			}
			
			// Log details of aggressive clients only if they exist
			if len(aggressiveClients) > 0 {
				for _, client := range aggressiveClients {
					s.logActivity(ctx, "warning", fmt.Sprintf("Aggressive client detected: %s (difficulty: %d, reputation: %.1f)", 
						client.IpAddress.String(), client.Difficulty.Int32, client.ReputationScore.Float64), map[string]interface{}{
						"ip":                client.IpAddress.String(),
						"difficulty":        client.Difficulty.Int32,
						"failure_rate":      fmt.Sprintf("%.2f%%", client.FailureRate.Float64*100),
						"reconnect_rate":    fmt.Sprintf("%.2f%%", client.ReconnectRate.Float64*100),
						"reputation_score":  client.ReputationScore.Float64,
						"suspicious_score":  client.SuspiciousActivityScore.Float64,
						"connection_count":  client.ConnectionCount.Int32,
						"avg_solve_time_ms": client.AvgSolveTimeMs.Int64,
						"event":             "aggressive_client_alert",
					})
				}
			}
		}
	}
}

// Database helper functions for write-only operations

func (s *Server) generateClientID(clientAddr string) string {
	return uuid.New().String()
}

func (s *Server) logActivity(ctx context.Context, level, message string, metadata map[string]interface{}) {
	// Convert metadata to JSONB
	var metadataJSON []byte
	if metadata != nil {
		metadataJSON, _ = json.Marshal(metadata)
	}
	
	params := generated.CreateLogParams{
		Column1:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		Level:    level,
		Message:  message,
		Metadata: metadataJSON,
	}
	
	_, err := s.queries.CreateLog(ctx, s.dbpool, params)
	if err != nil {
		log.Printf("Failed to create log entry: %v", err)
	}
}

func (s *Server) logConnection(ctx context.Context, clientID string, remoteAddr netip.Addr, algorithm string) (generated.Connection, error) {
	var algo generated.PowAlgorithm
	switch algorithm {
	case "sha256":
		algo = generated.PowAlgorithmSha256
	case "argon2":
		algo = generated.PowAlgorithmArgon2
	default:
		algo = generated.PowAlgorithmArgon2
	}

	params := generated.CreateConnectionParams{
		ClientID:   clientID,
		RemoteAddr: remoteAddr,
		Status:     generated.ConnectionStatusConnected,
		Algorithm:  algo,
	}

	return s.queries.CreateConnection(ctx, s.dbpool, params)
}

func (s *Server) updateConnectionStatus(ctx context.Context, connectionID pgtype.UUID, status generated.ConnectionStatus) {
	if connectionID == (pgtype.UUID{}) {
		return // Skip if no valid connection ID
	}

	params := generated.UpdateConnectionStatusParams{
		ID:     connectionID,
		Status: status,
	}

	_, err := s.queries.UpdateConnectionStatus(ctx, s.dbpool, params)
	if err != nil {
		log.Printf("Failed to update connection status: %v", err)
	}
}

func (s *Server) logChallenge(ctx context.Context, seed string, difficulty int32, algorithm, clientID string) (generated.Challenge, error) {
	var algo generated.PowAlgorithm
	switch algorithm {
	case "sha256":
		algo = generated.PowAlgorithmSha256
	case "argon2":
		algo = generated.PowAlgorithmArgon2
	default:
		algo = generated.PowAlgorithmArgon2
	}

	params := generated.CreateChallengeParams{
		Seed:       seed,
		Difficulty: difficulty,
		Algorithm:  algo,
		ClientID:   clientID,
		Status:     generated.ChallengeStatusPending,
		// Argon2 parameters (only used for argon2 challenges)
		Argon2Time:    pgtype.Int4{Int32: 1, Valid: algorithm == "argon2"},
		Argon2Memory:  pgtype.Int4{Int32: 64 * 1024, Valid: algorithm == "argon2"},
		Argon2Threads: pgtype.Int2{Int16: 4, Valid: algorithm == "argon2"},
		Argon2Keylen:  pgtype.Int4{Int32: 32, Valid: algorithm == "argon2"},
	}

	return s.queries.CreateChallenge(ctx, s.dbpool, params)
}

func (s *Server) updateChallengeStatus(ctx context.Context, challengeID pgtype.UUID, status generated.ChallengeStatus) {
	if challengeID == (pgtype.UUID{}) {
		return // Skip if no valid challenge ID
	}

	params := generated.UpdateChallengeStatusParams{
		ID:     challengeID,
		Status: status,
	}

	_, err := s.queries.UpdateChallengeStatus(ctx, s.dbpool, params)
	if err != nil {
		log.Printf("Failed to update challenge status: %v", err)
	}
}

func (s *Server) logSolution(ctx context.Context, challengeID pgtype.UUID, solution string, valid bool, solveTime time.Duration) {
	if challengeID == (pgtype.UUID{}) {
		return // Skip if no valid challenge ID
	}

	params := generated.CreateSolutionParams{
		ChallengeID: challengeID,
		Nonce:       solution,
		Hash:        pgtype.Text{String: "", Valid: false}, // Can be empty for now
		Attempts:    pgtype.Int4{Int32: 1, Valid: true},
		SolveTimeMs: solveTime.Milliseconds(),
		Verified:    valid,
	}

	_, err := s.queries.CreateSolution(ctx, s.dbpool, params)
	if err != nil {
		log.Printf("Failed to log solution: %v", err)
	}
}
