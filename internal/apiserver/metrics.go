package apiserver

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"world-of-wisdom/api/db"
)

type RecordMetricRequest struct {
	MetricName     string                 `json:"metricName" binding:"required"`
	MetricValue    float64                `json:"metricValue" binding:"required"`
	Labels         map[string]interface{} `json:"labels,omitempty"`
	ServerInstance string                 `json:"serverInstance,omitempty"`
}

func (s *Server) recordMetric(c *gin.Context) {
	var req RecordMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.handleError(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	// Convert labels map to JSON
	var labelsJSON []byte
	var err error
	if req.Labels != nil {
		labelsJSON, err = json.Marshal(req.Labels)
		if err != nil {
			s.handleError(c, http.StatusBadRequest, "Invalid labels format", err)
			return
		}
	} else {
		labelsJSON = []byte("{}")
	}

	serverInstance := req.ServerInstance
	if serverInstance == "" {
		serverInstance = "default"
	}

	err = s.queries.RecordMetric(ctx, s.db, db.RecordMetricParams{
		MetricName:     req.MetricName,
		MetricValue:    req.MetricValue,
		Labels:         labelsJSON,
		ServerInstance: pgtype.Text{String: serverInstance, Valid: true},
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to record metric", err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":    "recorded",
		"timestamp": gin.H{},
	})
}

func (s *Server) getMetricsByName(c *gin.Context) {
	metricName := c.Param("name")
	limit := int32(getLimit(c, 100))

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	metrics, err := s.queries.GetMetricsByName(ctx, s.db, db.GetMetricsByNameParams{
		MetricName: metricName,
		Limit:      limit,
	})
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get metrics", err)
		return
	}

	s.handleSuccess(c, metrics)
}

func (s *Server) getRecentMetrics(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	metrics, err := s.queries.GetRecentMetrics(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get recent metrics", err)
		return
	}

	s.handleSuccess(c, metrics)
}

func (s *Server) getMetricHistory(c *gin.Context) {
	metricName := c.Param("name")

	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	history, err := s.queries.GetMetricHistory(ctx, s.db, metricName)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get metric history", err)
		return
	}

	s.handleSuccess(c, history)
}

func (s *Server) getSystemMetrics(c *gin.Context) {
	ctx, cancel := s.contextWithTimeout()
	defer cancel()

	metrics, err := s.queries.GetSystemMetrics(ctx, s.db)
	if err != nil {
		s.handleError(c, http.StatusInternalServerError, "Failed to get system metrics", err)
		return
	}

	s.handleSuccess(c, metrics)
}