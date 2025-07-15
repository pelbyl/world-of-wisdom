package apiserver

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// SetupRoutes configures the HTTP routes for the API server
func (s *Server) SetupRoutes() *echo.Echo {
	e := echo.New()
	
	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	
	// Configure CORS to allow requests from the web frontend
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"}, // Allow all origins for now
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"*"}, // Allow all headers
		AllowCredentials: false, // Set to false when using wildcard origin
	}))
	
	// Health check
	e.GET("/health", s.GetHealth)
	
	// API v1 endpoints
	e.GET("/api/v1/stats", s.GetStats)
	e.GET("/api/v1/challenges", s.GetChallenges)
	e.GET("/api/v1/connections", s.GetConnections)
	e.GET("/api/v1/metrics", s.GetMetrics)
	e.GET("/api/v1/recent-solves", s.GetRecentSolves)
	e.GET("/api/v1/logs", s.GetLogs)
	e.GET("/api/v1/client-behaviors", s.GetClientBehaviors)
	
	return e
}