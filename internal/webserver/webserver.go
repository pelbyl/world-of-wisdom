package webserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"world-of-wisdom/api/db"
	"world-of-wisdom/pkg/pow"
	"world-of-wisdom/pkg/wisdom"
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type WebServer struct {
	tcpServerAddr    string
	clients          map[*websocket.Conn]bool
	clientsMux       sync.RWMutex
	blockchain       []Block
	challenges       map[string]*Challenge
	connections      map[string]*ClientConnection
	recentLogs       []LogMessage // Store recent logs for stateless operation
	stats            *MiningStats
	mu               sync.RWMutex
	upgrader         websocket.Upgrader
	quoteProvider    *wisdom.QuoteProvider
	miningActive     bool
	stopMining       chan bool
	totalConnections int
	miningIntensity  int           // 1=low, 2=medium, 3=high, 4=extreme intensity
	activeMinerCount int           // concurrent miners
	cpuCores         int           // available CPU cores
	maxConcurrency   int           // resource-based max miners
	dataDir          string        // directory for persistent storage
	algorithm        string        // "sha256" or "argon2"
	db               *pgxpool.Pool // Database connection pool
	queries          *db.Queries   // SQLC generated queries

	// WebSocket load management
	messageQueue    chan WebSocketMessage
	isHighLoad      bool
	lastBroadcast   time.Time
	broadcastMux    sync.Mutex
	batchedMessages []WebSocketMessage
}

type Challenge struct {
	ID         string `json:"id"`
	Seed       string `json:"seed"`
	Difficulty int    `json:"difficulty"`
	Timestamp  int64  `json:"timestamp"`
	ClientID   string `json:"clientId"`
	Status     string `json:"status"`
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
	TotalChallenges      int     `json:"totalChallenges"`
	CompletedChallenges  int     `json:"completedChallenges"`
	AverageSolveTime     float64 `json:"averageSolveTime"`
	CurrentDifficulty    int     `json:"currentDifficulty"`
	HashRate             float64 `json:"hashRate"`
	LiveConnections      int     `json:"liveConnections,omitempty"`
	TotalConnections     int     `json:"totalConnections,omitempty"`
	NetworkIntensity     int     `json:"networkIntensity,omitempty"`
	DDosProtectionActive bool    `json:"ddosProtectionActive,omitempty"`
	ActiveMinerCount     int     `json:"activeMinerCount,omitempty"`
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
	Type         string             `json:"type"`
	Block        *Block             `json:"block,omitempty"`
	Challenge    *Challenge         `json:"challenge,omitempty"`
	Connection   *ClientConnection  `json:"connection,omitempty"`
	Stats        *MiningStats       `json:"stats,omitempty"`
	Blocks       []Block            `json:"blocks,omitempty"`
	Connections  []ClientConnection `json:"connections,omitempty"`
	Challenges   []Challenge        `json:"challenges,omitempty"`
	Logs         []LogMessage       `json:"logs,omitempty"`
	Metrics      *MetricsData       `json:"metrics,omitempty"`
	MiningActive bool               `json:"miningActive,omitempty"`
	Log          *LogMessage        `json:"log,omitempty"`
}

type LogMessage struct {
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"` // info, success, warning, error
	Message   string `json:"message"`
	Icon      string `json:"icon,omitempty"`
}

type PersistentData struct {
	Blockchain       []Block                      `json:"blockchain"`
	Stats            *MiningStats                 `json:"stats"`
	TotalConnections int                          `json:"totalConnections"`
	LastUpdated      int64                        `json:"lastUpdated"`
	Connections      map[string]*ClientConnection `json:"connections"`
}

func NewWebServer(tcpServerAddr string, algorithm string, dbpool *pgxpool.Pool, queries *db.Queries) *WebServer {
	cpuCores := runtime.NumCPU()
	maxConcurrency := cpuCores * 10 // 10 miners per CPU core

	log.Printf("🖥️  Detected %d CPU cores, max concurrency: %d miners", cpuCores, maxConcurrency)

	// Default to argon2 if not specified
	if algorithm == "" {
		algorithm = "argon2"
	}

	ws := &WebServer{
		tcpServerAddr: tcpServerAddr,
		clients:       make(map[*websocket.Conn]bool),
		blockchain:    make([]Block, 0),
		challenges:    make(map[string]*Challenge),
		connections:   make(map[string]*ClientConnection),
		recentLogs:    make([]LogMessage, 0, 100),
		stats: &MiningStats{
			CurrentDifficulty: 2,
			AverageSolveTime:  0,
			HashRate:          0,
		},
		dataDir: "/tmp/wisdom-data",
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		quoteProvider:    wisdom.NewQuoteProvider(),
		miningActive:     false,
		stopMining:       make(chan bool, 10),
		totalConnections: 0,
		miningIntensity:  1,
		activeMinerCount: 0,
		cpuCores:         cpuCores,
		maxConcurrency:   maxConcurrency,
		algorithm:        algorithm,
		db:               dbpool,
		queries:          queries,

		// WebSocket load management
		messageQueue:    make(chan WebSocketMessage, 1000),
		isHighLoad:      false,
		lastBroadcast:   time.Now(),
		batchedMessages: make([]WebSocketMessage, 0, 50),
	}

	// Create data directory and load persistent data
	os.MkdirAll(ws.dataDir, 0755)
	ws.loadData()

	// Start WebSocket message processing goroutine
	go ws.startMessageProcessor()

	return ws
}

// Load persistent data from disk
func (ws *WebServer) loadData() {
	filePath := ws.dataDir + "/blockchain.json"
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("📂 No existing data found, starting fresh")
		return
	}

	var persistentData PersistentData
	if err := json.Unmarshal(data, &persistentData); err != nil {
		log.Printf("❌ Failed to load data: %v", err)
		return
	}

	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.blockchain = persistentData.Blockchain
	if persistentData.Stats != nil {
		ws.stats = persistentData.Stats
	}
	ws.totalConnections = persistentData.TotalConnections
	if persistentData.Connections != nil {
		ws.connections = persistentData.Connections
	}

	log.Printf("📂 Loaded %d blocks and %d connections from storage", len(ws.blockchain), len(ws.connections))
}

// Save current state to disk
func (ws *WebServer) saveData() {
	ws.mu.RLock()
	persistentData := PersistentData{
		Blockchain:       ws.blockchain,
		Stats:            ws.stats,
		TotalConnections: ws.totalConnections,
		LastUpdated:      time.Now().UnixMilli(),
		Connections:      ws.connections,
	}
	ws.mu.RUnlock()

	data, err := json.MarshalIndent(persistentData, "", "  ")
	if err != nil {
		log.Printf("❌ Failed to serialize data: %v", err)
		return
	}

	filePath := ws.dataDir + "/blockchain.json"
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		log.Printf("❌ Failed to save data: %v", err)
		return
	}
}

// Auto-save data periodically
func (ws *WebServer) startAutoSave() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			ws.saveData()
		}
	}()
}

func (ws *WebServer) broadcastLog(level, message, icon string) {
	logMsg := &LogMessage{
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Message:   message,
		Icon:      icon,
	}

	// Store log in database for stateless operation
	if ws.db != nil && ws.queries != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		timestamp := time.UnixMilli(logMsg.Timestamp)
		_, err := ws.queries.CreateLog(ctx, ws.db, db.CreateLogParams{
			Column1:  timestamp,
			Level:    level,
			Message:  message,
			Icon:     pgtype.Text{String: icon, Valid: icon != ""},
			Metadata: nil,
		})

		if err != nil {
			log.Printf("❌ Failed to store log in database: %v", err)
		}
	}

	ws.broadcast(WebSocketMessage{
		Type: "log",
		Log:  logMsg,
	})
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
				// Check for configuration parameters
				if config, ok := msg["config"].(map[string]interface{}); ok {
					ws.startContinuousMiningWithConfig(config)
				} else {
					ws.startContinuousMining()
				}
			case "stop_mining":
				log.Printf("Received stop_mining request...")
				ws.stopContinuousMining()
			case "clear_state":
				log.Printf("Received clear_state request...")
				ws.clearServerState()
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

	challenges := make([]Challenge, 0, len(ws.challenges))
	for _, c := range ws.challenges {
		challenges = append(challenges, *c)
	}

	// Create copies to avoid modifying the original
	statsCopy := *ws.stats
	logsCopy := make([]LogMessage, len(ws.recentLogs))
	copy(logsCopy, ws.recentLogs)
	ws.mu.RUnlock()

	msg := WebSocketMessage{
		Type:         "init",
		Blocks:       ws.blockchain,
		Connections:  connections,
		Challenges:   challenges,
		Logs:         logsCopy,
		Stats:        &statsCopy,
		MiningActive: ws.miningActive,
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

// Smart broadcast that handles high load gracefully
func (ws *WebServer) broadcast(msg WebSocketMessage) {
	// Sanitize any metrics or stats before broadcasting
	if msg.Metrics != nil {
		sanitizeMetrics(msg.Metrics)
	}
	if msg.Stats != nil {
		sanitizeStats(msg.Stats)
	}

	// During high load, queue messages instead of immediate broadcast
	ws.broadcastMux.Lock()
	if ws.isHighLoad || len(ws.messageQueue) > 500 {
		// Queue message for batched processing
		select {
		case ws.messageQueue <- msg:
			// Message queued successfully
		default:
			// Queue is full, drop non-critical messages
			if msg.Type != "mining_status" && msg.Type != "init" {
				log.Printf("⚠️ WebSocket queue full, dropping message type: %s", msg.Type)
			}
		}
		ws.broadcastMux.Unlock()
		return
	}
	ws.broadcastMux.Unlock()

	// Normal immediate broadcast for low load
	ws.immediateBroadcast(msg)
}

// Immediate broadcast for low load situations
func (ws *WebServer) immediateBroadcast(msg WebSocketMessage) {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	// Create a copy of clients to avoid concurrent map access
	clientsCopy := make([]*websocket.Conn, 0, len(ws.clients))
	for conn := range ws.clients {
		clientsCopy = append(clientsCopy, conn)
	}

	// Write to clients outside the map iteration
	for _, conn := range clientsCopy {
		err := conn.WriteJSON(msg)
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			conn.Close()
			delete(ws.clients, conn)
		}
	}
}

// Message processor for handling high load scenarios
func (ws *WebServer) startMessageProcessor() {
	batchTicker := time.NewTicker(250 * time.Millisecond) // Process batches every 250ms
	loadCheckTicker := time.NewTicker(2 * time.Second)    // Check load every 2 seconds

	defer batchTicker.Stop()
	defer loadCheckTicker.Stop()

	for {
		select {
		case msg := <-ws.messageQueue:
			ws.broadcastMux.Lock()
			ws.batchedMessages = append(ws.batchedMessages, msg)
			ws.broadcastMux.Unlock()

		case <-batchTicker.C:
			ws.processBatchedMessages()

		case <-loadCheckTicker.C:
			ws.checkAndUpdateLoadStatus()
		}
	}
}

// Process batched messages efficiently
func (ws *WebServer) processBatchedMessages() {
	ws.broadcastMux.Lock()
	if len(ws.batchedMessages) == 0 {
		ws.broadcastMux.Unlock()
		return
	}

	// Take a copy of batched messages and clear the slice
	messagesToProcess := make([]WebSocketMessage, len(ws.batchedMessages))
	copy(messagesToProcess, ws.batchedMessages)
	ws.batchedMessages = ws.batchedMessages[:0] // Clear slice but keep capacity
	ws.broadcastMux.Unlock()

	// Group messages by type for efficiency
	logMessages := []WebSocketMessage{}
	criticalMessages := []WebSocketMessage{}

	for _, msg := range messagesToProcess {
		switch msg.Type {
		case "mining_status", "init", "stats":
			criticalMessages = append(criticalMessages, msg)
		case "log":
			logMessages = append(logMessages, msg)
		default:
			criticalMessages = append(criticalMessages, msg)
		}
	}

	// Send critical messages immediately
	for _, msg := range criticalMessages {
		ws.immediateBroadcast(msg)
	}

	// Batch log messages if there are many
	if len(logMessages) > 10 {
		// Send only the latest 5 log messages to avoid overwhelming
		latestLogs := logMessages[len(logMessages)-5:]
		for _, msg := range latestLogs {
			ws.immediateBroadcast(msg)
		}
		if len(logMessages) > 5 {
			log.Printf("🔥 High load: Batched %d log messages, sent latest 5", len(logMessages))
		}
	} else {
		// Send all log messages if there aren't too many
		for _, msg := range logMessages {
			ws.immediateBroadcast(msg)
		}
	}
}

// Check current load and update status
func (ws *WebServer) checkAndUpdateLoadStatus() {
	ws.broadcastMux.Lock()
	queueSize := len(ws.messageQueue)
	ws.broadcastMux.Unlock()

	ws.mu.RLock()
	activeMinerCount := ws.activeMinerCount
	miningIntensity := ws.miningIntensity
	currentDifficulty := ws.stats.CurrentDifficulty
	ws.mu.RUnlock()

	// Determine if we're in high load situation
	wasHighLoad := ws.isHighLoad
	isCurrentlyHighLoad := queueSize > 100 || activeMinerCount > 15 || miningIntensity >= 3

	if isCurrentlyHighLoad != wasHighLoad {
		ws.broadcastMux.Lock()
		ws.isHighLoad = isCurrentlyHighLoad
		ws.broadcastMux.Unlock()

		if isCurrentlyHighLoad {
			log.Printf("🔥 High load detected: Queue=%d, Miners=%d, Intensity=%d, Difficulty=%d - Switching to batched mode",
				queueSize, activeMinerCount, miningIntensity, currentDifficulty)
		} else {
			log.Printf("✅ Load normalized: Queue=%d, Miners=%d, Intensity=%d, Difficulty=%d - Switching to real-time mode",
				queueSize, activeMinerCount, miningIntensity, currentDifficulty)
		}
	}

	// Additional difficulty-specific monitoring during periods of low activity
	if !isCurrentlyHighLoad && miningIntensity <= 1 && activeMinerCount < 3 && currentDifficulty > 2 {
		log.Printf("📉 Low activity detected: Miners=%d, Intensity=%d, Difficulty=%d - Consider reducing difficulty",
			activeMinerCount, miningIntensity, currentDifficulty)
	}
}

type MiningConfig struct {
	InitialIntensity int  `json:"initialIntensity"` // 1-4 (1=low, 2=medium, 3=high, 4=extreme)
	MaxIntensity     int  `json:"maxIntensity"`     // 1-4
	IntensityStep    int  `json:"intensityStep"`    // seconds between changes
	AutoScale        bool `json:"autoScale"`        // enable auto intensity scaling
	MinMiners        int  `json:"minMiners"`        // minimum concurrent miners
	MaxMiners        int  `json:"maxMiners"`        // maximum concurrent miners
	Duration         int  `json:"duration"`         // simulation duration in seconds (0 = infinite)
	HighPerformance  bool `json:"highPerformance"`  // enable high-performance mode
	MaxDifficulty    int  `json:"maxDifficulty"`    // maximum PoW difficulty (1-8)
	CPUIntensive     bool `json:"cpuIntensive"`     // use maximum CPU resources
}

func (ws *WebServer) startContinuousMiningWithConfig(configMap map[string]interface{}) {
	// Parse configuration with defaults (resource-aware)
	config := MiningConfig{
		InitialIntensity: 1,
		MaxIntensity:     4,
		IntensityStep:    30,
		AutoScale:        true,
		MinMiners:        2,
		MaxMiners:        ws.maxConcurrency, // Use detected CPU capacity
		Duration:         0,                 // infinite
		HighPerformance:  false,
		MaxDifficulty:    6, // Maximum supported difficulty
		CPUIntensive:     false,
	}

	if val, ok := configMap["initialIntensity"].(float64); ok {
		config.InitialIntensity = int(val)
	}
	if val, ok := configMap["maxIntensity"].(float64); ok {
		config.MaxIntensity = int(val)
	}
	if val, ok := configMap["intensityStep"].(float64); ok {
		config.IntensityStep = int(val)
	}
	if val, ok := configMap["autoScale"].(bool); ok {
		config.AutoScale = val
	}
	if val, ok := configMap["minMiners"].(float64); ok {
		config.MinMiners = int(val)
	}
	if val, ok := configMap["maxMiners"].(float64); ok {
		config.MaxMiners = int(val)
	}
	if val, ok := configMap["duration"].(float64); ok {
		config.Duration = int(val)
	}
	if val, ok := configMap["highPerformance"].(bool); ok {
		config.HighPerformance = val
	}
	if val, ok := configMap["maxDifficulty"].(float64); ok {
		config.MaxDifficulty = int(val)
	}
	if val, ok := configMap["cpuIntensive"].(bool); ok {
		config.CPUIntensive = val
	}

	// High-performance overrides
	if config.HighPerformance {
		config.MaxMiners = ws.maxConcurrency * 2 // Push beyond normal limits
		config.MaxDifficulty = 6                 // Maximum supported PoW difficulty
		config.IntensityStep = 10                // Faster scaling
	}

	// CPU-intensive overrides
	if config.CPUIntensive {
		config.MaxMiners = ws.maxConcurrency * 3 // 3x normal capacity
		config.MaxDifficulty = 6                 // Maximum supported PoW difficulty
		runtime.GOMAXPROCS(ws.cpuCores)          // Use all cores
	}

	ws.mu.Lock()
	if ws.miningActive {
		log.Printf("Continuous mining already active, ignoring start request")
		ws.mu.Unlock()
		return
	}
	ws.miningActive = true
	ws.miningIntensity = config.InitialIntensity
	ws.activeMinerCount = 0
	ws.mu.Unlock()

	log.Printf("🚀 Starting CONFIGURED blockchain network simulation...")
	log.Printf("⚙️ Config: Intensity %d→%d, Miners %d→%d, AutoScale: %v",
		config.InitialIntensity, config.MaxIntensity, config.MinMiners, config.MaxMiners, config.AutoScale)

	ws.broadcastLog("info", "🚀 Starting CONFIGURED blockchain network simulation...", "🚀")
	configMsg := fmt.Sprintf("⚙️ Config: Intensity %d→%d, Miners %d→%d, AutoScale: %v",
		config.InitialIntensity, config.MaxIntensity, config.MinMiners, config.MaxMiners, config.AutoScale)
	ws.broadcastLog("info", configMsg, "⚙️")

	if config.Duration > 0 {
		durationMsg := fmt.Sprintf("⏱️ Simulation duration: %d seconds", config.Duration)
		log.Printf(durationMsg)
		ws.broadcastLog("info", durationMsg, "⏱️")
	}

	// Start configured network simulation
	go ws.configuredNetworkController(config)

	ws.broadcast(WebSocketMessage{
		Type:         "mining_status",
		MiningActive: true,
		Stats: &MiningStats{
			CurrentDifficulty: ws.stats.CurrentDifficulty,
		},
	})
}

func (ws *WebServer) configuredNetworkController(config MiningConfig) {
	defer func() {
		ws.mu.Lock()
		ws.miningActive = false
		ws.activeMinerCount = 0
		ws.mu.Unlock()
		ws.broadcastLog("info", "⏹️ Configured network simulation completed", "⏹️")
	}()

	var intensityTicker *time.Ticker
	if config.AutoScale {
		intensityTicker = time.NewTicker(time.Duration(config.IntensityStep) * time.Second)
		defer intensityTicker.Stop()
	}

	var durationTimer *time.Timer
	if config.Duration > 0 {
		durationTimer = time.NewTimer(time.Duration(config.Duration) * time.Second)
		defer durationTimer.Stop()
	}

	// Start miner spawning
	go ws.spawnMinersWithConfig(config)

	for {
		select {
		case <-ws.stopMining:
			ws.broadcastLog("info", "🛑 Stopping configured network simulation...", "🛑")
			// Drain signals
			for len(ws.stopMining) > 0 {
				<-ws.stopMining
			}
			return
		case <-func() <-chan time.Time {
			if durationTimer != nil {
				return durationTimer.C
			}
			return make(chan time.Time) // never triggers if duration is 0
		}():
			ws.broadcastLog("success", "✅ Simulation duration completed", "✅")
			return
		case <-func() <-chan time.Time {
			if intensityTicker != nil {
				return intensityTicker.C
			}
			return make(chan time.Time) // never triggers if autoScale is false
		}():
			ws.mu.Lock()
			if !ws.miningActive {
				ws.mu.Unlock()
				return
			}

			if config.AutoScale && ws.miningIntensity < config.MaxIntensity && rand.Float64() < 0.7 {
				ws.miningIntensity++
				msg := fmt.Sprintf("📈 Auto-scaled network intensity to level %d", ws.miningIntensity)
				log.Printf(msg)
				ws.broadcastLog("info", msg, "📈")

				if ws.miningIntensity >= 3 {
					ws.broadcastLog("warning", "🔒 DDoS PROTECTION ACTIVATED! High network load detected", "🔒")
				}
			}
			ws.mu.Unlock()
		}
	}
}

func (ws *WebServer) spawnMinersWithConfig(config MiningConfig) {
	for {
		ws.mu.RLock()
		if !ws.miningActive {
			ws.mu.RUnlock()
			return
		}
		intensity := ws.miningIntensity
		activeCount := ws.activeMinerCount
		ws.mu.RUnlock()

		// Calculate target miners based on intensity and config
		targetMiners := config.MinMiners
		baseInterval := 3 * time.Second

		switch intensity {
		case 1: // Low intensity
			targetMiners = config.MinMiners + rand.Intn(3)
			baseInterval = time.Duration(3+rand.Intn(4)) * time.Second
		case 2: // Medium intensity
			targetMiners = (config.MinMiners+config.MaxMiners)/2 + rand.Intn(5)
			baseInterval = time.Duration(1+rand.Intn(3)) * time.Second
		case 3: // High intensity (DDoS protection activates)
			targetMiners = config.MaxMiners - rand.Intn(5)
			baseInterval = time.Duration(200+rand.Intn(800)) * time.Millisecond
		case 4: // EXTREME intensity (stress testing) - conservative limits
			// Conservative cap: never exceed 2x CPU cores for stability
			safetyCap := ws.cpuCores * 2
			extremeMax := config.MaxMiners + rand.Intn(config.MaxMiners/6) // Reduced from /4 to /6
			if extremeMax > safetyCap {
				extremeMax = safetyCap
			}
			targetMiners = extremeMax
			baseInterval = time.Duration(200+rand.Intn(500)) * time.Millisecond // More conservative timing
		}

		// Ensure within bounds with conservative global safety cap
		globalSafetyCap := ws.cpuCores * 3 // Reduced from 4x to 3x CPU cores
		if targetMiners > globalSafetyCap {
			targetMiners = globalSafetyCap
			log.Printf("⚠️ Capping miners at %d (safety limit: %dx CPU cores)", globalSafetyCap, 3)
		}
		if targetMiners > config.MaxMiners && ws.miningIntensity < 4 {
			targetMiners = config.MaxMiners
		}
		if targetMiners < config.MinMiners {
			targetMiners = config.MinMiners
		}

		// Spawn miners if below target
		if activeCount < targetMiners {
			go ws.simulateRealisticMinerWithConfig(config)
			ws.mu.Lock()
			ws.activeMinerCount++
			ws.mu.Unlock()
			msg := fmt.Sprintf("⛏️ Spawned miner %d/%d (configured intensity %d)", activeCount+1, targetMiners, intensity)
			log.Printf(msg)
			ws.broadcastLog("info", msg, "⛏️")
		}

		time.Sleep(baseInterval)
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
	ws.miningIntensity = 1
	ws.activeMinerCount = 0
	ws.mu.Unlock()

	log.Printf("🚀 Starting realistic blockchain network simulation...")
	log.Printf("📡 Simulating real-world mining network with dynamic intensity")
	log.Printf("🔒 DDoS protection will activate under high load")
	log.Printf("⚡ Network will scale from 1-20+ concurrent miners")

	ws.broadcastLog("info", "🚀 Starting realistic blockchain network simulation...", "🚀")
	ws.broadcastLog("info", "📡 Simulating real-world mining network with dynamic intensity", "📡")
	ws.broadcastLog("info", "🔒 DDoS protection will activate under high load", "🔒")
	ws.broadcastLog("info", "⚡ Network will scale from 1-20+ concurrent miners", "⚡")

	// Start main network simulation controller
	go ws.networkSimulationController()

	ws.broadcast(WebSocketMessage{
		Type:         "mining_status",
		MiningActive: true,
		Stats: &MiningStats{
			CurrentDifficulty: ws.stats.CurrentDifficulty,
		},
	})
}

func (ws *WebServer) networkSimulationController() {
	defer func() {
		ws.mu.Lock()
		ws.miningActive = false
		ws.activeMinerCount = 0
		ws.mu.Unlock()
		log.Printf("⏹️  Network simulation stopped")
	}()

	intensityTicker := time.NewTicker(30 * time.Second) // Change intensity every 30s
	defer intensityTicker.Stop()

	// Start with low intensity
	go ws.spawnMinersForIntensity()

	for {
		select {
		case <-ws.stopMining:
			log.Printf("🛑 Stopping network simulation...")
			// Drain signals
			for len(ws.stopMining) > 0 {
				<-ws.stopMining
			}
			return
		case <-intensityTicker.C:
			ws.mu.Lock()
			if !ws.miningActive {
				ws.mu.Unlock()
				return
			}

			// Gradually increase intensity to simulate network growth
			if ws.miningIntensity < 3 && rand.Float64() < 0.7 { // 70% chance to increase
				ws.miningIntensity++
				msg := fmt.Sprintf("📈 Network intensity increased to level %d", ws.miningIntensity)
				log.Printf(msg)
				ws.broadcastLog("info", msg, "📈")

				if ws.miningIntensity >= 3 {
					log.Printf("🔒 DDoS PROTECTION ACTIVATED! High network load detected")
					log.Printf("🛡️  Adaptive difficulty adjustment enabled")
					log.Printf("⚡ Connection rate limiting engaged")

					ws.broadcastLog("warning", "🔒 DDoS PROTECTION ACTIVATED! High network load detected", "🔒")
					ws.broadcastLog("warning", "🛡️ Adaptive difficulty adjustment enabled", "🛡️")
					ws.broadcastLog("warning", "⚡ Connection rate limiting engaged", "⚡")
				}
			} else if ws.miningIntensity > 1 && rand.Float64() < 0.3 { // 30% chance to decrease
				oldIntensity := ws.miningIntensity
				ws.miningIntensity--
				msg := fmt.Sprintf("📉 Network intensity decreased to level %d", ws.miningIntensity)
				log.Printf(msg)
				ws.broadcastLog("info", msg, "📉")

				if oldIntensity >= 3 && ws.miningIntensity < 3 {
					log.Printf("🔓 DDoS protection deactivated - network load normalized")
					ws.broadcastLog("success", "🔓 DDoS protection deactivated - network load normalized", "🔓")
				}
			}
			ws.mu.Unlock()
		}
	}
}

func (ws *WebServer) spawnMinersForIntensity() {
	for {
		ws.mu.RLock()
		if !ws.miningActive {
			ws.mu.RUnlock()
			return
		}
		intensity := ws.miningIntensity
		activeCount := ws.activeMinerCount
		ws.mu.RUnlock()

		// Calculate target miners and spawn interval based on intensity
		var targetMiners int
		var baseInterval time.Duration

		switch intensity {
		case 1: // Low intensity
			targetMiners = 2 + rand.Intn(3)                            // 2-4 miners
			baseInterval = time.Duration(3+rand.Intn(4)) * time.Second // 3-6s
		case 2: // Medium intensity
			targetMiners = 5 + rand.Intn(6)                            // 5-10 miners
			baseInterval = time.Duration(1+rand.Intn(3)) * time.Second // 1-3s
		case 3: // High intensity (triggers DDoS protection)
			targetMiners = 15 + rand.Intn(10)                                   // 15-24 miners
			baseInterval = time.Duration(200+rand.Intn(800)) * time.Millisecond // 200ms-1s
		}

		// Spawn new miners if below target
		if activeCount < targetMiners {
			go ws.simulateRealisticMiner()
			ws.mu.Lock()
			ws.activeMinerCount++
			ws.mu.Unlock()
			msg := fmt.Sprintf("⛏️ Spawned miner %d/%d (intensity level %d)", activeCount+1, targetMiners, intensity)
			log.Printf(msg)
			ws.broadcastLog("info", msg, "⛏️")
		}

		// Random disconnections (simulate network churn)
		if activeCount > 1 && rand.Float64() < 0.1 { // 10% chance of miner leaving
			ws.mu.Lock()
			ws.activeMinerCount = max(0, ws.activeMinerCount-1)
			ws.mu.Unlock()
			log.Printf("👋 Miner disconnected, active: %d", ws.activeMinerCount-1)
		}

		time.Sleep(baseInterval)
	}
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

func (ws *WebServer) clearServerState() {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	log.Printf("🗑️  Clearing server state...")

	// Stop mining if active
	if ws.miningActive {
		select {
		case ws.stopMining <- true:
		default:
		}
		ws.miningActive = false
	}

	// Clear all data structures
	ws.blockchain = []Block{}
	ws.challenges = make(map[string]*Challenge)
	ws.connections = make(map[string]*ClientConnection)
	ws.recentLogs = make([]LogMessage, 0, 100)
	ws.activeMinerCount = 0
	ws.miningIntensity = 1

	// Reset stats
	ws.stats = &MiningStats{
		CurrentDifficulty:    2,
		HashRate:             0,
		AverageSolveTime:     0,
		TotalChallenges:      0,
		CompletedChallenges:  0,
		LiveConnections:      0,
		TotalConnections:     0,
		NetworkIntensity:     1,
		DDosProtectionActive: false,
		ActiveMinerCount:     0,
	}

	log.Printf("✅ Server state cleared - all data reset")

	// Broadcast clear_state message to all clients
	ws.broadcast(WebSocketMessage{
		Type: "clear_state",
	})

	// Send fresh init data
	for conn := range ws.clients {
		ws.sendInitialData(conn)
	}
}

func (ws *WebServer) simulateRealisticMinerWithConfig(config MiningConfig) {
	defer func() {
		ws.mu.Lock()
		ws.activeMinerCount = max(0, ws.activeMinerCount-1)
		ws.mu.Unlock()
	}()

	clientID := fmt.Sprintf("miner-%d-%d", time.Now().UnixNano()%10000, rand.Intn(1000))

	connection := &ClientConnection{
		ID:          clientID,
		Address:     fmt.Sprintf("192.168.%d.%d:simulated", rand.Intn(255), rand.Intn(255)),
		ConnectedAt: time.Now().Unix(),
		Status:      "connecting",
	}

	ws.mu.Lock()
	ws.connections[clientID] = connection
	ws.totalConnections++
	ws.mu.Unlock()

	msg := fmt.Sprintf("🔌 Miner %s connecting from %s", clientID[:8], connection.Address)
	log.Printf(msg)
	ws.broadcastLog("info", msg, "🔌")

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Simulate realistic connection time
	connectionDelay := time.Duration(100+rand.Intn(500)) * time.Millisecond
	time.Sleep(connectionDelay)

	// Random chance of connection failure (simulate network issues)
	if rand.Float64() < 0.05 { // 5% chance of connection failure
		connection.Status = "disconnected"
		log.Printf("❌ Miner %s failed to connect (network timeout)", clientID[:8])
		ws.broadcast(WebSocketMessage{
			Type:       "connection",
			Connection: connection,
		})
		return
	}

	connection.Status = "connected"
	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Simulate multiple mining attempts (realistic miner behavior)
	miningAttempts := 1 + rand.Intn(3) // 1-3 attempts per session
	if config.CPUIntensive {
		miningAttempts = 3 + rand.Intn(5) // More attempts for CPU intensive mode
	}

	for attempt := 0; attempt < miningAttempts; attempt++ {
		// Check if mining is still active
		ws.mu.RLock()
		if !ws.miningActive {
			ws.mu.RUnlock()
			break
		}
		ws.mu.RUnlock()

		// Generate challenge with adaptive difficulty based on configuration
		ws.mu.Lock()
		currentDifficulty := ws.stats.CurrentDifficulty
		// Adaptive difficulty based on network intensity and config
		oldDifficulty := currentDifficulty
		maxAllowedDifficulty := min(config.MaxDifficulty, 6) // Never exceed 6 (PoW constraint)

		if ws.miningIntensity >= 3 && currentDifficulty < maxAllowedDifficulty {
			// Increase difficulty during high load
			currentDifficulty = min(maxAllowedDifficulty, currentDifficulty+1)
			ws.stats.CurrentDifficulty = currentDifficulty
			log.Printf("🔧 Difficulty increased: %d → %d (intensity %d, configured mode)", oldDifficulty, currentDifficulty, ws.miningIntensity)
			ws.recordDifficultyAdjustment("increase", oldDifficulty, currentDifficulty)
		} else if ws.miningIntensity <= 1 && currentDifficulty > 1 && ws.activeMinerCount < 5 {
			// Decrease difficulty during low load
			currentDifficulty = max(1, currentDifficulty-1)
			ws.stats.CurrentDifficulty = currentDifficulty
			log.Printf("🔧 Difficulty decreased: %d → %d (intensity %d, low load)", oldDifficulty, currentDifficulty, ws.miningIntensity)
			ws.recordDifficultyAdjustment("decrease", oldDifficulty, currentDifficulty)
		}
		if config.HighPerformance || config.CPUIntensive {
			newDifficulty := min(maxAllowedDifficulty, currentDifficulty+2)
			if newDifficulty != currentDifficulty {
				ws.stats.CurrentDifficulty = newDifficulty
				currentDifficulty = newDifficulty
				log.Printf("🔧 Difficulty boosted: %d → %d (high performance mode)", oldDifficulty, currentDifficulty)
				ws.recordDifficultyAdjustment("boost", oldDifficulty, currentDifficulty)
			}
		}
		ws.mu.Unlock()

		seed, challengeDifficulty, err := ws.generateChallenge(currentDifficulty)
		if err != nil {
			log.Printf("❌ Failed to generate challenge for %s: %v", clientID[:8], err)
			continue
		}

		webChallenge := &Challenge{
			ID:         fmt.Sprintf("challenge-%d-%d", time.Now().UnixNano(), rand.Intn(1000)),
			Seed:       seed,
			Difficulty: challengeDifficulty,
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
			Type:      "challenge",
			Challenge: webChallenge,
		})

		ws.broadcast(WebSocketMessage{
			Type:       "connection",
			Connection: connection,
		})

		log.Printf("⛏️  Miner %s solving difficulty %d challenge (attempt %d/%d)", clientID[:8], challengeDifficulty, attempt+1, miningAttempts)

		// Mine the challenge
		start := time.Now()
		var solution string
		if ws.algorithm == "sha256" {
			challenge := &pow.Challenge{Seed: seed, Difficulty: challengeDifficulty}
			solution, err = pow.SolveChallenge(challenge)
		} else {
			challenge, _ := pow.GenerateArgon2Challenge(challengeDifficulty)
			challenge.Seed = seed
			solution, err = pow.SolveArgon2Challenge(challenge)
		}
		elapsed := time.Since(start)

		if err != nil {
			msg := fmt.Sprintf("❌ Miner %s failed to solve challenge", clientID[:8])
			log.Printf("❌ Miner %s failed to solve challenge: %v", clientID[:8], err)
			ws.broadcastLog("error", msg, "❌")
			webChallenge.Status = "failed"
		} else {
			msg := fmt.Sprintf("✅ Miner %s solved challenge in %v (difficulty %d)", clientID[:8], elapsed, challengeDifficulty)
			log.Printf(msg)
			ws.broadcastLog("success", msg, "✅")
			webChallenge.Status = "completed"
			connection.ChallengesCompleted++

			// Create block
			ws.createAndBroadcastBlock(webChallenge, solution, seed, challengeDifficulty, elapsed)
		}

		ws.broadcast(WebSocketMessage{
			Type:      "challenge_update",
			Challenge: webChallenge,
		})

		// Random break between attempts (simulate real miner behavior)
		if attempt < miningAttempts-1 {
			time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
		}
	}

	// Miner session end
	connection.Status = "disconnected"
	log.Printf("👋 Miner %s disconnecting after %d challenges", clientID[:8], connection.ChallengesCompleted)

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Keep connection in history for a while, then clean up
	time.Sleep(10 * time.Second)
	ws.mu.Lock()
	delete(ws.connections, clientID)
	ws.mu.Unlock()
}

func (ws *WebServer) simulateRealisticMiner() {
	defer func() {
		ws.mu.Lock()
		ws.activeMinerCount = max(0, ws.activeMinerCount-1)
		ws.mu.Unlock()
	}()

	clientID := fmt.Sprintf("miner-%d-%d", time.Now().UnixNano()%10000, rand.Intn(1000))

	connection := &ClientConnection{
		ID:          clientID,
		Address:     fmt.Sprintf("192.168.%d.%d:simulated", rand.Intn(255), rand.Intn(255)),
		ConnectedAt: time.Now().Unix(),
		Status:      "connecting",
	}

	ws.mu.Lock()
	ws.connections[clientID] = connection
	ws.totalConnections++
	ws.mu.Unlock()

	msg := fmt.Sprintf("🔌 Miner %s connecting from %s", clientID[:8], connection.Address)
	log.Printf(msg)
	ws.broadcastLog("info", msg, "🔌")

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Simulate realistic connection time
	connectionDelay := time.Duration(100+rand.Intn(500)) * time.Millisecond
	time.Sleep(connectionDelay)

	// Random chance of connection failure (simulate network issues)
	if rand.Float64() < 0.05 { // 5% chance of connection failure
		connection.Status = "disconnected"
		log.Printf("❌ Miner %s failed to connect (network timeout)", clientID[:8])
		ws.broadcast(WebSocketMessage{
			Type:       "connection",
			Connection: connection,
		})
		return
	}

	connection.Status = "connected"
	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Simulate multiple mining attempts (realistic miner behavior)
	miningAttempts := 1 + rand.Intn(3) // 1-3 attempts per session

	for attempt := 0; attempt < miningAttempts; attempt++ {
		// Check if mining is still active
		ws.mu.RLock()
		if !ws.miningActive {
			ws.mu.RUnlock()
			break
		}
		ws.mu.RUnlock()

		// Generate challenge with adaptive difficulty based on network intensity
		ws.mu.Lock()
		currentDifficulty := ws.stats.CurrentDifficulty
		// Adaptive difficulty based on network intensity (max difficulty 6)
		maxAllowedDifficulty := 6 // PoW constraint
		oldDifficulty := currentDifficulty

		if ws.miningIntensity >= 3 && currentDifficulty < maxAllowedDifficulty {
			// Increase difficulty during high load
			currentDifficulty = min(maxAllowedDifficulty, currentDifficulty+1)
			ws.stats.CurrentDifficulty = currentDifficulty
			log.Printf("🔧 Difficulty increased: %d → %d (intensity %d, realistic mode)", oldDifficulty, currentDifficulty, ws.miningIntensity)
			ws.recordDifficultyAdjustment("increase", oldDifficulty, currentDifficulty)
		} else if ws.miningIntensity <= 1 && currentDifficulty > 1 && ws.activeMinerCount < 5 {
			// Decrease difficulty during low load
			currentDifficulty = max(1, currentDifficulty-1)
			ws.stats.CurrentDifficulty = currentDifficulty
			log.Printf("🔧 Difficulty decreased: %d → %d (intensity %d, low load)", oldDifficulty, currentDifficulty, ws.miningIntensity)
			ws.recordDifficultyAdjustment("decrease", oldDifficulty, currentDifficulty)
		}
		ws.mu.Unlock()

		seed, challengeDifficulty, err := ws.generateChallenge(currentDifficulty)
		if err != nil {
			log.Printf("❌ Failed to generate challenge for %s: %v", clientID[:8], err)
			continue
		}

		webChallenge := &Challenge{
			ID:         fmt.Sprintf("challenge-%d-%d", time.Now().UnixNano(), rand.Intn(1000)),
			Seed:       seed,
			Difficulty: challengeDifficulty,
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
			Type:      "challenge",
			Challenge: webChallenge,
		})

		ws.broadcast(WebSocketMessage{
			Type:       "connection",
			Connection: connection,
		})

		log.Printf("⛏️  Miner %s solving difficulty %d challenge (attempt %d/%d)", clientID[:8], challengeDifficulty, attempt+1, miningAttempts)

		// Mine the challenge
		start := time.Now()
		var solution string
		if ws.algorithm == "sha256" {
			challenge := &pow.Challenge{Seed: seed, Difficulty: challengeDifficulty}
			solution, err = pow.SolveChallenge(challenge)
		} else {
			challenge, _ := pow.GenerateArgon2Challenge(challengeDifficulty)
			challenge.Seed = seed
			solution, err = pow.SolveArgon2Challenge(challenge)
		}
		elapsed := time.Since(start)

		if err != nil {
			msg := fmt.Sprintf("❌ Miner %s failed to solve challenge", clientID[:8])
			log.Printf("❌ Miner %s failed to solve challenge: %v", clientID[:8], err)
			ws.broadcastLog("error", msg, "❌")
			webChallenge.Status = "failed"
		} else {
			msg := fmt.Sprintf("✅ Miner %s solved challenge in %v (difficulty %d)", clientID[:8], elapsed, challengeDifficulty)
			log.Printf(msg)
			ws.broadcastLog("success", msg, "✅")
			webChallenge.Status = "completed"
			connection.ChallengesCompleted++

			// Create block
			ws.createAndBroadcastBlock(webChallenge, solution, seed, challengeDifficulty, elapsed)
		}

		ws.broadcast(WebSocketMessage{
			Type:      "challenge_update",
			Challenge: webChallenge,
		})

		// Random break between attempts (simulate real miner behavior)
		if attempt < miningAttempts-1 {
			time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
		}
	}

	// Miner session end
	connection.Status = "disconnected"
	log.Printf("👋 Miner %s disconnecting after %d challenges", clientID[:8], connection.ChallengesCompleted)

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Keep connection in history for a while, then clean up
	time.Sleep(10 * time.Second)
	ws.mu.Lock()
	delete(ws.connections, clientID)
	ws.mu.Unlock()
}

func (ws *WebServer) createAndBroadcastBlock(webChallenge *Challenge, solution string, seed string, difficulty int, elapsed time.Duration) {
	data := seed + solution
	hash := sha256.Sum256([]byte(data))
	hashHex := hex.EncodeToString(hash[:])

	sol := &Solution{
		ChallengeID: webChallenge.ID,
		Nonce:       solution,
		Hash:        hashHex,
		Attempts:    int(elapsed.Nanoseconds() / 1000), // Rough attempt estimate
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
	msg := fmt.Sprintf("🎉 New block #%d mined by %s! Hash: %s", block.Index, webChallenge.ClientID[:8], block.Hash[:16]+"...")
	log.Printf(msg)
	ws.broadcastLog("success", msg, "🎉")

	// Update stats
	newAvgSolveTime := (ws.stats.AverageSolveTime*float64(ws.stats.CompletedChallenges-1) + float64(elapsed.Milliseconds())) / float64(ws.stats.CompletedChallenges)
	if math.IsInf(newAvgSolveTime, 0) || math.IsNaN(newAvgSolveTime) {
		ws.stats.AverageSolveTime = float64(elapsed.Milliseconds())
	} else {
		ws.stats.AverageSolveTime = newAvgSolveTime
	}

	elapsedSeconds := elapsed.Seconds()
	if elapsedSeconds > 0.001 {
		ws.stats.HashRate = 1000.0 / elapsedSeconds
	} else {
		ws.stats.HashRate = 1000000.0
	}

	if math.IsInf(ws.stats.HashRate, 0) || math.IsNaN(ws.stats.HashRate) {
		ws.stats.HashRate = 0
	}

	// Update connection counts and network status
	ws.stats.LiveConnections = len(ws.connections)
	ws.stats.TotalConnections = ws.totalConnections
	ws.stats.NetworkIntensity = ws.miningIntensity
	ws.stats.DDosProtectionActive = ws.miningIntensity >= 3 // DDoS protection kicks in at high intensity
	ws.stats.ActiveMinerCount = ws.activeMinerCount
	statsCopy := *ws.stats
	ws.mu.Unlock()

	// Broadcast the new block
	ws.broadcast(WebSocketMessage{
		Type:  "block",
		Block: &block,
	})

	// Broadcast updated stats
	ws.broadcast(WebSocketMessage{
		Type:  "stats",
		Stats: &statsCopy,
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
	ws.totalConnections++
	ws.mu.Unlock()

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	seed, challengeDifficulty, err := ws.generateChallenge(ws.stats.CurrentDifficulty)
	if err != nil {
		log.Printf("Failed to generate challenge: %v", err)
		return
	}

	webChallenge := &Challenge{
		ID:         fmt.Sprintf("challenge-%d", time.Now().UnixNano()),
		Seed:       seed,
		Difficulty: challengeDifficulty,
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
		Type:      "challenge",
		Challenge: webChallenge,
	})

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	log.Printf("⛏️  Client %s solving difficulty %d challenge...", clientID, challengeDifficulty)
	start := time.Now()
	var solution string
	if ws.algorithm == "sha256" {
		challenge := &pow.Challenge{Seed: seed, Difficulty: challengeDifficulty}
		solution, err = pow.SolveChallenge(challenge)
	} else {
		challenge, _ := pow.GenerateArgon2Challenge(challengeDifficulty)
		challenge.Seed = seed
		solution, err = pow.SolveArgon2Challenge(challenge)
	}
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("❌ Client %s failed to solve challenge: %v", clientID, err)
		webChallenge.Status = "failed"
		connection.Status = "disconnected"
	} else {
		log.Printf("✅ Client %s solved challenge in %v (difficulty %d)", clientID, elapsed, challengeDifficulty)
		webChallenge.Status = "completed"
		connection.Status = "connected"
		connection.ChallengesCompleted++

		data := seed + solution
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
		Type:      "challenge_update",
		Challenge: webChallenge,
	})

	ws.broadcast(WebSocketMessage{
		Type:       "connection",
		Connection: connection,
	})

	// Update live connection counts in stats
	ws.mu.Lock()
	ws.stats.LiveConnections = len(ws.connections)
	ws.stats.TotalConnections = ws.totalConnections
	statsCopy := *ws.stats
	ws.mu.Unlock()

	ws.broadcast(WebSocketMessage{
		Type:  "stats",
		Stats: &statsCopy,
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

func (ws *WebServer) HandleClearState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ws.clearServerState()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
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

// recordDifficultyAdjustment records a difficulty adjustment to the database
func (ws *WebServer) recordDifficultyAdjustment(direction string, oldDifficulty, newDifficulty int) {
	if ws.queries != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		labels := map[string]interface{}{
			"direction":      direction,
			"old_difficulty": oldDifficulty,
			"new_difficulty": newDifficulty,
			"algorithm":      ws.algorithm,
		}

		labelsJSON, _ := json.Marshal(labels)

		params := db.RecordMetricParams{
			MetricName:     "difficulty_adjustment",
			MetricValue:    1.0,
			Labels:         labelsJSON,
			ServerInstance: pgtype.Text{String: "webserver", Valid: true},
		}

		err := ws.queries.RecordMetric(ctx, ws.db, params)
		if err != nil {
			log.Printf("Failed to record difficulty adjustment: %v", err)
		}
	}
}

// getDifficultyAdjustmentCount retrieves the count of difficulty adjustments in the last hour
func (ws *WebServer) getDifficultyAdjustmentCount() int64 {
	if ws.queries == nil {
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := ws.queries.CountDifficultyAdjustments(ctx, ws.db)
	if err != nil {
		log.Printf("Failed to get difficulty adjustment count: %v", err)
		return 0
	}

	return count
}

func (ws *WebServer) startMetricsBroadcast() {
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		for range ticker.C {
			// Fetch difficulty adjustment count from database
			difficultyAdjustments := ws.getDifficultyAdjustmentCount()

			// Fetch and broadcast Prometheus metrics
			metrics := &MetricsData{
				Timestamp:             time.Now().UnixMilli(),
				ConnectionsTotal:      float64(len(ws.connections)),
				CurrentDifficulty:     float64(ws.stats.CurrentDifficulty),
				PuzzlesSolvedTotal:    float64(ws.stats.CompletedChallenges),
				PuzzlesFailedTotal:    0,
				AverageSolveTime:      ws.stats.AverageSolveTime,
				ConnectionRate:        0,
				DifficultyAdjustments: float64(difficultyAdjustments),
				ActiveConnections:     float64(len(ws.connections)),
			}

			ws.broadcast(WebSocketMessage{
				Type:    "metrics",
				Metrics: metrics,
			})
		}
	}()
}

// generateChallenge creates a challenge using the configured algorithm
func (ws *WebServer) generateChallenge(difficulty int) (seed string, challengeDifficulty int, err error) {
	if ws.algorithm == "sha256" {
		challenge, err := pow.GenerateChallenge(difficulty)
		if err != nil {
			return "", 0, err
		}
		return challenge.Seed, challenge.Difficulty, nil
	} else {
		challenge, err := pow.GenerateArgon2Challenge(difficulty)
		if err != nil {
			return "", 0, err
		}
		return challenge.Seed, challenge.Difficulty, nil
	}
}

// verifyChallenge verifies a challenge solution using the configured algorithm
func (ws *WebServer) verifyChallenge(seed string, nonce string, difficulty int) bool {
	if ws.algorithm == "sha256" {
		return pow.VerifyPoW(seed, nonce, difficulty)
	} else {
		// For verification we need to recreate the challenge with same parameters
		challenge, err := pow.GenerateArgon2Challenge(difficulty)
		if err != nil {
			return false
		}
		challenge.Seed = seed
		return pow.VerifyArgon2PoW(challenge, nonce)
	}
}

func (ws *WebServer) Start() {
	log.Println("WebServer started")
	ws.startMetricsBroadcast()
	ws.startAutoSave()
}

func (ws *WebServer) Stop() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()

	for conn := range ws.clients {
		conn.Close()
	}
}
