package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"world-of-wisdom/pkg/pow"
)

// Stats holds load testing statistics
type Stats struct {
	totalConnections  int64
	successfulSolves  int64
	failedConnections int64
	totalSolveTime    int64
	minSolveTime      int64
	maxSolveTime      int64
	averageDifficulty float64
	connectionsPerSec float64
	solvesPerSec      float64
	mu                sync.RWMutex
}

func (s *Stats) recordConnection(success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddInt64(&s.totalConnections, 1)
	if !success {
		atomic.AddInt64(&s.failedConnections, 1)
	}
}

func (s *Stats) recordSolve(duration time.Duration, difficulty int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	atomic.AddInt64(&s.successfulSolves, 1)
	durationMs := duration.Milliseconds()
	atomic.AddInt64(&s.totalSolveTime, durationMs)

	// Update min/max solve times
	if s.minSolveTime == 0 || durationMs < s.minSolveTime {
		s.minSolveTime = durationMs
	}
	if durationMs > s.maxSolveTime {
		s.maxSolveTime = durationMs
	}

	// Update average difficulty (simple moving average)
	count := atomic.LoadInt64(&s.successfulSolves)
	s.averageDifficulty = (s.averageDifficulty*float64(count-1) + float64(difficulty)) / float64(count)
}

func (s *Stats) getStats() (int64, int64, int64, float64, float64, float64, int64, int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	totalConn := atomic.LoadInt64(&s.totalConnections)
	successSolves := atomic.LoadInt64(&s.successfulSolves)
	failedConn := atomic.LoadInt64(&s.failedConnections)
	totalTime := atomic.LoadInt64(&s.totalSolveTime)

	var avgSolveTime float64
	if successSolves > 0 {
		avgSolveTime = float64(totalTime) / float64(successSolves)
	}

	return totalConn, successSolves, failedConn, avgSolveTime, s.averageDifficulty, s.connectionsPerSec, s.minSolveTime, s.maxSolveTime
}

func (s *Stats) updateRates(startTime time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	elapsed := time.Since(startTime).Seconds()
	if elapsed > 0 {
		s.connectionsPerSec = float64(atomic.LoadInt64(&s.totalConnections)) / elapsed
		s.solvesPerSec = float64(atomic.LoadInt64(&s.successfulSolves)) / elapsed
	}
}

// LoadClient represents a load testing client
type LoadClient struct {
	serverAddr    string
	algorithm     string
	maxDifficulty int
	timeout       time.Duration
	stats         *Stats
	clientID      string
}

func NewLoadClient(serverAddr, algorithm string, maxDiff int, timeout time.Duration) *LoadClient {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	clientID := fmt.Sprintf("%s-%d-%d", hostname, pid, time.Now().Unix())

	return &LoadClient{
		serverAddr:    serverAddr,
		algorithm:     algorithm,
		maxDifficulty: maxDiff,
		timeout:       timeout,
		stats:         &Stats{},
		clientID:      clientID,
	}
}

func (lc *LoadClient) connect() error {
	conn, err := net.DialTimeout("tcp", lc.serverAddr, lc.timeout)
	if err != nil {
		lc.stats.recordConnection(false)
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	lc.stats.recordConnection(true)

	// Set connection timeout
	conn.SetDeadline(time.Now().Add(lc.timeout))

	// Receive challenge
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read challenge: %w", err)
	}

	challenge := string(buffer[:n])
	log.Printf("[%s] Received challenge: %s", lc.clientID, challenge)

	// Parse challenge and solve
	startTime := time.Now()
	solution, difficulty, err := lc.solveChallenge(challenge)
	solveTime := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("failed to solve challenge: %w", err)
	}

	lc.stats.recordSolve(solveTime, difficulty)

	// Send solution
	_, err = conn.Write([]byte(solution))
	if err != nil {
		return fmt.Errorf("failed to send solution: %w", err)
	}

	// Receive wisdom
	n, err = conn.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to read wisdom: %w", err)
	}

	wisdom := string(buffer[:n])
	log.Printf("[%s] Received wisdom (difficulty %d, solved in %v): %s",
		lc.clientID, difficulty, solveTime, wisdom)

	return nil
}

func (lc *LoadClient) solveChallenge(challenge string) (string, int, error) {
	// Simple challenge parsing - extract seed and difficulty from challenge string
	// Expected format: "Solve PoW: <seed> with prefix <zeros>"
	parts := []string{}
	current := ""
	for _, char := range challenge {
		if char == ' ' || char == '\n' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	if len(parts) < 3 {
		return "", 0, fmt.Errorf("invalid challenge format")
	}

	seed := parts[2] // "Solve PoW: <seed> ..."

	// Extract difficulty from the number of zeros in prefix
	difficulty := 1
	for i, part := range parts {
		if part == "prefix" && i+1 < len(parts) {
			zeros := parts[i+1]
			difficulty = len(zeros)
			break
		}
	}

	// For this simplified version, we'll use the basic SHA-256 PoW
	// In the real implementation, the server would specify the algorithm
	powChallenge := &pow.Challenge{
		Seed:       seed,
		Difficulty: difficulty,
	}

	solution, err := pow.SolveChallenge(powChallenge)
	if err != nil {
		return "", 0, fmt.Errorf("failed to solve challenge: %w", err)
	}

	return solution, difficulty, nil
}

func (lc *LoadClient) runContinuous(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := lc.connect(); err != nil {
				log.Printf("[%s] Connection error: %v", lc.clientID, err)
			}
		}
	}
}

func (lc *LoadClient) runBurst(ctx context.Context, burstSize int, burstInterval time.Duration) {
	ticker := time.NewTicker(burstInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			log.Printf("[%s] Starting burst of %d connections", lc.clientID, burstSize)

			var wg sync.WaitGroup
			for i := 0; i < burstSize; i++ {
				wg.Add(1)
				go func(id int) {
					defer wg.Done()
					if err := lc.connect(); err != nil {
						log.Printf("[%s] Burst connection %d error: %v", lc.clientID, id, err)
					}
				}(i)
			}
			wg.Wait()

			log.Printf("[%s] Burst completed", lc.clientID)
		}
	}
}

func (lc *LoadClient) printStats(startTime time.Time) {
	lc.stats.updateRates(startTime)

	totalConn, successSolves, failedConn, avgSolveTime, avgDiff, connPerSec, minSolve, maxSolve := lc.stats.getStats()

	successRate := float64(successSolves) / float64(totalConn) * 100

	fmt.Printf("\n[%s] === Load Test Statistics ===\n", lc.clientID)
	fmt.Printf("Total Connections: %d\n", totalConn)
	fmt.Printf("Successful Solves: %d\n", successSolves)
	fmt.Printf("Failed Connections: %d\n", failedConn)
	fmt.Printf("Success Rate: %.2f%%\n", successRate)
	fmt.Printf("Connections/sec: %.2f\n", connPerSec)
	fmt.Printf("Solves/sec: %.2f\n", lc.stats.solvesPerSec)
	fmt.Printf("Average Solve Time: %.2f ms\n", avgSolveTime)
	fmt.Printf("Min Solve Time: %d ms\n", minSolve)
	fmt.Printf("Max Solve Time: %d ms\n", maxSolve)
	fmt.Printf("Average Difficulty: %.2f\n", avgDiff)
	fmt.Printf("Test Duration: %.2f seconds\n", time.Since(startTime).Seconds())
	fmt.Printf("=============================\n\n")
}

func main() {
	var (
		serverAddr    = flag.String("server", "localhost:8080", "Server address")
		algorithm     = flag.String("algorithm", "argon2", "PoW algorithm (sha256 or argon2)")
		maxDifficulty = flag.Int("max-difficulty", 6, "Maximum difficulty to handle")
		duration      = flag.Duration("duration", 5*time.Minute, "Test duration")
		mode          = flag.String("mode", "continuous", "Test mode: continuous, burst, mixed")
		interval      = flag.Duration("interval", 5*time.Second, "Connection interval for continuous mode")
		burstSize     = flag.Int("burst-size", 10, "Number of connections per burst")
		burstInterval = flag.Duration("burst-interval", 30*time.Second, "Interval between bursts")
		concurrency   = flag.Int("concurrency", 1, "Number of concurrent clients")
		timeout       = flag.Duration("timeout", 30*time.Second, "Connection timeout")
		statsInterval = flag.Duration("stats-interval", 30*time.Second, "Statistics reporting interval")
	)
	flag.Parse()

	log.Printf("Starting load client: %s", *mode)
	log.Printf("Target: %s", *serverAddr)
	log.Printf("Algorithm: %s", *algorithm)
	log.Printf("Duration: %v", *duration)
	log.Printf("Concurrency: %d", *concurrency)

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	startTime := time.Now()
	var wg sync.WaitGroup

	// Create multiple concurrent clients
	clients := make([]*LoadClient, *concurrency)
	for i := 0; i < *concurrency; i++ {
		clients[i] = NewLoadClient(*serverAddr, *algorithm, *maxDifficulty, *timeout)
	}

	// Start clients based on mode
	for i, client := range clients {
		wg.Add(1)
		go func(id int, lc *LoadClient) {
			defer wg.Done()

			switch *mode {
			case "continuous":
				lc.runContinuous(ctx, *interval)
			case "burst":
				lc.runBurst(ctx, *burstSize, *burstInterval)
			case "mixed":
				// Alternate between continuous and burst
				if id%2 == 0 {
					lc.runContinuous(ctx, *interval)
				} else {
					lc.runBurst(ctx, *burstSize/2, *burstInterval*2)
				}
			default:
				log.Printf("Unknown mode: %s", *mode)
				return
			}
		}(i, client)
	}

	// Statistics reporting
	go func() {
		ticker := time.NewTicker(*statsInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Print combined stats from all clients
				for i, client := range clients {
					fmt.Printf("\n--- Client %d ---\n", i+1)
					client.printStats(startTime)
				}
			}
		}
	}()

	// Wait for test completion
	wg.Wait()

	// Final statistics
	fmt.Println("\n=== FINAL LOAD TEST RESULTS ===")
	for i, client := range clients {
		fmt.Printf("\n--- Client %d Final Stats ---\n", i+1)
		client.printStats(startTime)
	}

	// Combined statistics
	var totalConnections, totalSolves, totalFailed int64
	var totalSolveTime int64
	for _, client := range clients {
		conn, solves, failed, _, _, _, _, _ := client.stats.getStats()
		totalConnections += conn
		totalSolves += solves
		totalFailed += failed
		totalSolveTime += atomic.LoadInt64(&client.stats.totalSolveTime)
	}

	overallSuccessRate := float64(totalSolves) / float64(totalConnections) * 100
	overallAvgSolveTime := float64(totalSolveTime) / float64(totalSolves)
	elapsed := time.Since(startTime).Seconds()
	overallThroughput := float64(totalSolves) / elapsed

	fmt.Printf("\n=== OVERALL STATISTICS ===\n")
	fmt.Printf("Total Clients: %d\n", *concurrency)
	fmt.Printf("Total Connections: %d\n", totalConnections)
	fmt.Printf("Total Successful Solves: %d\n", totalSolves)
	fmt.Printf("Total Failed Connections: %d\n", totalFailed)
	fmt.Printf("Overall Success Rate: %.2f%%\n", overallSuccessRate)
	fmt.Printf("Overall Average Solve Time: %.2f ms\n", overallAvgSolveTime)
	fmt.Printf("Overall Throughput: %.2f solves/sec\n", overallThroughput)
	fmt.Printf("Test Duration: %.2f seconds\n", elapsed)
	fmt.Printf("========================\n")
}
