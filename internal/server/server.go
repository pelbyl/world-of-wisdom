package server

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"world-of-wisdom/pkg/metrics"
	"world-of-wisdom/pkg/pow"
	"world-of-wisdom/pkg/wisdom"
)

type Server struct {
	listener       net.Listener
	quoteProvider  *wisdom.QuoteProvider
	difficulty     int
	timeout        time.Duration
	mu             sync.RWMutex
	activeConns    sync.WaitGroup
	shutdownChan   chan struct{}
	
	// Adaptive difficulty tracking
	solveTimes     []time.Duration
	connectionRate int64
	lastAdjustment time.Time
	adaptiveMode   bool
	
	// PoW algorithm selection
	algorithm      string // "sha256" or "argon2"
}

type Config struct {
	Port         string
	Difficulty   int
	Timeout      time.Duration
	AdaptiveMode bool
	MetricsPort  string
	Algorithm    string // "sha256" or "argon2"
}

func NewServer(cfg Config) (*Server, error) {
	listener, err := net.Listen("tcp", cfg.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", cfg.Port, err)
	}

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
	log.Printf("New connection from %s", clientAddr)

	// Record connection
	metrics.RecordConnection("accepted")

	conn.SetDeadline(time.Now().Add(s.timeout))

	// Track connection rate for adaptive difficulty
	s.trackConnection()

	difficulty := s.getDifficulty()
	
	// Generate challenge based on algorithm
	var challengeStr string
	var verifySolution func(string) bool
	
	if s.algorithm == "sha256" {
		challenge, err := pow.GenerateChallenge(difficulty)
		if err != nil {
			log.Printf("Failed to generate challenge: %v", err)
			conn.Write([]byte("Error: Failed to generate challenge\n"))
			return
		}
		challengeStr = challenge.String()
		verifySolution = func(response string) bool {
			return pow.VerifyPoW(challenge.Seed, response, challenge.Difficulty)
		}
	} else {
		challenge, err := pow.GenerateArgon2Challenge(difficulty)
		if err != nil {
			log.Printf("Failed to generate Argon2 challenge: %v", err)
			conn.Write([]byte("Error: Failed to generate challenge\n"))
			return
		}
		challengeStr = challenge.String()
		verifySolution = func(response string) bool {
			return pow.VerifyArgon2PoW(challenge, response)
		}
	}

	_, err := conn.Write([]byte(challengeStr + "\n"))
	if err != nil {
		log.Printf("Failed to send challenge to %s: %v", clientAddr, err)
		return
	}

	solveStart := time.Now()
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		log.Printf("Client %s disconnected or timed out", clientAddr)
		return
	}

	response := strings.TrimSpace(scanner.Text())
	solveTime := time.Since(solveStart)

	if verifySolution(response) {
		log.Printf("Client %s solved the %s challenge in %v", clientAddr, s.algorithm, solveTime)
		s.recordSolveTime(solveTime)
		
		// Record metrics
		metrics.RecordPuzzleSolved(difficulty, solveTime)
		metrics.RecordProcessingTime("success", time.Since(startTime))
		
		quote := s.quoteProvider.GetRandomQuote()
		conn.Write([]byte(quote + "\n"))
	} else {
		log.Printf("Client %s failed the %s challenge", clientAddr, s.algorithm)
		
		// Record metrics
		metrics.RecordPuzzleFailed(difficulty)
		metrics.RecordProcessingTime("failed", time.Since(startTime))
		
		conn.Write([]byte("Error: Invalid proof of work\n"))
	}
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
		"difficulty":           s.difficulty,
		"adaptive_mode":        s.adaptiveMode,
		"avg_solve_time_ms":    avgSolveTime.Milliseconds(),
		"connection_rate":      connectionRatePerMinute,
		"recent_solve_count":   len(s.solveTimes),
		"last_adjustment":      s.lastAdjustment.Unix(),
	}
}

func (s *Server) Addr() string {
	return s.listener.Addr().String()
}