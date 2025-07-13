package apiserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) getAPIDocumentation(c *gin.Context) {
	docs := gin.H{
		"title":       "Word of Wisdom REST API",
		"version":     "1.0.0",
		"description": "REST API for Word of Wisdom PoW server with type-safe database operations",
		"baseURL":     "/api/v1",
		"endpoints": gin.H{
			"challenges": gin.H{
				"GET /challenges":                  "Get recent challenges (limit query param)",
				"POST /challenges":                 "Create new challenge",
				"GET /challenges/:id":              "Get challenge by ID",
				"PATCH /challenges/:id/status":     "Update challenge status",
				"GET /challenges/client/:clientId": "Get pending challenge for client",
				"GET /challenges/difficulty/:diff": "Get challenges by difficulty (1-6)",
				"GET /challenges/algorithm/:algo":  "Get challenges by algorithm (sha256|argon2)",
			},
			"solutions": gin.H{
				"GET /solutions":               "Get recent solutions (limit query param)",
				"POST /solutions":              "Create new solution",
				"GET /solutions/:id":           "Get solution by ID",
				"GET /solutions/challenge/:id": "Get solutions for challenge",
				"PATCH /solutions/:id/verify":  "Verify/unverify solution",
				"GET /solutions/stats":         "Get solution statistics",
			},
			"connections": gin.H{
				"GET /connections":                  "Get recent connections (limit query param)",
				"POST /connections":                 "Create new connection",
				"GET /connections/:id":              "Get connection by ID",
				"GET /connections/client/:clientId": "Get active connection for client",
				"PATCH /connections/:id/status":     "Update connection status",
				"PATCH /connections/:id/stats":      "Update connection statistics",
				"GET /connections/active":           "Get all active connections",
				"GET /connections/stats":            "Get connection statistics",
			},
			"blocks": gin.H{
				"GET /blocks":            "Get recent blocks (limit query param)",
				"POST /blocks":           "Create new block",
				"GET /blocks/:id":        "Get block by ID",
				"GET /blocks/index/:idx": "Get block by index",
				"GET /blocks/latest":     "Get latest block",
				"GET /blocks/blockchain": "Get entire blockchain",
			},
			"metrics": gin.H{
				"POST /metrics":              "Record new metric",
				"GET /metrics/name/:name":    "Get metrics by name (limit query param)",
				"GET /metrics/recent":        "Get most recent metrics",
				"GET /metrics/history/:name": "Get metric history (bucketed)",
				"GET /metrics/system":        "Get key system metrics",
			},
		},
		"examples": gin.H{
			"createChallenge": gin.H{
				"url": "POST /api/v1/challenges",
				"body": gin.H{
					"seed":          "random-seed-123",
					"difficulty":    2,
					"algorithm":     "argon2",
					"clientId":      "client-001",
					"argon2Time":    3,
					"argon2Memory":  65536,
					"argon2Threads": 4,
					"argon2KeyLen":  32,
				},
			},
			"createSolution": gin.H{
				"url": "POST /api/v1/solutions",
				"body": gin.H{
					"challengeId": "uuid-here",
					"nonce":       "12345",
					"hash":        "abc123...",
					"attempts":    42,
					"solveTimeMs": 1500,
					"verified":    true,
				},
			},
			"createConnection": gin.H{
				"url": "POST /api/v1/connections",
				"body": gin.H{
					"clientId":   "client-001",
					"remoteAddr": "192.168.1.100",
					"algorithm":  "argon2",
				},
			},
			"recordMetric": gin.H{
				"url": "POST /api/v1/metrics",
				"body": gin.H{
					"metricName":  "current_difficulty",
					"metricValue": 2.5,
					"labels": gin.H{
						"algorithm": "argon2",
						"instance":  "server-1",
					},
					"serverInstance": "wisdom-server-1",
				},
			},
		},
		"responseFormat": gin.H{
			"success": gin.H{
				"data":      "...",
				"timestamp": "2024-01-01T00:00:00Z",
			},
			"error": gin.H{
				"error":     "Error message",
				"timestamp": "2024-01-01T00:00:00Z",
				"details":   "Detailed error (debug mode only)",
			},
		},
		"queryParameters": gin.H{
			"limit": "Limit number of results (default: 50, max: 1000)",
		},
		"algorithms":         []string{"sha256", "argon2"},
		"challengeStatuses":  []string{"pending", "solving", "completed", "failed", "expired"},
		"connectionStatuses": []string{"connected", "solving", "disconnected", "failed"},
	}

	c.JSON(http.StatusOK, docs)
}
