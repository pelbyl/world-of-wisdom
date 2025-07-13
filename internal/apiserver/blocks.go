package apiserver

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"world-of-wisdom/api/db"
)

type CreateBlockRequest struct {
	BlockIndex   int32   `json:"blockIndex" binding:"required,min=0"`
	ChallengeID  *string `json:"challengeId,omitempty"`
	SolutionID   *string `json:"solutionId,omitempty"`
	Quote        *string `json:"quote,omitempty"`
	PreviousHash *string `json:"previousHash,omitempty"`
	BlockHash    string  `json:"blockHash" binding:"required"`
}

func (s *Server) getBlocks(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	limit := getLimit(c, 50)

	blocks, err := s.queries.GetRecentBlocks(ctx, s.db, int32(limit))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get blocks", err)
		return
	}

	s.handleSuccess(c, blocks)
}

func (s *Server) createBlock(c *gin.Context) {
	var req CreateBlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	params := db.CreateBlockParams{
		BlockIndex: req.BlockIndex,
		BlockHash:  req.BlockHash,
	}

	// Parse optional UUIDs
	if req.ChallengeID != nil {
		challengeUUID, err := parseUUID(*req.ChallengeID)
		if err != nil {
			s.handleError(c, http.StatusBadRequest, "Invalid challenge ID", err)
			return
		}
		params.ChallengeID = pgtype.UUID{
			Bytes: challengeUUID.Bytes,
			Valid: true,
		}
	}

	if req.SolutionID != nil {
		solutionUUID, err := parseUUID(*req.SolutionID)
		if err != nil {
			s.handleError(c, http.StatusBadRequest, "Invalid solution ID", err)
			return
		}
		params.SolutionID = pgtype.UUID{
			Bytes: solutionUUID.Bytes,
			Valid: true,
		}
	}

	// Set optional fields
	if req.Quote != nil {
		params.Quote = pgtype.Text{String: *req.Quote, Valid: true}
	}
	if req.PreviousHash != nil {
		params.PreviousHash = pgtype.Text{String: *req.PreviousHash, Valid: true}
	}

	block, err := s.queries.CreateBlock(ctx, s.db, params)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to create block", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":      block,
		"timestamp": block.CreatedAt,
	})
}

func (s *Server) getBlock(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid block ID", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	block, err := s.queries.GetBlock(ctx, s.db, int32(id))
	if err != nil {
		s.handleError(c, http.StatusNotFound, "Block not found", err)
		return
	}

	s.handleSuccess(c, block)
}

func (s *Server) getBlockByIndex(c *gin.Context) {
	indexStr := c.Param("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid block index", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	block, err := s.queries.GetBlockByIndex(ctx, s.db, int32(index))
	if err != nil {
		s.handleError(c, http.StatusNotFound, "Block not found", err)
		return
	}

	s.handleSuccess(c, block)
}

func (s *Server) getLatestBlock(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	block, err := s.queries.GetLatestBlock(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusNotFound, "No blocks found", err)
		return
	}

	s.handleSuccess(c, block)
}

func (s *Server) getBlockchain(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	blockchain, err := s.queries.GetBlockchain(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get blockchain", err)
		return
	}

	s.handleSuccess(c, blockchain)
}
