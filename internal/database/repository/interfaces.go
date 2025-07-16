package repository

import (
	"context"

	"github.com/google/uuid"
	db "world-of-wisdom/internal/database/generated"
)

// Type aliases for generated types
type (
	Challenge                      = db.Challenge
	CreateChallengeParams          = db.CreateChallengeParams
	UpdateChallengeStatusParams    = db.UpdateChallengeStatusParams
	GetChallengesFilteredParams    = db.GetChallengesFilteredParams
	GetChallengesFilteredRow       = db.GetChallengesFilteredRow
	GetChallengeStatsRow           = db.GetChallengeStatsRow
	ChallengeStatus                = db.ChallengeStatus
	
	Solution                       = db.Solution
	CreateSolutionParams           = db.CreateSolutionParams
	GetRecentSolutionsRow          = db.GetRecentSolutionsRow
	
	Connection                     = db.Connection
	CreateConnectionParams         = db.CreateConnectionParams
	UpdateConnectionStatusParams   = db.UpdateConnectionStatusParams
	GetConnectionStatsRow          = db.GetConnectionStatsRow
	ConnectionStatus               = db.ConnectionStatus
	
	RecordMetricParams             = db.RecordMetricParams
	GetSystemMetricsRow            = db.GetSystemMetricsRow
	GetMetricsByTimeRangeParams    = db.GetMetricsByTimeRangeParams
	GetMetricsByTimeRangeRow       = db.GetMetricsByTimeRangeRow
	GetAggregatedMetricsParams     = db.GetAggregatedMetricsParams
	GetAggregatedMetricsRow        = db.GetAggregatedMetricsRow
	
	Log                            = db.Log
	CreateLogParams                = db.CreateLogParams
	GetLogsByLevelParams           = db.GetLogsByLevelParams
	GetLogsPaginatedParams         = db.GetLogsPaginatedParams
	
	Queries                        = db.Queries
)

// ChallengeRepository defines challenge-related database operations
type ChallengeRepository interface {
	Create(ctx context.Context, challenge CreateChallengeParams) (Challenge, error)
	GetByID(ctx context.Context, id uuid.UUID) (Challenge, error)
	GetByClientID(ctx context.Context, clientID string) (Challenge, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ChallengeStatus) error
	GetFiltered(ctx context.Context, params GetChallengesFilteredParams) ([]GetChallengesFilteredRow, error)
	GetRecent(ctx context.Context, limit int32) ([]Challenge, error)
	GetStats(ctx context.Context) (GetChallengeStatsRow, error)
}

// SolutionRepository defines solution-related database operations
type SolutionRepository interface {
	Create(ctx context.Context, solution CreateSolutionParams) (Solution, error)
	GetByID(ctx context.Context, id uuid.UUID) (Solution, error)
	GetByChallenge(ctx context.Context, challengeID uuid.UUID) ([]Solution, error)
	GetRecent(ctx context.Context, limit int32) ([]GetRecentSolutionsRow, error)
}

// ConnectionRepository defines connection-related database operations
type ConnectionRepository interface {
	Create(ctx context.Context, conn CreateConnectionParams) (Connection, error)
	GetByID(ctx context.Context, id uuid.UUID) (Connection, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ConnectionStatus) error
	GetActive(ctx context.Context) ([]Connection, error)
	GetFiltered(ctx context.Context, status ConnectionStatus) ([]Connection, error)
	GetStats(ctx context.Context) (GetConnectionStatsRow, error)
}

// MetricsRepository defines metrics-related database operations
type MetricsRepository interface {
	Record(ctx context.Context, metric RecordMetricParams) error
	GetSystem(ctx context.Context) ([]GetSystemMetricsRow, error)
	GetByTimeRange(ctx context.Context, params GetMetricsByTimeRangeParams) ([]GetMetricsByTimeRangeRow, error)
	GetAggregated(ctx context.Context, params GetAggregatedMetricsParams) ([]GetAggregatedMetricsRow, error)
}

// LogRepository defines log-related database operations
type LogRepository interface {
	Create(ctx context.Context, log CreateLogParams) (Log, error)
	GetRecent(ctx context.Context, limit int32) ([]Log, error)
	GetByLevel(ctx context.Context, params GetLogsByLevelParams) ([]Log, error)
	GetPaginated(ctx context.Context, params GetLogsPaginatedParams) ([]Log, error)
}

// Repository aggregates all repository interfaces
type Repository interface {
	Challenges() ChallengeRepository
	Solutions() SolutionRepository
	Connections() ConnectionRepository
	Metrics() MetricsRepository
	Logs() LogRepository
	
	// Direct queries access for complex operations
	Queries() *Queries
	
	// Transaction support
	WithTx(ctx context.Context, fn func(Repository) error) error
}