package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Database represents the database connection and configuration
type Database struct {
	pool *pgxpool.Pool
}

// Config holds database connection configuration
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// New creates a new database instance
func New(cfg Config) (*Database, error) {
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database, cfg.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{pool: pool}, nil
}

// Close closes the database connection pool
func (db *Database) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// Pool returns the underlying connection pool
func (db *Database) Pool() *pgxpool.Pool {
	return db.pool
}

// Health checks if the database is healthy
func (db *Database) Health(ctx context.Context) error {
	return db.pool.Ping(ctx)
}