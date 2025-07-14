package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	db "world-of-wisdom/internal/database/generated"
)

// repository implements the Repository interface
type repository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

// txRepository implements the Repository interface for transactions
type txRepository struct {
	tx      pgx.Tx
	queries *db.Queries
}

// Challenges returns the challenge repository for transactions
func (r *txRepository) Challenges() ChallengeRepository {
	return &challengeRepo{queries: r.queries, db: r.tx}
}

// Solutions returns the solution repository for transactions
func (r *txRepository) Solutions() SolutionRepository {
	return &solutionRepo{queries: r.queries, db: r.tx}
}

// Connections returns the connection repository for transactions
func (r *txRepository) Connections() ConnectionRepository {
	return &connectionRepo{queries: r.queries, db: r.tx}
}

// Metrics returns the metrics repository for transactions
func (r *txRepository) Metrics() MetricsRepository {
	return &metricsRepo{queries: r.queries, db: r.tx}
}

// Logs returns the log repository for transactions
func (r *txRepository) Logs() LogRepository {
	return &logRepo{queries: r.queries, db: r.tx}
}

// Queries returns direct access to generated queries for transactions
func (r *txRepository) Queries() *db.Queries {
	return r.queries
}

// GetBlockchainStats returns blockchain statistics for transactions
func (r *txRepository) GetBlockchainStats(ctx context.Context) (db.GetBlockchainStatsRow, error) {
	return r.queries.GetBlockchainStats(ctx, r.tx)
}

// WithTx is not supported for transaction repositories
func (r *txRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	return fmt.Errorf("nested transactions are not supported")
}

// New creates a new repository instance
func New(pool *pgxpool.Pool) Repository {
	return &repository{
		pool:    pool,
		queries: db.New(),
	}
}

// Challenges returns the challenge repository
func (r *repository) Challenges() ChallengeRepository {
	return &challengeRepo{queries: r.queries, db: r.pool}
}

// Solutions returns the solution repository
func (r *repository) Solutions() SolutionRepository {
	return &solutionRepo{queries: r.queries, db: r.pool}
}

// Connections returns the connection repository
func (r *repository) Connections() ConnectionRepository {
	return &connectionRepo{queries: r.queries, db: r.pool}
}

// Metrics returns the metrics repository
func (r *repository) Metrics() MetricsRepository {
	return &metricsRepo{queries: r.queries, db: r.pool}
}

// Logs returns the log repository
func (r *repository) Logs() LogRepository {
	return &logRepo{queries: r.queries, db: r.pool}
}

// Queries returns direct access to generated queries
func (r *repository) Queries() *db.Queries {
	return r.queries
}

// GetBlockchainStats returns blockchain statistics
func (r *repository) GetBlockchainStats(ctx context.Context) (db.GetBlockchainStatsRow, error) {
	return r.queries.GetBlockchainStats(ctx, r.pool)
}

// WithTx executes a function within a transaction
func (r *repository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txRepo := &txRepository{
		tx:      tx,
		queries: r.queries,
	}

	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

