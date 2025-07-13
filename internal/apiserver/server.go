package apiserver

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"world-of-wisdom/api/db"
)

type Server struct {
	db      *pgxpool.Pool
	queries *db.Queries
}

func New(dbpool *pgxpool.Pool, queries *db.Queries) *Server {
	return &Server{
		db:      dbpool,
		queries: queries,
	}
}

func (s *Server) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1")
	{
		// Challenges endpoints
		challenges := api.Group("/challenges")
		{
			challenges.GET("", s.getChallenges)
			challenges.POST("", s.createChallenge)
			challenges.GET("/:id", s.getChallenge)
			challenges.PATCH("/:id/status", s.updateChallengeStatus)
			challenges.GET("/client/:clientId", s.getChallengeByClient)
			challenges.GET("/difficulty/:difficulty", s.getChallengesByDifficulty)
			challenges.GET("/algorithm/:algorithm", s.getChallengesByAlgorithm)
		}

		// Solutions endpoints
		solutions := api.Group("/solutions")
		{
			solutions.GET("", s.getSolutions)
			solutions.POST("", s.createSolution)
			solutions.GET("/:id", s.getSolution)
			solutions.GET("/challenge/:challengeId", s.getSolutionsByChallenge)
			solutions.PATCH("/:id/verify", s.verifySolution)
			solutions.GET("/stats", s.getSolutionStats)
		}

		// Connections endpoints
		connections := api.Group("/connections")
		{
			connections.GET("", s.getConnections)
			connections.POST("", s.createConnection)
			connections.GET("/:id", s.getConnection)
			connections.GET("/client/:clientId", s.getConnectionByClient)
			connections.PATCH("/:id/status", s.updateConnectionStatus)
			connections.PATCH("/:id/stats", s.updateConnectionStats)
			connections.GET("/active", s.getActiveConnections)
			connections.GET("/stats", s.getConnectionStats)
		}

		// Blocks endpoints
		blocks := api.Group("/blocks")
		{
			blocks.GET("", s.getBlocks)
			blocks.POST("", s.createBlock)
			blocks.GET("/:id", s.getBlock)
			blocks.GET("/index/:index", s.getBlockByIndex)
			blocks.GET("/latest", s.getLatestBlock)
			blocks.GET("/blockchain", s.getBlockchain)
		}

		// Metrics endpoints
		metrics := api.Group("/metrics")
		{
			metrics.POST("", s.recordMetric)
			metrics.GET("/name/:name", s.getMetricsByName)
			metrics.GET("/recent", s.getRecentMetrics)
			metrics.GET("/history/:name", s.getMetricHistory)
			metrics.GET("/system", s.getSystemMetrics)
		}

		// Documentation endpoint
		api.GET("/docs", s.getAPIDocumentation)
	}
}

// Helper function to parse UUID from string
func parseUUID(uuidStr string) (pgtype.UUID, error) {
	var uuid pgtype.UUID
	err := uuid.Scan(uuidStr)
	return uuid, err
}

// Helper function to handle errors
func (s *Server) handleError(c *gin.Context, statusCode int, message string, err error) {
	response := gin.H{
		"error":     message,
		"timestamp": time.Now().UTC(),
	}
	if gin.Mode() == gin.DebugMode && err != nil {
		response["details"] = err.Error()
	}
	c.JSON(statusCode, response)
}

// Helper function for successful responses
func (s *Server) handleSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
}

// Helper function to get limit parameter
func getLimit(c *gin.Context, defaultLimit int) int {
	limitStr := c.DefaultQuery("limit", strconv.Itoa(defaultLimit))
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		return defaultLimit
	}
	return limit
}

// Helper function to create context with timeout
func (s *Server) contextWithTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}