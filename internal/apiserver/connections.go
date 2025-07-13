package apiserver

import (
	"net/http"
	"net/netip"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"world-of-wisdom/api/db"
)

type CreateConnectionRequest struct {
	ClientID   string `json:"clientId" binding:"required"`
	RemoteAddr string `json:"remoteAddr" binding:"required"`
	Algorithm  string `json:"algorithm" binding:"required,oneof=sha256 argon2"`
}

type UpdateConnectionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=connected solving disconnected failed"`
}

type UpdateConnectionStatsRequest struct {
	ChallengesAttempted int32 `json:"challengesAttempted" binding:"min=0"`
	ChallengesCompleted int32 `json:"challengesCompleted" binding:"min=0"`
	TotalSolveTimeMs    int64 `json:"totalSolveTimeMs" binding:"min=0"`
}

func (s *Server) getConnections(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	limit := getLimit(c, 50)
	
	connections, err := s.queries.GetRecentConnections(ctx, s.db, int32(limit))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get connections", err)
		return
	}

	s.handleSuccess(c, connections)
}

func (s *Server) createConnection(c *gin.Context) {
	var req CreateConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert string algorithm to enum
	var algorithm db.PowAlgorithm
	switch req.Algorithm {
	case "sha256":
		algorithm = db.PowAlgorithmSha256
	case "argon2":
		algorithm = db.PowAlgorithmArgon2
	default:
		s.handleError(c, http.StatusBadRequest, "Invalid algorithm", nil)
		return
	}

	// Parse remote address
	remoteAddr, err := netip.ParseAddr(req.RemoteAddr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid remote address", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	params := db.CreateConnectionParams{
		ClientID:   req.ClientID,
		RemoteAddr: remoteAddr,
		Status:     db.ConnectionStatusConnected,
		Algorithm:  algorithm,
	}

	connection, err := s.queries.CreateConnection(ctx, s.db, params)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to create connection", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":      connection,
		"timestamp": connection.ConnectedAt,
	})
}

func (s *Server) getConnection(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	connection, err := s.queries.GetConnection(ctx, s.db, uuid)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "Connection not found", err)
		return
	}

	s.handleSuccess(c, connection)
}

func (s *Server) getConnectionByClient(c *gin.Context) {
	clientID := c.Param("clientId")

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	connection, err := s.queries.GetConnectionByClientID(ctx, s.db, clientID)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "No active connection found for client", err)
		return
	}

	s.handleSuccess(c, connection)
}

func (s *Server) updateConnectionStatus(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	var req UpdateConnectionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert string status to enum
	var status db.ConnectionStatus
	switch req.Status {
	case "connected":
		status = db.ConnectionStatusConnected
	case "solving":
		status = db.ConnectionStatusSolving
	case "disconnected":
		status = db.ConnectionStatusDisconnected
	case "failed":
		status = db.ConnectionStatusFailed
	default:
		s.handleError(c, http.StatusBadRequest, "Invalid status", nil)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	connection, err := s.queries.UpdateConnectionStatus(ctx, s.db, db.UpdateConnectionStatusParams{
		ID:     uuid,
		Status: status,
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to update connection status", err)
		return
	}

	s.handleSuccess(c, connection)
}

func (s *Server) updateConnectionStats(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid connection ID", err)
		return
	}

	var req UpdateConnectionStatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	connection, err := s.queries.UpdateConnectionStats(ctx, s.db, db.UpdateConnectionStatsParams{
		ID: uuid,
		ChallengesAttempted: pgtype.Int4{
			Int32: req.ChallengesAttempted,
			Valid: true,
		},
		ChallengesCompleted: pgtype.Int4{
			Int32: req.ChallengesCompleted,
			Valid: true,
		},
		TotalSolveTimeMs: pgtype.Int8{
			Int64: req.TotalSolveTimeMs,
			Valid: true,
		},
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to update connection stats", err)
		return
	}

	s.handleSuccess(c, connection)
}

func (s *Server) getActiveConnections(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	connections, err := s.queries.GetActiveConnections(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get active connections", err)
		return
	}

	s.handleSuccess(c, connections)
}

func (s *Server) getConnectionStats(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	stats, err := s.queries.GetConnectionStats(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get connection stats", err)
		return
	}

	s.handleSuccess(c, stats)
}