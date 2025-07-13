package apiserver

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"world-of-wisdom/api/db"
)

type CreateChallengeRequest struct {
	Seed           string `json:"seed" binding:"required"`
	Difficulty     int32  `json:"difficulty" binding:"required,min=1,max=6"`
	Algorithm      string `json:"algorithm" binding:"required,oneof=sha256 argon2"`
	ClientID       string `json:"clientId" binding:"required"`
	Argon2Time     *int32 `json:"argon2Time,omitempty"`
	Argon2Memory   *int32 `json:"argon2Memory,omitempty"`
	Argon2Threads  *int16 `json:"argon2Threads,omitempty"`
	Argon2KeyLen   *int32 `json:"argon2KeyLen,omitempty"`
}

type UpdateChallengeStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=pending solving completed failed expired"`
}

func (s *Server) getChallenges(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	limit := getLimit(c, 50)
	
	challenges, err := s.queries.GetRecentChallenges(ctx, s.db, int32(limit))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get challenges", err)
		return
	}

	s.handleSuccess(c, challenges)
}

func (s *Server) createChallenge(c *gin.Context) {
	var req CreateChallengeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

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

	params := db.CreateChallengeParams{
		Seed:          req.Seed,
		Difficulty:    req.Difficulty,
		Algorithm:     algorithm,
		ClientID:      req.ClientID,
		Status:        db.ChallengeStatusPending,
		Argon2Time:    pgtype.Int4{},
		Argon2Memory:  pgtype.Int4{},
		Argon2Threads: pgtype.Int2{},
		Argon2Keylen:  pgtype.Int4{},
	}

	// Set Argon2 parameters if provided
	if req.Argon2Time != nil {
		params.Argon2Time = pgtype.Int4{Int32: *req.Argon2Time, Valid: true}
	}
	if req.Argon2Memory != nil {
		params.Argon2Memory = pgtype.Int4{Int32: *req.Argon2Memory, Valid: true}
	}
	if req.Argon2Threads != nil {
		params.Argon2Threads = pgtype.Int2{Int16: *req.Argon2Threads, Valid: true}
	}
	if req.Argon2KeyLen != nil {
		params.Argon2Keylen = pgtype.Int4{Int32: *req.Argon2KeyLen, Valid: true}
	}

	challenge, err := s.queries.CreateChallenge(ctx, s.db, params)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to create challenge", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":      challenge,
		"timestamp": challenge.CreatedAt,
	})
}

func (s *Server) getChallenge(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid challenge ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	challenge, err := s.queries.GetChallenge(ctx, s.db, uuid)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "Challenge not found", err)
		return
	}

	s.handleSuccess(c, challenge)
}

func (s *Server) updateChallengeStatus(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid challenge ID", err)
		return
	}

	var req UpdateChallengeStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Convert string status to enum
	var status db.ChallengeStatus
	switch req.Status {
	case "pending":
		status = db.ChallengeStatusPending
	case "solving":
		status = db.ChallengeStatusSolving
	case "completed":
		status = db.ChallengeStatusCompleted
	case "failed":
		status = db.ChallengeStatusFailed
	case "expired":
		status = db.ChallengeStatusExpired
	default:
		s.handleError(c, http.StatusBadRequest, "Invalid status", nil)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	challenge, err := s.queries.UpdateChallengeStatus(ctx, s.db, db.UpdateChallengeStatusParams{
		ID:     uuid,
		Status: status,
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to update challenge status", err)
		return
	}

	s.handleSuccess(c, challenge)
}

func (s *Server) getChallengeByClient(c *gin.Context) {
	clientID := c.Param("clientId")

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	challenge, err := s.queries.GetChallengeByClientID(ctx, s.db, clientID)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "No pending challenge found for client", err)
		return
	}

	s.handleSuccess(c, challenge)
}

func (s *Server) getChallengesByDifficulty(c *gin.Context) {
	difficultyStr := c.Param("difficulty")
	difficulty, err := strconv.Atoi(difficultyStr)
	if err != nil || difficulty < 1 || difficulty > 6 {
		s.handleError(c, http.StatusBadRequest, "Invalid difficulty (must be 1-6)", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	challenges, err := s.queries.GetChallengesByDifficulty(ctx, s.db, int32(difficulty))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get challenges by difficulty", err)
		return
	}

	s.handleSuccess(c, challenges)
}

func (s *Server) getChallengesByAlgorithm(c *gin.Context) {
	algorithmStr := c.Param("algorithm")
	
	var algorithm db.PowAlgorithm
	switch algorithmStr {
	case "sha256":
		algorithm = db.PowAlgorithmSha256
	case "argon2":
		algorithm = db.PowAlgorithmArgon2
	default:
		s.handleError(c, http.StatusBadRequest, "Invalid algorithm (must be sha256 or argon2)", nil)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	challenges, err := s.queries.GetChallengesByAlgorithm(ctx, s.db, algorithm)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get challenges by algorithm", err)
		return
	}

	s.handleSuccess(c, challenges)
}