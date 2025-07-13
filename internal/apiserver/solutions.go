package apiserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"world-of-wisdom/api/db"
)

type CreateSolutionRequest struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Nonce       string `json:"nonce" binding:"required"`
	Hash        string `json:"hash" binding:"required"`
	Attempts    int32  `json:"attempts" binding:"required,min=1"`
	SolveTimeMs int64  `json:"solveTimeMs" binding:"required,min=0"`
	Verified    bool   `json:"verified"`
}

type VerifySolutionRequest struct {
	Verified bool `json:"verified" binding:"required"`
}

func (s *Server) getSolutions(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	limit := getLimit(c, 50)
	
	solutions, err := s.queries.GetRecentSolutions(ctx, s.db, int32(limit))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get solutions", err)
		return
	}

	s.handleSuccess(c, solutions)
}

func (s *Server) createSolution(c *gin.Context) {
	var req CreateSolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	challengeUUID, err := parseUUID(req.ChallengeID)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid challenge ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	params := db.CreateSolutionParams{
		ChallengeID: challengeUUID,
		Nonce:       req.Nonce,
		Hash:        req.Hash,
		Attempts: pgtype.Int4{
			Int32: req.Attempts,
			Valid: true,
		},
		SolveTimeMs: req.SolveTimeMs,
		Verified:    req.Verified,
	}

	solution, err := s.queries.CreateSolution(ctx, s.db, params)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to create solution", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":      solution,
		"timestamp": solution.CreatedAt,
	})
}

func (s *Server) getSolution(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid solution ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	solution, err := s.queries.GetSolution(ctx, s.db, uuid)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "Solution not found", err)
		return
	}

	s.handleSuccess(c, solution)
}

func (s *Server) getSolutionsByChallenge(c *gin.Context) {
	challengeIDStr := c.Param("challengeId")
	challengeUUID, err := parseUUID(challengeIDStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid challenge ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	solutions, err := s.queries.GetSolutionsByChallenge(ctx, s.db, challengeUUID)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get solutions for challenge", err)
		return
	}

	s.handleSuccess(c, solutions)
}

func (s *Server) verifySolution(c *gin.Context) {
	idStr := c.Param("id")
	uuid, err := parseUUID(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid solution ID", err)
		return
	}

	var req VerifySolutionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	solution, err := s.queries.VerifySolution(ctx, s.db, db.VerifySolutionParams{
		ID:       uuid,
		Verified: req.Verified,
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to verify solution", err)
		return
	}

	s.handleSuccess(c, solution)
}

func (s *Server) getSolutionStats(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	stats, err := s.queries.GetSolutionStats(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get solution stats", err)
		return
	}

	s.handleSuccess(c, stats)
}