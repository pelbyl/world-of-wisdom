package behavior

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"sync"
	"time"

	generated "world-of-wisdom/internal/database/generated"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ClientBehavior struct {
	IP                    netip.Addr
	ConnectionCount       int
	FailureRate           float64
	AvgSolveTime          time.Duration
	LastConnection        time.Time
	ReconnectRate         float64
	Difficulty            int
	ReputationScore       float64
	SuspiciousScore       float64
	ConnectionTimestampID pgtype.UUID
}

type Tracker struct {
	dbpool  *pgxpool.Pool
	queries *generated.Queries
	cache   map[string]*ClientBehavior
	mu      sync.RWMutex
}

func NewTracker(dbpool *pgxpool.Pool) *Tracker {
	return &Tracker{
		dbpool:  dbpool,
		queries: generated.New(),
		cache:   make(map[string]*ClientBehavior),
	}
}

func (t *Tracker) GetClientBehavior(ctx context.Context, ip netip.Addr) (*ClientBehavior, error) {
	ipStr := ip.String()
	
	// Check cache first
	t.mu.RLock()
	if cached, ok := t.cache[ipStr]; ok {
		t.mu.RUnlock()
		return cached, nil
	}
	t.mu.RUnlock()

	// Query from database
	behavior, err := t.queries.GetClientBehaviorByIP(ctx, t.dbpool, ip)
	if err != nil {
		// Create new client behavior if not found
		newBehavior, err := t.queries.CreateClientBehavior(ctx, t.dbpool, ip)
		if err != nil {
			return nil, fmt.Errorf("failed to create client behavior: %w", err)
		}
		behavior = newBehavior
	}

	// Create ClientBehavior struct
	cb := &ClientBehavior{
		IP:              ip,
		ConnectionCount: int(behavior.ConnectionCount.Int32),
		FailureRate:     behavior.FailureRate.Float64,
		AvgSolveTime:    time.Duration(behavior.AvgSolveTimeMs.Int64) * time.Millisecond,
		LastConnection:  behavior.LastConnection.Time,
		ReconnectRate:   behavior.ReconnectRate.Float64,
		Difficulty:      int(behavior.Difficulty.Int32),
		ReputationScore: behavior.ReputationScore.Float64,
		SuspiciousScore: behavior.SuspiciousActivityScore.Float64,
	}

	// Update cache
	t.mu.Lock()
	t.cache[ipStr] = cb
	t.mu.Unlock()

	return cb, nil
}

func (t *Tracker) RecordConnection(ctx context.Context, ip netip.Addr) (*ClientBehavior, error) {
	// Update or create client behavior
	behavior, err := t.queries.UpdateClientBehavior(ctx, t.dbpool, ip)
	if err != nil {
		// Try to create if doesn't exist
		behavior, err = t.queries.CreateClientBehavior(ctx, t.dbpool, ip)
		if err != nil {
			return nil, fmt.Errorf("failed to record connection: %w", err)
		}
	}

	// Create connection timestamp
	connTimestamp, err := t.queries.CreateConnectionTimestamp(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to create connection timestamp: %v", err)
	}

	// Update reconnect rate
	err = t.queries.UpdateClientReconnectRate(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to update reconnect rate: %v", err)
	}

	// Calculate and update difficulty
	oldDifficulty := behavior.Difficulty.Int32
	newDifficulty, err := t.queries.CalculateAndUpdateClientDifficulty(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to calculate adaptive difficulty: %v", err)
		newDifficulty = behavior.Difficulty
	}
	
	// Log difficulty change if it occurred
	if oldDifficulty != newDifficulty.Int32 {
		log.Printf("Client %s difficulty changed from %d to %d", ip.String(), oldDifficulty, newDifficulty.Int32)
	}

	// Update suspicious activity score
	err = t.queries.UpdateSuspiciousActivityScore(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to update suspicious activity score: %v", err)
	}

	// Create updated ClientBehavior
	cb := &ClientBehavior{
		IP:                    ip,
		ConnectionCount:       int(behavior.ConnectionCount.Int32),
		FailureRate:           behavior.FailureRate.Float64,
		AvgSolveTime:          time.Duration(behavior.AvgSolveTimeMs.Int64) * time.Millisecond,
		LastConnection:        behavior.LastConnection.Time,
		ReconnectRate:         behavior.ReconnectRate.Float64,
		Difficulty:            int(newDifficulty.Int32),
		ReputationScore:       behavior.ReputationScore.Float64,
		SuspiciousScore:       behavior.SuspiciousActivityScore.Float64,
		ConnectionTimestampID: connTimestamp.ID,
	}

	// Update cache
	t.mu.Lock()
	t.cache[ip.String()] = cb
	t.mu.Unlock()

	return cb, nil
}

func (t *Tracker) RecordChallengeResult(ctx context.Context, ip netip.Addr, success bool, solveTime time.Duration) error {
	// Update challenge statistics
	err := t.queries.UpdateClientChallengeStats(ctx, t.dbpool, generated.UpdateClientChallengeStatsParams{
		IpAddress:    ip,
		IsSuccessful: success,
		SolveTimeMs:  solveTime.Milliseconds(),
	})
	if err != nil {
		return fmt.Errorf("failed to update challenge stats: %w", err)
	}

	// Update reputation based on result
	err = t.queries.UpdateClientReputation(ctx, t.dbpool, generated.UpdateClientReputationParams{
		IpAddress:        ip,
		ChallengeSuccess: success,
	})
	if err != nil {
		log.Printf("Failed to update reputation: %v", err)
	}

	// Recalculate difficulty
	_, err = t.queries.CalculateAndUpdateClientDifficulty(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to recalculate difficulty: %v", err)
	}

	// Update suspicious activity score
	err = t.queries.UpdateSuspiciousActivityScore(ctx, t.dbpool, ip)
	if err != nil {
		log.Printf("Failed to update suspicious activity score: %v", err)
	}

	// Clear cache entry to force refresh
	t.mu.Lock()
	delete(t.cache, ip.String())
	t.mu.Unlock()

	return nil
}

func (t *Tracker) RecordDisconnection(ctx context.Context, connectionTimestampID pgtype.UUID, challengeCompleted bool) error {
	if connectionTimestampID == (pgtype.UUID{}) {
		return nil // Skip if no valid ID
	}

	err := t.queries.UpdateConnectionTimestamp(ctx, t.dbpool, generated.UpdateConnectionTimestampParams{
		ID:                 connectionTimestampID,
		ChallengeCompleted: challengeCompleted,
	})
	if err != nil {
		return fmt.Errorf("failed to update connection timestamp: %w", err)
	}

	return nil
}

func (t *Tracker) GetActiveClients(ctx context.Context, limit int) ([]generated.GetActiveClientsRow, error) {
	return t.queries.GetActiveClients(ctx, t.dbpool, int32(limit))
}

func (t *Tracker) GetClientStats(ctx context.Context, limit int) ([]generated.GetClientBehaviorStatsRow, error) {
	return t.queries.GetClientBehaviorStats(ctx, t.dbpool, int32(limit))
}

func (t *Tracker) GetAggressiveClients(ctx context.Context, limit int) ([]generated.GetTopAggressiveClientsRow, error) {
	return t.queries.GetTopAggressiveClients(ctx, t.dbpool, int32(limit))
}

func (t *Tracker) ClearCache() {
	t.mu.Lock()
	t.cache = make(map[string]*ClientBehavior)
	t.mu.Unlock()
}

func (t *Tracker) GetCachedBehaviors() map[string]*ClientBehavior {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	copy := make(map[string]*ClientBehavior)
	for k, v := range t.cache {
		copy[k] = v
	}
	return copy
}