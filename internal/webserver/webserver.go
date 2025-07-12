package webserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
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

type WebSocketMessage struct {
	Type       string      `json:"type"`
	Block      *Block      `json:"block,omitempty"`
	Challenge  *Challenge  `json:"challenge,omitempty"`
	Connection *ClientConnection `json:"connection,omitempty"`
	Stats      *MiningStats `json:"stats,omitempty"`
	Blocks     []Block     `json:"blocks,omitempty"`
	Connections []ClientConnection `json:"connections,omitempty"`
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
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		quoteProvider: wisdom.NewQuoteProvider(),
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

		if msgType, ok := msg["type"].(string); ok && msgType == "simulate_client" {
			go ws.simulateClient()
		}
	}
}

func (ws *WebServer) sendInitialData(conn *websocket.Conn) {
	ws.mu.RLock()
	connections := make([]ClientConnection, 0, len(ws.connections))
	for _, c := range ws.connections {
		connections = append(connections, *c)
	}
	ws.mu.RUnlock()

	msg := WebSocketMessage{
		Type:        "init",
		Blocks:      ws.blockchain,
		Connections: connections,
		Stats:       ws.stats,
	}

	conn.WriteJSON(msg)
}

func (ws *WebServer) broadcast(msg WebSocketMessage) {
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

	start := time.Now()
	solution, err := pow.SolveChallenge(challenge)
	elapsed := time.Since(start)

	if err != nil {
		webChallenge.Status = "failed"
		connection.Status = "disconnected"
	} else {
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
		ws.stats.AverageSolveTime = (ws.stats.AverageSolveTime*float64(ws.stats.CompletedChallenges-1) + float64(elapsed.Milliseconds())) / float64(ws.stats.CompletedChallenges)
		ws.stats.HashRate = 1000.0 / (float64(elapsed.Milliseconds()) / 1000.0)
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

func (ws *WebServer) Start() {
	log.Println("WebServer started")
}

func (ws *WebServer) Stop() {
	ws.clientsMux.Lock()
	defer ws.clientsMux.Unlock()
	
	for conn := range ws.clients {
		conn.Close()
	}
}