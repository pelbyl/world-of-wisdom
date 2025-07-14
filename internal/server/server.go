package server

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

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
}

type Config struct {
	Port         string
	Difficulty   int
	Timeout      time.Duration
	AdaptiveMode bool
	MetricsPort  string
	Algorithm    string // "sha256" or "argon2"
	DatabaseURL  string
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

	return &Server{
		listener:       listener,
		quoteProvider:  wisdom.NewQuoteProvider(),
		difficulty:     cfg.Difficulty,
		timeout:        cfg.Timeout,
		shutdownChan:   make(chan struct{}),
		dbpool:         dbpool,
		queries:        generated.New(),
		solveTimes:     make([]time.Duration, 0, 100),
		lastAdjustment: time.Now(),
		adaptiveMode:   cfg.AdaptiveMode,
		algorithm:      algorithm,
	}, nil
}

func (s *Server) Start() error {
	log.Printf("Server listening on %s with difficulty %d", s.listener.Addr(), s.difficulty)

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

	// Parse remote address
	remoteAddr, err := netip.ParseAddr(strings.Split(clientAddr, ":")[0])
	if err != nil {
		log.Printf("Failed to parse remote address %s: %v", clientAddr, err)
		return
	}

	// Create connection record in database
	ctx := context.Background()
	connectionRecord, err := s.logConnection(ctx, clientID, remoteAddr, s.algorithm)
	if err != nil {
		log.Printf("Failed to log connection: %v", err)
		// Continue anyway - don't fail the connection due to DB issues
	}

	// Record connection metrics
	metrics.RecordConnection("accepted")
	conn.SetDeadline(time.Now().Add(s.timeout))

	// Track connection rate for adaptive difficulty
	s.trackConnection()

	difficulty := s.getDifficulty()

	// Generate challenge based on algorithm
	var challengeStr string
	var verifySolution func(string) bool
	var challengeSeed string

	if s.algorithm == "sha256" {
		challenge, err := pow.GenerateChallenge(difficulty)
		if err != nil {
			log.Printf("Failed to generate challenge: %v", err)
			conn.Write([]byte("Error: Failed to generate challenge\n"))
			s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
			return
		}
		challengeStr = challenge.String()
		challengeSeed = challenge.Seed
		verifySolution = func(response string) bool {
			return pow.VerifyPoW(challenge.Seed, response, challenge.Difficulty)
		}
	} else {
		challenge, err := pow.GenerateArgon2Challenge(difficulty)
		if err != nil {
			log.Printf("Failed to generate Argon2 challenge: %v", err)
			conn.Write([]byte("Error: Failed to generate challenge\n"))
			s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
			return
		}
		challengeStr = challenge.String()
		challengeSeed = challenge.Seed
		verifySolution = func(response string) bool {
			return pow.VerifyArgon2PoW(challenge, response)
		}
	}

	// Log challenge to database
	challengeRecord, err := s.logChallenge(ctx, challengeSeed, int32(difficulty), s.algorithm, clientID)
	if err != nil {
		log.Printf("Failed to log challenge: %v", err)
		// Continue anyway
	}

	// Update connection status to solving
	s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusSolving)

	_, err = conn.Write([]byte(challengeStr + "\n"))
	if err != nil {
		log.Printf("Failed to send challenge to %s: %v", clientAddr, err)
		s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusFailed)
		return
	}

	solveStart := time.Now()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		log.Printf("Client %s disconnected or timed out", clientAddr)
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

	// Update final connection status
	s.updateConnectionStatus(ctx, connectionRecord.ID, generated.ConnectionStatusDisconnected)
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

// Database helper functions for write-only operations

func (s *Server) generateClientID(clientAddr string) string {
	return uuid.New().String()
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
