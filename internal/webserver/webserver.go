package webserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"world-of-wisdom/pkg/pow"
	"world-of-wisdom/pkg/wisdom"
)

type WebServer struct {
	tcpServerAddr string
	clients       map[*websocket.Conn]bool
	clientsMux    sync.RWMutex
	blockchain    []Block
	challenges    map[string]*Challenge
	connections   map[string]*ClientConnection
	stats         *MiningStats
	mu            sync.RWMutex
	upgrader      websocket.Upgrader
	quoteProvider *wisdom.QuoteProvider
	miningActive  bool
	stopMining    chan bool
}

type Challenge struct {
	ID        string    `json:"id"`
	Seed      string    `json:"seed"`
	Difficulty int      `json:"difficulty"`
	Timestamp int64     `json:"timestamp"`
	ClientID  string    `json:"clientId"`
	Status    string    `json:"status"`
}

type Solution struct {
	ChallengeID string `json:"challengeId"`
	Nonce       string `json:"nonce"`
	Hash        string `json:"hash"`
	Attempts    int    `json:"attempts"`
	TimeToSolve int64  `json:"timeToSolve"`
	Timestamp   int64  `json:"timestamp"`
}

type Block struct {
	Index        int        `json:"index"`
	Timestamp    int64      `json:"timestamp"`
	Challenge    *Challenge `json:"challenge"`
	Solution     *Solution  `json:"solution,omitempty"`
	Quote        string     `json:"quote,omitempty"`
	PreviousHash string     `json:"previousHash"`
	Hash         string     `json:"hash"`
}

type MiningStats struct {
	TotalChallenges     int     `json:"totalChallenges"`
	CompletedChallenges int     `json:"completedChallenges"`
	AverageSolveTime    float64 `json:"averageSolveTime"`
	CurrentDifficulty   int     `json:"currentDifficulty"`
	HashRate            float64 `json:"hashRate"`
}

type ClientConnection struct {
	ID                  string `json:"id"`
	Address             string `json:"address"`
	ConnectedAt         int64  `json:"connectedAt"`
	Status              string `json:"status"`
	ChallengesCompleted int    `json:"challengesCompleted"`
}

type MetricsData struct {
	Timestamp             int64   `json:"timestamp"`
	ConnectionsTotal      float64 `json:"connectionsTotal"`
	CurrentDifficulty     float64 `json:"currentDifficulty"`
	PuzzlesSolvedTotal    float64 `json:"puzzlesSolvedTotal"`
	PuzzlesFailedTotal    float64 `json:"puzzlesFailedTotal"`
	AverageSolveTime      float64 `json:"averageSolveTime"`
	ConnectionRate        float64 `json:"connectionRate"`
	DifficultyAdjustments float64 `json:"difficultyAdjustments"`
	ActiveConnections     float64 `json:"activeConnections"`
}

type WebSocketMessage struct {
	Type         string      `json:"type"`
	Block        *Block      `json:"block,omitempty"`
	Challenge    *Challenge  `json:"challenge,omitempty"`
	Connection   *ClientConnection `json:"connection,omitempty"`
	Stats        *MiningStats `json:"stats,omitempty"`
	Blocks       []Block     `json:"blocks,omitempty"`
	Connections  []ClientConnection `json:"connections,omitempty"`
	Metrics      *MetricsData `json:"metrics,omitempty"`
	MiningActive bool        `json:"miningActive,omitempty"`
}

func NewWebServer(tcpServerAddr string) *WebServer {
	return &WebServer{
		tcpServerAddr: tcpServerAddr,
		clients:       make(map[*websocket.Conn]bool),
		blockchain:    make([]Block, 0),
		challenges:    make(map[string]*Challenge),
		connections:   make(map[string]*ClientConnection),
		stats: &MiningStats{
			CurrentDifficulty: 2,
			AverageSolveTime:  0,
			HashRate:          0,
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		quoteProvider: wisdom.NewQuoteProvider(),
		miningActive:  false,
		stopMining:    make(chan bool, 10),
	}
}

func (ws *WebServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ws.clientsMux.Lock()
	ws.clients[conn] = true
	ws.clientsMux.Unlock()

	ws.sendInitialData(conn)

	defer func() {
		ws.clientsMux.Lock()
		delete(ws.clients, conn)
		ws.clientsMux.Unlock()
	}()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "simulate_client":
				log.Printf("Received simulate_client request, starting simulation...")
				go ws.simulateClient()
			case "start_mining":
				log.Printf("Received start_mining request...")
				ws.startContinuousMining()
			case "stop_mining":
				log.Printf("Received stop_mining request...")
				ws.stopContinuousMining()
			}
		}
	}
}

func (ws *WebServer) sendInitialData(conn *websocket.Conn) {
	ws.mu.RLock()
	connections := make([]ClientConnection, 0, len(ws.connections))
	for _, c := range ws.connections {
		connections = append(connections, *c)
	}
	
	// Create a copy of stats to avoid modifying the original
	statsCopy := *ws.stats
	ws.mu.RUnlock()

	msg := WebSocketMessage{
		Type:        "init",
		Blocks:      ws.blockchain,
		Connections: connections,
		Stats:       &statsCopy,
	}

	// Sanitize before sending
	if msg.Stats != nil {
		sanitizeStats(msg.Stats)
	}

	conn.WriteJSON(msg)
}

func sanitizeFloat(f float64) float64 {
	if math.IsInf(f, 0) || math.IsNaN(f) {
		return 0
	}
	return f
}

func sanitizeMetrics(metrics *MetricsData) {
	if metrics == nil {
		return
	}
	metrics.ConnectionsTotal = sanitizeFloat(metrics.ConnectionsTotal)
	metrics.CurrentDifficulty = sanitizeFloat(metrics.CurrentDifficulty)
	metrics.PuzzlesSolvedTotal = sanitizeFloat(metrics.PuzzlesSolvedTotal)
	metrics.PuzzlesFailedTotal = sanitizeFloat(metrics.PuzzlesFailedTotal)
	metrics.AverageSolveTime = sanitizeFloat(metrics.AverageSolveTime)
	metrics.ConnectionRate = sanitizeFloat(metrics.ConnectionRate)
	metrics.DifficultyAdjustments = sanitizeFloat(metrics.DifficultyAdjustments)
	metrics.ActiveConnections = sanitizeFloat(metrics.ActiveConnections)
}

func sanitizeStats(stats *MiningStats) {
	if stats == nil {
		return
	}
	stats.AverageSolveTime = sanitizeFloat(stats.AverageSolveTime)
	stats.HashRate = sanitizeFloat(stats.HashRate)
}

func (ws *WebServer) broadcast(msg WebSocketMessage) {
	// Sanitize any metrics or stats before broadcasting
	if msg.Metrics != nil {
		log.Printf("Sanitizing metrics before broadcast")
		sanitizeMetrics(msg.Metrics)
	}
	if msg.Stats != nil {
		log.Printf("Sanitizing stats before broadcast: HashRate=%f, AvgSolveTime=%f", msg.Stats.HashRate, msg.Stats.AverageSolveTime)
		sanitizeStats(msg.Stats)
		log.Printf("After sanitization: HashRate=%f, AvgSolveTime=%f", msg.Stats.HashRate, msg.Stats.AverageSolveTime)
	}
	
	ws.clientsMux.RLock()
	defer ws.clientsMux.RUnlock()

	for conn := range ws.clients {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(ws.clients, conn)
		}
	}
}

func (ws *WebServer) startContinuousMining() {
	ws.mu.Lock()
	if ws.miningActive {
		log.Printf("Continuous mining already active, ignoring start request")
		ws.mu.Unlock()
		return
	}
	ws.miningActive = true
	ws.mu.Unlock()

	log.Printf("🚀 Starting continuous mining simulation...")
	log.Printf("⛏️  Auto-spawning miners every 2-6 seconds")
	log.Printf("📈 Difficulty will adapt based on network activity")

	go func() {
		ticker := time.NewTicker(time.Duration(2+rand.Intn(4)) * time.Second)
		defer ticker.Stop()
		clientCount := 0

		for {
			select {
			case <-ws.stopMining:
				ws.mu.Lock()
				ws.miningActive = false
				ws.mu.Unlock()
				log.Printf("⏹️  Continuous mining stopped after %d clients", clientCount)
				// Drain any remaining signals
				for len(ws.stopMining) > 0 {
					<-ws.stopMining
				}
				return
			case <-ticker.C:
				// Check if mining is still active before spawning
				ws.mu.RLock()
				active := ws.miningActive
				ws.mu.RUnlock()
				
				if !active {
					log.Printf("⏹️  Mining deactivated, stopping goroutine")
					return
				}
				
				clientCount++
				log.Printf("🔨 Spawning mining client #%d...", clientCount)
				ws.simulateClient()
				nextInterval := time.Duration(2+rand.Intn(4)) * time.Second
				log.Printf("⏱️  Next client will spawn in %v", nextInterval)
				ticker.Reset(nextInterval)
			}
		}
	}()

	ws.broadcast(WebSocketMessage{
		Type:         "mining_status",
		MiningActive: true,
		Stats: &MiningStats{
			CurrentDifficulty: ws.stats.CurrentDifficulty,
		},
	})
}

func (ws *WebServer) stopContinuousMining() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	
	if !ws.miningActive {
		log.Printf("Continuous mining not active, ignoring stop request")
		return
	}

	log.Printf("🛑 Stopping continuous mining simulation...")

	select {
	case ws.stopMining <- true:
		log.Printf("✅ Stop signal sent to mining goroutine")
	default:
		log.Printf("⚠️  Stop signal channel full, mining may already be stopping")
	}

	ws.broadcast(WebSocketMessage{
		Type:         "mining_status",
		MiningActive: false,
		Stats: &MiningStats{
			CurrentDifficulty: ws.stats.CurrentDifficulty,
		},
	})
}

func (ws *WebServer) simulateClient() {
	clientID := fmt.Sprintf("sim-%d", rand.Intn(10000))
	
	connection := &ClientConnection{
		ID:          clientID,
		Address:     "127.0.0.1:simulated",
		ConnectedAt: time.Now().Unix(),
		Status:      "connected",
	}

	ws.mu.Lock()
	ws.connections[clientID] = connection
	ws.mu.Unlock()

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	challenge, err := pow.GenerateChallenge(ws.stats.CurrentDifficulty)
	if err != nil {
		log.Printf("Failed to generate challenge: %v", err)
		return
	}

	webChallenge := &Challenge{
		ID:         fmt.Sprintf("challenge-%d", time.Now().UnixNano()),
		Seed:       challenge.Seed,
		Difficulty: challenge.Difficulty,
		Timestamp:  time.Now().UnixMilli(),
		ClientID:   clientID,
		Status:     "solving",
	}

	ws.mu.Lock()
	ws.challenges[webChallenge.ID] = webChallenge
	ws.stats.TotalChallenges++
	ws.mu.Unlock()

	connection.Status = "solving"
	ws.broadcast(WebSocketMessage{
		Type:       "challenge",
		Challenge:  webChallenge,
	})

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	log.Printf("⛏️  Client %s solving difficulty %d challenge...", clientID, challenge.Difficulty)
	start := time.Now()
	solution, err := pow.SolveChallenge(challenge)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("❌ Client %s failed to solve challenge: %v", clientID, err)
		webChallenge.Status = "failed"
		connection.Status = "disconnected"
	} else {
		log.Printf("✅ Client %s solved challenge in %v (difficulty %d)", clientID, elapsed, challenge.Difficulty)
		webChallenge.Status = "completed"
		connection.Status = "connected"
		connection.ChallengesCompleted++

		data := challenge.Seed + solution
		hash := sha256.Sum256([]byte(data))
		hashHex := hex.EncodeToString(hash[:])

		sol := &Solution{
			ChallengeID: webChallenge.ID,
			Nonce:       solution,
			Hash:        hashHex,
			Attempts:    1000,
			TimeToSolve: elapsed.Milliseconds(),
			Timestamp:   time.Now().UnixMilli(),
		}

		quote := ws.quoteProvider.GetRandomQuote()

		previousHash := "0000000000000000000000000000000000000000000000000000000000000000"
		if len(ws.blockchain) > 0 {
			previousHash = ws.blockchain[len(ws.blockchain)-1].Hash
		}

		blockData := fmt.Sprintf("%d%d%s%s", len(ws.blockchain), time.Now().Unix(), previousHash, hashHex)
		blockHash := sha256.Sum256([]byte(blockData))
		blockHashHex := hex.EncodeToString(blockHash[:])

		block := Block{
			Index:        len(ws.blockchain),
			Timestamp:    time.Now().UnixMilli(),
			Challenge:    webChallenge,
			Solution:     sol,
			Quote:        quote,
			PreviousHash: previousHash,
			Hash:         blockHashHex,
		}

		ws.mu.Lock()
		ws.blockchain = append(ws.blockchain, block)
		ws.stats.CompletedChallenges++
		log.Printf("🎉 New block #%d mined! Hash: %s", block.Index, block.Hash[:16]+"...")
		newAvgSolveTime := (ws.stats.AverageSolveTime*float64(ws.stats.CompletedChallenges-1) + float64(elapsed.Milliseconds())) / float64(ws.stats.CompletedChallenges)
		if math.IsInf(newAvgSolveTime, 0) || math.IsNaN(newAvgSolveTime) {
			ws.stats.AverageSolveTime = float64(elapsed.Milliseconds())
		} else {
			ws.stats.AverageSolveTime = newAvgSolveTime
		}
		
		// Prevent division by zero for HashRate calculation
		elapsedSeconds := elapsed.Seconds()
		if elapsedSeconds > 0.001 { // Minimum 1ms to avoid huge hash rates
			ws.stats.HashRate = 1000.0 / elapsedSeconds
		} else {
			ws.stats.HashRate = 1000000.0 // Cap at 1M hashes/sec for very fast solves
		}
		
		// Additional safety check
		if math.IsInf(ws.stats.HashRate, 0) || math.IsNaN(ws.stats.HashRate) {
			ws.stats.HashRate = 0
		}
		ws.mu.Unlock()

		ws.broadcast(WebSocketMessage{
			Type:  "block",
			Block: &block,
		})
	}

	ws.broadcast(WebSocketMessage{
		Type:       "challenge_update",
		Challenge:  webChallenge,
	})

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	ws.broadcast(WebSocketMessage{
		Type:  "stats",
		Stats: ws.stats,
	})
}

func (ws *WebServer) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws.stats)
}

func (ws *WebServer) HandleSimulateClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	go ws.simulateClient()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func (ws *WebServer) fetchPrometheusMetrics() (*MetricsData, error) {
	resp, err := http.Get("http://server:2112/metrics")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	metrics := &MetricsData{
		Timestamp: time.Now().UnixMilli(),
	}

	lines := strings.Split(string(body), "\n")
	log.Printf("Processing %d lines from Prometheus metrics", len(lines))
	for _, line := range lines {
		if strings.HasPrefix(line, "wisdom_connections_total") && strings.Contains(line, `status="accepted"`) {
			if value := extractMetricValue(line); value != -1 {
				metrics.ConnectionsTotal = value
			}
		} else if strings.HasPrefix(line, "wisdom_current_difficulty") && !strings.HasPrefix(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.CurrentDifficulty = value
			}
		} else if strings.HasPrefix(line, "wisdom_puzzles_solved_total") && !strings.Contains(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.PuzzlesSolvedTotal += value
			}
		} else if strings.HasPrefix(line, "wisdom_puzzles_failed_total") && !strings.Contains(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.PuzzlesFailedTotal += value
			}
		} else if strings.HasPrefix(line, "wisdom_average_solve_time_seconds") && !strings.HasPrefix(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.AverageSolveTime = value * 1000 // Convert to ms
			}
		} else if strings.HasPrefix(line, "wisdom_connection_rate_per_minute") && !strings.HasPrefix(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.ConnectionRate = value
			}
		} else if strings.HasPrefix(line, "wisdom_difficulty_adjustments_total") && !strings.Contains(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.DifficultyAdjustments += value
			}
		} else if strings.HasPrefix(line, "wisdom_active_connections") && !strings.HasPrefix(line, "#") {
			if value := extractMetricValue(line); value != -1 {
				metrics.ActiveConnections = value
			}
		}
	}

	return metrics, nil
}

func extractMetricValue(line string) float64 {
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return -1
	}
	
	valueStr := parts[len(parts)-1]
	
	// Handle special Prometheus values
	if valueStr == "+Inf" || valueStr == "-Inf" || valueStr == "NaN" {
		return -1
	}
	
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return -1
	}
	
	// Additional check for infinity and NaN after parsing
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return -1
	}
	
	return value
}

func (ws *WebServer) startMetricsBroadcast() {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			// Temporarily disable Prometheus metrics to test basic WebSocket
			log.Printf("Metrics broadcast disabled for debugging")
			
			// Send dummy metrics to test JSON serialization
			metrics := &MetricsData{
				Timestamp:         time.Now().UnixMilli(),
				ConnectionsTotal:  float64(len(ws.connections)),
				CurrentDifficulty: float64(ws.stats.CurrentDifficulty),
				PuzzlesSolvedTotal: float64(ws.stats.CompletedChallenges),
				PuzzlesFailedTotal: 0,
				AverageSolveTime:   ws.stats.AverageSolveTime,
				ConnectionRate:     0,
				DifficultyAdjustments: 0,
				ActiveConnections: float64(len(ws.connections)),
			}

			ws.broadcast(WebSocketMessage{
				Type:    "metrics",
				Metrics: metrics,
			})
		}
	}()
}

func (ws *WebServer) Start() {
	log.Println("WebServer started")
	ws.startMetricsBroadcast()
}

func (ws *WebServer) Stop() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()
	
	for conn := range ws.clients {
		conn.Close()
	}
}