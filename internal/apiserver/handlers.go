package apiserver

import (
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"world-of-wisdom/internal/database/repository"
	"world-of-wisdom/internal/blockchain"
	"world-of-wisdom/internal/behavior"
	"github.com/jackc/pgx/v5/pgxpool"
	generated "world-of-wisdom/internal/database/generated"
)

type Server struct {
	db              *pgxpool.Pool
	repo            repository.Repository
	blockchain      *blockchain.Blockchain
	behaviorTracker *behavior.Tracker
}

func NewServer(database *pgxpool.Pool, bc *blockchain.Blockchain) *Server {
	return &Server{
		db:              database,
		repo:            repository.New(database),
		blockchain:      bc,
		behaviorTracker: behavior.NewTracker(database),
	}
}

func (s *Server) GetHealth(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get basic stats for health check
	challengeStats, _ := s.repo.Challenges().GetStats(ctx)
	connectionStats, _ := s.repo.Connections().GetStats(ctx)
	
	// Determine health status
	var status HealthDataStatus = Healthy
	if connectionStats.ActiveConnections == 0 {
		status = Degraded
	}
	
	// Calculate active challenges (pending + solving)
	activeChallenges := int(challengeStats.PendingCount + challengeStats.SolvingCount)
	liveConnections := int(connectionStats.ActiveConnections)
	totalBlocks := 0 // TODO: Get from blockchain when available
	miningActive := true
	
	// Get current difficulty from most recent challenge
	difficulty := 2 // default
	recentChallenges, err := s.repo.Challenges().GetRecent(ctx, 1)
	if err == nil && len(recentChallenges) > 0 {
		difficulty = int(recentChallenges[0].Difficulty)
	}
	algorithm := HealthDataAlgorithmArgon2
	
	healthData := HealthData{
		Status:           &status,
		MiningActive:     &miningActive,
		TotalBlocks:      &totalBlocks,
		ActiveChallenges: &activeChallenges,
		LiveConnections:  &liveConnections,
		Algorithm:        &algorithm,
		Difficulty:       &difficulty,
	}
	
	response := HealthResponse{
		Data:   &healthData,
		Status: HealthResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetStats(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get all required stats
	challengeStats, _ := s.repo.Challenges().GetStats(ctx)
	connectionStats, _ := s.repo.Connections().GetStats(ctx)
	blockchainStats, _ := s.repo.GetBlockchainStats(ctx)
	
	// Convert types properly
	totalChallenges := int(challengeStats.TotalCount)
	completedChallenges := int(challengeStats.CompletedCount)
	averageSolveTime := float32(challengeStats.AvgSolveTimeMs)
	
	// Get current difficulty from most recent challenge
	currentDifficulty := 2 // default
	recentChallenges, err := s.repo.Challenges().GetRecent(ctx, 1)
	if err == nil && len(recentChallenges) > 0 {
		currentDifficulty = int(recentChallenges[0].Difficulty)
	}
	hashRate := float32(0.0)
	
	totalConnections := int(connectionStats.TotalConnections)
	activeConnections := int(connectionStats.ActiveConnections)
	
	totalBlocks := int(blockchainStats.TotalBlocks)
	activeChallengesCount := int(challengeStats.PendingCount + challengeStats.SolvingCount)
	
	miningActive := true
	algorithm := "argon2"
	intensity := 2
	activeMiners := int(connectionStats.ActiveConnections)
	
	// Build stats response
	statsData := StatsData{
		Stats: &MiningStats{
			TotalChallenges:     &totalChallenges,
			CompletedChallenges: &completedChallenges,
			AverageSolveTime:    &averageSolveTime,
			CurrentDifficulty:   &currentDifficulty,
			HashRate:            &hashRate,
		},
		MiningActive: &miningActive,
		Connections: &ConnectionStats{
			Total:  &totalConnections,
			Active: &activeConnections,
		},
		Blockchain: &BlockchainStats{
			Blocks:    &totalBlocks,
			LastBlock: nil, // TODO: Implement when blockchain is available
		},
		Challenges: &ChallengeStats{
			Active: &activeChallengesCount,
		},
		System: &SystemStats{
			Algorithm:     &algorithm,
			Intensity:     &intensity,
			ActiveMiners:  &activeMiners,
		},
	}
	
	response := StatsResponse{
		Data:   &statsData,
		Status: Success,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetChallenges(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse query parameters
	limitStr := c.QueryParam("limit")
	statusStr := c.QueryParam("status")
	algorithmStr := c.QueryParam("algorithm")
	
	limit := int32(50) // default
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			limit = int32(parsed)
		}
	}
	
	// Use GetRecent as a simple workaround for the nullable enum issue
	allChallenges, err := s.repo.Challenges().GetRecent(ctx, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get challenges: "+err.Error())
	}
	
	// Convert to the expected format for the response logic
	challenges := make([]repository.GetChallengesFilteredRow, 0, len(allChallenges))
	for _, ch := range allChallenges {
		// Apply optional filters
		if statusStr != "" && string(ch.Status) != statusStr {
			continue
		}
		if algorithmStr != "" && string(ch.Algorithm) != algorithmStr {
			continue
		}
		
		var solveTimeMs int64
		if ch.Status == "completed" && ch.SolvedAt.Valid && ch.CreatedAt.Valid {
			solveTimeMs = ch.SolvedAt.Time.Sub(ch.CreatedAt.Time).Milliseconds()
		}
		
		challenges = append(challenges, repository.GetChallengesFilteredRow{
			ID:          ch.ID,
			Seed:        ch.Seed,
			Difficulty:  ch.Difficulty,
			Algorithm:   ch.Algorithm,
			ClientID:    ch.ClientID,
			Status:      ch.Status,
			CreatedAt:   ch.CreatedAt,
			SolvedAt:    ch.SolvedAt,
			ExpiresAt:   ch.ExpiresAt,
			SolveTimeMs: solveTimeMs,
		})
	}
	
	// Convert to API format
	challengeDetails := make([]ChallengeDetail, len(challenges))
	for i, ch := range challenges {
		id := ch.ID.String()
		difficulty := int(ch.Difficulty)
		algorithm := ChallengeDetailAlgorithm(ch.Algorithm)
		status := ChallengeDetailStatus(ch.Status)
		
		var createdAt, expiresAt *time.Time
		if ch.CreatedAt.Valid {
			createdAt = &ch.CreatedAt.Time
		}
		if ch.ExpiresAt.Valid {
			expiresAt = &ch.ExpiresAt.Time
		}
		
		challengeDetails[i] = ChallengeDetail{
			Id:          &id,
			Seed:        &ch.Seed,
			Difficulty:  &difficulty,
			Algorithm:   &algorithm,
			ClientId:    &ch.ClientID,
			Status:      &status,
			CreatedAt:   createdAt,
			ExpiresAt:   expiresAt,
			SolvedAt:    nil,
			SolveTimeMs: nil,
		}
		
		if ch.SolvedAt.Valid {
			challengeDetails[i].SolvedAt = &ch.SolvedAt.Time
			if ch.SolveTimeMs > 0 {
				solveTime := int(ch.SolveTimeMs)
				challengeDetails[i].SolveTimeMs = &solveTime
			}
		}
	}
	
	total := len(challenges)
	
	response := ChallengesResponse{
		Data: &struct {
			Challenges *[]ChallengeDetail `json:"challenges,omitempty"`
			Total      *int               `json:"total,omitempty"`
		}{
			Challenges: &challengeDetails,
			Total:      &total,
		},
		Status: ChallengesResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetConnections(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse status filter
	statusStr := c.QueryParam("status")
	
	// Get connections (active by default)
	var connections []generated.Connection
	var err error
	
	if statusStr != "" {
		status := generated.ConnectionStatus(statusStr)
		connections, err = s.repo.Connections().GetFiltered(ctx, status)
	} else {
		connections, err = s.repo.Connections().GetActive(ctx)
	}
	
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get connections")
	}
	
	// Convert to API format
	connectionDetails := make([]ConnectionDetail, len(connections))
	for i, conn := range connections {
		id := conn.ID.String()
		remoteAddr := conn.RemoteAddr.String()
		status := ConnectionDetailStatus(conn.Status)
		algorithm := ConnectionDetailAlgorithm(conn.Algorithm)
		
		var connectedAt, disconnectedAt *time.Time
		if conn.ConnectedAt.Valid {
			connectedAt = &conn.ConnectedAt.Time
		}
		if conn.DisconnectedAt.Valid {
			disconnectedAt = &conn.DisconnectedAt.Time
		}
		
		challengesAttempted := int(conn.ChallengesAttempted.Int32)
		challengesCompleted := int(conn.ChallengesCompleted.Int32) 
		totalSolveTimeMs := int(conn.TotalSolveTimeMs.Int64)
		
		connectionDetails[i] = ConnectionDetail{
			Id:                  &id,
			ClientId:            &conn.ClientID,
			RemoteAddr:          &remoteAddr,
			Status:              &status,
			Algorithm:           &algorithm,
			ConnectedAt:         connectedAt,
			DisconnectedAt:      disconnectedAt,
			ChallengesAttempted: &challengesAttempted,
			ChallengesCompleted: &challengesCompleted,
			TotalSolveTimeMs:    &totalSolveTimeMs,
		}
	}
	
	// Get stats for totals
	stats, _ := s.repo.Connections().GetStats(ctx)
	totalConnections := int(stats.TotalConnections)
	activeConnections := int(stats.ActiveConnections)
	
	response := ConnectionsResponse{
		Data: &struct {
			Active      *int                `json:"active,omitempty"`
			Connections *[]ConnectionDetail `json:"connections,omitempty"`
			Total       *int                `json:"total,omitempty"`
		}{
			Connections: &connectionDetails,
			Total:       &totalConnections,
			Active:      &activeConnections,
		},
		Status: ConnectionsResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetMetrics(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get system metrics
	metrics, err := s.repo.Metrics().GetSystem(ctx)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get metrics")
	}
	
	// Convert to API format
	metricData := make([]MetricData, len(metrics))
	for i, m := range metrics {
		var timestamp *time.Time
		if m.Time.Valid {
			timestamp = &m.Time.Time
		}
		
		value := float32(m.MetricValue)
		
		metricData[i] = MetricData{
			Time:       timestamp,
			MetricName: &m.MetricName,
			Value:      &value,
			AvgValue:   &value,  // Simplified
			MaxValue:   &value,  // Simplified
			MinValue:   &value,  // Simplified
			Labels:     nil,
		}
	}
	
	response := MetricsResponse{
		Data: &struct {
			Metrics *[]MetricData `json:"metrics,omitempty"`
		}{
			Metrics: &metricData,
		},
		Status: MetricsResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetRecentSolves(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Get recent solutions
	solutions, err := s.repo.Solutions().GetRecent(ctx, 10)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get recent solves")
	}
	
	// Convert to blockchain blocks format
	blocks := make([]Block, len(solutions))
	for i, sol := range solutions {
		index := i
		var timestamp *int64
		if sol.CreatedAt.Valid {
			ts := sol.CreatedAt.Time.Unix()
			timestamp = &ts
		}
		
		quote := "Wisdom through proof of work"
		previousHash := "0000000000000000000000000000000000000000000000000000000000000000"
		var hash string
		if sol.Hash.Valid {
			hash = sol.Hash.String
		}
		
		blocks[i] = Block{
			Index:        &index,
			Timestamp:    timestamp,
			Challenge:    nil, // TODO: Load challenge details
			Solution:     nil, // TODO: Load solution details
			Quote:        &quote,
			PreviousHash: &previousHash,
			Hash:         &hash,
		}
	}
	
	response := RecentSolvesResponse{
		Data:   &blocks,
		Status: RecentSolvesResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetLogs(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse limit parameter
	limitStr := c.QueryParam("limit")
	limit := int32(100) // default
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			limit = int32(parsed)
		}
	}
	
	logs, err := s.repo.Logs().GetRecent(ctx, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get logs")
	}
	
	// Convert to API format
	logMessages := make([]LogMessage, len(logs))
	for i, log := range logs {
		var timestamp *int64
		if log.Timestamp.Valid {
			ts := log.Timestamp.Time.Unix()
			timestamp = &ts
		}
		
		level := LogMessageLevel(log.Level)
		icon := "📝"
		
		logMessages[i] = LogMessage{
			Timestamp: timestamp,
			Level:     &level,
			Message:   &log.Message,
			Icon:      &icon,
		}
	}
	
	response := LogsResponse{
		Data:   &logMessages,
		Status: LogsResponseStatusSuccess,
	}
	
	return c.JSON(http.StatusOK, response)
}

func (s *Server) GetClientBehaviors(c echo.Context) error {
	ctx := c.Request().Context()
	
	// Parse limit parameter
	limitStr := c.QueryParam("limit")
	limit := 100 // default
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 32); err == nil {
			limit = int(parsed)
		}
	}
	
	// Get active clients
	activeClients, err := s.behaviorTracker.GetActiveClients(ctx, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get active clients")
	}
	
	// Get aggressive clients
	aggressiveClients, err := s.behaviorTracker.GetAggressiveClients(ctx, 20)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to get aggressive clients")
	}
	
	// Convert to response format
	type ClientBehaviorInfo struct {
		IP                    string  `json:"ip"`
		Difficulty            int     `json:"difficulty"`
		ConnectionCount       int     `json:"connectionCount"`
		FailureRate           float64 `json:"failureRate"`
		AvgSolveTime          int64   `json:"avgSolveTime"`
		ReconnectRate         float64 `json:"reconnectRate"`
		Reputation            float64 `json:"reputation"`
		Suspicious            float64 `json:"suspicious"`
		LastConnection        string  `json:"lastConnection"`
		IsAggressive          bool    `json:"isAggressive"`
		SuccessfulChallenges  int     `json:"successfulChallenges"`
		FailedChallenges      int     `json:"failedChallenges"`
		TotalChallenges       int     `json:"totalChallenges"`
	}
	
	clients := make([]ClientBehaviorInfo, len(activeClients))
	aggressiveIPs := make(map[string]bool)
	
	// Mark aggressive IPs
	for _, aggressive := range aggressiveClients {
		aggressiveIPs[aggressive.IpAddress.String()] = true
	}
	
	// Convert active clients
	for i, client := range activeClients {
		ipStr := client.IpAddress.String()
		clients[i] = ClientBehaviorInfo{
			IP:                   ipStr,
			Difficulty:           int(client.Difficulty.Int32),
			ConnectionCount:      int(client.ConnectionCount.Int32),
			FailureRate:          client.FailureRate.Float64,
			AvgSolveTime:         client.AvgSolveTimeMs.Int64,
			ReconnectRate:        client.ReconnectRate.Float64,
			Reputation:           client.ReputationScore.Float64,
			Suspicious:           client.SuspiciousActivityScore.Float64,
			LastConnection:       client.LastConnection.Time.Format(time.RFC3339),
			IsAggressive:         aggressiveIPs[ipStr],
			SuccessfulChallenges: int(client.SuccessfulChallenges.Int32),
			FailedChallenges:     int(client.FailedChallenges.Int32),
			TotalChallenges:      int(client.TotalChallenges.Int32),
		}
	}
	
	// Add aggressive clients that are not in active list
	for _, aggressive := range aggressiveClients {
		ipStr := aggressive.IpAddress.String()
		found := false
		for _, active := range clients {
			if active.IP == ipStr {
				found = true
				break
			}
		}
		if !found {
			clients = append(clients, ClientBehaviorInfo{
				IP:                   ipStr,
				Difficulty:           int(aggressive.Difficulty.Int32),
				ConnectionCount:      int(aggressive.ConnectionCount.Int32),
				FailureRate:          aggressive.FailureRate.Float64,
				AvgSolveTime:         aggressive.AvgSolveTimeMs.Int64,
				ReconnectRate:        aggressive.ReconnectRate.Float64,
				Reputation:           aggressive.ReputationScore.Float64,
				Suspicious:           aggressive.SuspiciousActivityScore.Float64,
				LastConnection:       aggressive.LastConnection.Time.Format(time.RFC3339),
				IsAggressive:         true,
				SuccessfulChallenges: int(aggressive.SuccessfulChallenges.Int32),
				FailedChallenges:     int(aggressive.FailedChallenges.Int32),
				TotalChallenges:      int(aggressive.TotalChallenges.Int32),
			})
		}
	}
	
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"clients": clients,
			"total":   len(clients),
		},
		"status": "success",
	}
	
	return c.JSON(http.StatusOK, response)
}