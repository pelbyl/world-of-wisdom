package apiserver

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"world-of-wisdom/api/db"
)

// @Summary Get recent logs
// @Description Get recent activity logs with optional limit
// @Tags logs
// @Accept json
// @Produce json
// @Param limit query int false "Number of logs to return (default: 100, max: 1000)"
// @Success 200 {object} map[string]interface{} "List of logs"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs [get]
func (s *Server) getLogs(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	limit := getLimit(c, 100)

	logs, err := s.queries.GetRecentLogs(ctx, s.db, int32(limit))
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to fetch logs", err)
		return
	}

	s.handleSuccess(c, logs)
}

// @Summary Get paginated logs
// @Description Get logs with cursor-based pagination
// @Tags logs
// @Accept json
// @Produce json
// @Param before query string false "Timestamp to get logs before (RFC3339 format)"
// @Param limit query int false "Number of logs to return (default: 50, max: 1000)"
// @Success 200 {object} map[string]interface{} "List of logs"
// @Failure 400 {object} map[string]interface{} "Invalid parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs/paginated [get]
func (s *Server) getLogsPaginated(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	beforeStr := c.Query("before")
	limit := getLimit(c, 50)

	var beforeTime pgtype.Timestamptz
	if beforeStr != "" {
		parsedTime, err := time.Parse(time.RFC3339, beforeStr)
		if err != nil {
			s.handleError(c, http.StatusBadRequest, "Invalid before timestamp format", err)
			return
		}
		beforeTime = pgtype.Timestamptz{Time: parsedTime, Valid: true}
	}

	logs, err := s.queries.GetLogsPaginated(ctx, s.db, db.GetLogsPaginatedParams{
		Column1: beforeTime,
		Limit:   int32(limit),
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to fetch logs", err)
		return
	}

	s.handleSuccess(c, logs)
}

// @Summary Get logs by level
// @Description Get logs filtered by level
// @Tags logs
// @Accept json
// @Produce json
// @Param level path string true "Log level (info, success, warning, error)"
// @Param limit query int false "Number of logs to return (default: 100, max: 1000)"
// @Success 200 {object} map[string]interface{} "List of logs"
// @Failure 400 {object} map[string]interface{} "Invalid level"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs/level/{level} [get]
func (s *Server) getLogsByLevel(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	level := c.Param("level")
	if level != "info" && level != "success" && level != "warning" && level != "error" {
		s.handleError(c, http.StatusBadRequest, "Invalid log level", nil)
		return
	}

	limit := getLimit(c, 100)

	logs, err := s.queries.GetLogsByLevel(ctx, s.db, db.GetLogsByLevelParams{
		Level: level,
		Limit: int32(limit),
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to fetch logs", err)
		return
	}

	s.handleSuccess(c, logs)
}

// @Summary Create a new log entry
// @Description Create a new activity log entry
// @Tags logs
// @Accept json
// @Produce json
// @Param log body object true "Log entry"
// @Success 201 {object} map[string]interface{} "Created log"
// @Failure 400 {object} map[string]interface{} "Invalid request body"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs [post]
func (s *Server) createLog(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	var req struct {
		Timestamp *time.Time             `json:"timestamp"`
		Level     string                 `json:"level" binding:"required,oneof=info success warning error"`
		Message   string                 `json:"message" binding:"required"`
		Icon      string                 `json:"icon,omitempty"`
		Metadata  map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var timestamp pgtype.Timestamptz
	if req.Timestamp != nil {
		timestamp = pgtype.Timestamptz{Time: *req.Timestamp, Valid: true}
	} else {
		timestamp = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	}

	var metadataBytes []byte
	if req.Metadata != nil {
		var err error
		metadataBytes, err = json.Marshal(req.Metadata)
		if err != nil {
			s.handleError(c, http.StatusBadRequest, "Invalid metadata format", err)
			return
		}
	}

	log, err := s.queries.CreateLog(ctx, s.db, db.CreateLogParams{
		Column1:  timestamp,
		Level:    req.Level,
		Message:  req.Message,
		Icon:     pgtype.Text{String: req.Icon, Valid: req.Icon != ""},
		Metadata: metadataBytes,
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to create log", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":      log,
		"timestamp": time.Now().UTC(),
	})
}

// @Summary Get log statistics
// @Description Get counts of logs grouped by level
// @Tags logs
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Log statistics"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs/stats [get]
func (s *Server) getLogStats(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	stats, err := s.queries.CountLogsByLevel(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to fetch log statistics", err)
		return
	}

	s.handleSuccess(c, stats)
}

// @Summary Get logs in time range
// @Description Get logs within a specific time range
// @Tags logs
// @Accept json
// @Produce json
// @Param from query string true "Start time (RFC3339 format)"
// @Param to query string true "End time (RFC3339 format)"
// @Success 200 {object} map[string]interface{} "List of logs"
// @Failure 400 {object} map[string]interface{} "Invalid time format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /api/v1/logs/range [get]
func (s *Server) getLogsInTimeRange(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		s.handleError(c, http.StatusBadRequest, "Both 'from' and 'to' parameters are required", nil)
		return
	}

	fromTime, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid 'from' timestamp format", err)
		return
	}

	toTime, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid 'to' timestamp format", err)
		return
	}

	logs, err := s.queries.GetLogsInTimeRange(ctx, s.db, db.GetLogsInTimeRangeParams{
		Timestamp:   pgtype.Timestamptz{Time: fromTime, Valid: true},
		Timestamp_2: pgtype.Timestamptz{Time: toTime, Valid: true},
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to fetch logs", err)
		return
	}

	s.handleSuccess(c, logs)
}
