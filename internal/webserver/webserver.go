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
	blockchain       []Block
	challenges       map[string]*Challenge
	connections      map[string]*ClientConnection
	recentLogs       []LogMessage // Store recent logs for stateless operation
	stats            *MiningStats
	mu               sync.RWMutex
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

type APIResponse struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
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
		blockchain:    make([]Block, 0),
		challenges:    make(map[string]*Challenge),
		connections:   make(map[string]*ClientConnection),
		recentLogs:    make([]LogMessage, 0, 100),
		stats: &MiningStats{
			CurrentDifficulty: 2,
			AverageSolveTime:  0,
			HashRate:          0,
		},
		dataDir:          "/tmp/wisdom-data",
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
	}

	// Create data directory and load persistent data
	os.MkdirAll(ws.dataDir, 0755)
	ws.loadData()

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

func (ws *WebServer) storeLog(level, message, icon string) {
	logMsg := &LogMessage{
		Timestamp: time.Now().UnixMilli(),
		Level:     level,
		Message:   message,
		Icon:      icon,
	}

	// Store log in recent logs buffer
	ws.mu.Lock()
	ws.recentLogs = append(ws.recentLogs, *logMsg)
	if len(ws.recentLogs) > 100 {
		ws.recentLogs = ws.recentLogs[1:]
	}
	ws.mu.Unlock()

	// Store log in database
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
}

// REST API Handlers

func (ws *WebServer) HandleRecentSolves(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ws.mu.RLock()
	// Get last 10 blocks
	recentBlocks := ws.blockchain
	if len(ws.blockchain) > 10 {
		recentBlocks = ws.blockchain[len(ws.blockchain)-10:]
	}
	blocksCopy := make([]Block, len(recentBlocks))
	copy(blocksCopy, recentBlocks)
	ws.mu.RUnlock()
	
	response := APIResponse{
		Status: "success",
		Data:   blocksCopy,
	}
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ws.mu.RLock()
	health := map[string]interface{}{
		"status":           "healthy",
		"miningActive":     ws.miningActive,
		"totalBlocks":      len(ws.blockchain),
		"activeChallenges": len(ws.challenges),
		"liveConnections":  len(ws.connections),
		"algorithm":        ws.algorithm,
		"difficulty":       ws.stats.CurrentDifficulty,
	}
	ws.mu.RUnlock()
	
	response := APIResponse{
		Status: "success",
		Data:   health,
	}
	json.NewEncoder(w).Encode(response)
}

func (ws *WebServer) HandleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ctx := context.Background()
	
	// Get logs from database
	dbLogs, err := ws.queries.GetRecentLogs(ctx, ws.db, 100)
	if err != nil {
		log.Printf("Error fetching logs from database: %v", err)
		http.Error(w, "Failed to fetch logs", http.StatusInternalServerError)
		return
	}
	
	// Convert database logs to API format
	logs := make([]LogMessage, len(dbLogs))
	for i, dbLog := range dbLogs {
		logs[i] = LogMessage{
			Timestamp: dbLog.Timestamp.Time.Unix() * 1000,
			Level:     dbLog.Level,
			Message:   dbLog.Message,
			Icon:      dbLog.Icon.String,
		}
	}
	
	response := APIResponse{
		Status: "success",
		Data:   logs,
	}
	json.NewEncoder(w).Encode(response)
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

	ws.storeLog("info", "🚀 Starting CONFIGURED blockchain network simulation...", "🚀")
	configMsg := fmt.Sprintf("⚙️ Config: Intensity %d→%d, Miners %d→%d, AutoScale: %v",
		config.InitialIntensity, config.MaxIntensity, config.MinMiners, config.MaxMiners, config.AutoScale)
	ws.storeLog("info", configMsg, "⚙️")

	if config.Duration > 0 {
		durationMsg := fmt.Sprintf("⏱️ Simulation duration: %d seconds", config.Duration)
		log.Printf(durationMsg)
		ws.storeLog("info", durationMsg, "⏱️")
	}

	// Start configured network simulation
	go ws.configuredNetworkController(config)
}

func (ws *WebServer) configuredNetworkController(config MiningConfig) {
	defer func() {
		ws.mu.Lock()
		ws.miningActive = false
		ws.activeMinerCount = 0
		ws.mu.Unlock()
		ws.storeLog("info", "⏹️ Configured network simulation completed", "⏹️")
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
			ws.storeLog("info", "🛑 Stopping configured network simulation...", "🛑")
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
			ws.storeLog("success", "✅ Simulation duration completed", "✅")
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
				ws.storeLog("info", msg, "📈")

				if ws.miningIntensity >= 3 {
					ws.storeLog("warning", "🔒 DDoS PROTECTION ACTIVATED! High network load detected", "🔒")
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
			ws.storeLog("info", msg, "⛏️")
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

	ws.storeLog("info", "🚀 Starting realistic blockchain network simulation...", "🚀")
	ws.storeLog("info", "📡 Simulating real-world mining network with dynamic intensity", "📡")
	ws.storeLog("info", "🔒 DDoS protection will activate under high load", "🔒")
	ws.storeLog("info", "⚡ Network will scale from 1-20+ concurrent miners", "⚡")

	// Start main network simulation controller
	go ws.networkSimulationController()
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
				ws.storeLog("info", msg, "📈")

				if ws.miningIntensity >= 3 {
					log.Printf("🔒 DDoS PROTECTION ACTIVATED! High network load detected")
					log.Printf("🛡️  Adaptive difficulty adjustment enabled")
					log.Printf("⚡ Connection rate limiting engaged")

					ws.storeLog("warning", "🔒 DDoS PROTECTION ACTIVATED! High network load detected", "🔒")
					ws.storeLog("warning", "🛡️ Adaptive difficulty adjustment enabled", "🛡️")
					ws.storeLog("warning", "⚡ Connection rate limiting engaged", "⚡")
				}
			} else if ws.miningIntensity > 1 && rand.Float64() < 0.3 { // 30% chance to decrease
				oldIntensity := ws.miningIntensity
				ws.miningIntensity--
				msg := fmt.Sprintf("📉 Network intensity decreased to level %d", ws.miningIntensity)
				log.Printf(msg)
				ws.storeLog("info", msg, "📉")

				if oldIntensity >= 3 && ws.miningIntensity < 3 {
					log.Printf("🔓 DDoS protection deactivated - network load normalized")
					ws.storeLog("success", "🔓 DDoS protection deactivated - network load normalized", "🔓")
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
			ws.storeLog("info", msg, "⛏️")
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
	ws.storeLog("info", msg, "🔌")


	// Simulate realistic connection time
	connectionDelay := time.Duration(100+rand.Intn(500)) * time.Millisecond
	time.Sleep(connectionDelay)

	// Random chance of connection failure (simulate network issues)
	if rand.Float64() < 0.05 { // 5% chance of connection failure
		connection.Status = "disconnected"
		log.Printf("❌ Miner %s failed to connect (network timeout)", clientID[:8])
		return
	}

	connection.Status = "connected"

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
			ws.storeLog("error", msg, "❌")
			webChallenge.Status = "failed"
		} else {
			msg := fmt.Sprintf("✅ Miner %s solved challenge in %v (difficulty %d)", clientID[:8], elapsed, challengeDifficulty)
			log.Printf(msg)
			ws.storeLog("success", msg, "✅")
			webChallenge.Status = "completed"
			connection.ChallengesCompleted++

			// Create block
			ws.createAndBroadcastBlock(webChallenge, solution, seed, challengeDifficulty, elapsed)
		}


		// Random break between attempts (simulate real miner behavior)
		if attempt < miningAttempts-1 {
			time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
		}
	}

	// Miner session end
	connection.Status = "disconnected"
	log.Printf("👋 Miner %s disconnecting after %d challenges", clientID[:8], connection.ChallengesCompleted)


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
	ws.storeLog("info", msg, "🔌")


	// Simulate realistic connection time
	connectionDelay := time.Duration(100+rand.Intn(500)) * time.Millisecond
	time.Sleep(connectionDelay)

	// Random chance of connection failure (simulate network issues)
	if rand.Float64() < 0.05 { // 5% chance of connection failure
		connection.Status = "disconnected"
		log.Printf("❌ Miner %s failed to connect (network timeout)", clientID[:8])
		return
	}

	connection.Status = "connected"

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
			ws.storeLog("error", msg, "❌")
			webChallenge.Status = "failed"
		} else {
			msg := fmt.Sprintf("✅ Miner %s solved challenge in %v (difficulty %d)", clientID[:8], elapsed, challengeDifficulty)
			log.Printf(msg)
			ws.storeLog("success", msg, "✅")
			webChallenge.Status = "completed"
			connection.ChallengesCompleted++

			// Create block
			ws.createAndBroadcastBlock(webChallenge, solution, seed, challengeDifficulty, elapsed)
		}


		// Random break between attempts (simulate real miner behavior)
		if attempt < miningAttempts-1 {
			time.Sleep(time.Duration(500+rand.Intn(2000)) * time.Millisecond)
		}
	}

	// Miner session end
	connection.Status = "disconnected"
	log.Printf("👋 Miner %s disconnecting after %d challenges", clientID[:8], connection.ChallengesCompleted)


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
	ws.storeLog("success", msg, "🎉")

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
	ws.mu.Unlock()
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

	}



	// Update live connection counts in stats
	ws.mu.Lock()
	ws.stats.LiveConnections = len(ws.connections)
	ws.stats.TotalConnections = ws.totalConnections
	ws.mu.Unlock()

}

func (ws *WebServer) HandleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ws.mu.RLock()
	statsData := map[string]interface{}{
		"stats": ws.stats,
		"miningActive": ws.miningActive,
		"connections": map[string]interface{}{
			"total": ws.totalConnections,
			"active": len(ws.connections),
		},
		"blockchain": map[string]interface{}{
			"blocks": len(ws.blockchain),
			"lastBlock": func() interface{} {
				if len(ws.blockchain) > 0 {
					return ws.blockchain[len(ws.blockchain)-1]
				}
				return nil
			}(),
		},
		"challenges": map[string]interface{}{
			"active": len(ws.challenges),
		},
		"system": map[string]interface{}{
			"algorithm": ws.algorithm,
			"intensity": ws.miningIntensity,
			"activeMiners": ws.activeMinerCount,
		},
	}
	ws.mu.RUnlock()
	
	response := APIResponse{
		Status: "success",
		Data:   statsData,
	}
	json.NewEncoder(w).Encode(response)
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
			// Metrics would be sent here if WebSocket was enabled
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
	log.Println("WebServer stopped")
}
