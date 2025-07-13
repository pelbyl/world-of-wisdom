package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	PostgresHost     string
	PostgresPort     int
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	PostgresSSLMode  string

	RedisHost     string
	RedisPort     int
	RedisPassword string
	RedisDB       int
}

type Database struct {
	Postgres *sqlx.DB
	Redis    *redis.Client
	ctx      context.Context
}

func NewDatabase(cfg Config) (*Database, error) {
	// PostgreSQL connection
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDB, cfg.PostgresSSLMode)

	db, err := sqlx.Connect("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	// Redis connection
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Println("✅ Connected to PostgreSQL and Redis")

	return &Database{
		Postgres: db,
		Redis:    rdb,
		ctx:      ctx,
	}, nil
}

func (d *Database) Close() error {
	if err := d.Postgres.Close(); err != nil {
		return fmt.Errorf("failed to close PostgreSQL: %w", err)
	}
	if err := d.Redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis: %w", err)
	}
	return nil
}

// Challenge operations
type Challenge struct {
	ID            string     `db:"id" json:"id"`
	Seed          string     `db:"seed" json:"seed"`
	Difficulty    int        `db:"difficulty" json:"difficulty"`
	Algorithm     string     `db:"algorithm" json:"algorithm"`
	ClientID      string     `db:"client_id" json:"clientId"`
	Status        string     `db:"status" json:"status"`
	CreatedAt     time.Time  `db:"created_at" json:"createdAt"`
	SolvedAt      *time.Time `db:"solved_at" json:"solvedAt,omitempty"`
	ExpiresAt     time.Time  `db:"expires_at" json:"expiresAt"`
	Argon2Time    *int       `db:"argon2_time" json:"argon2Time,omitempty"`
	Argon2Memory  *int       `db:"argon2_memory" json:"argon2Memory,omitempty"`
	Argon2Threads *int       `db:"argon2_threads" json:"argon2Threads,omitempty"`
	Argon2Keylen  *int       `db:"argon2_keylen" json:"argon2Keylen,omitempty"`
}

type Solution struct {
	ID          string    `db:"id" json:"id"`
	ChallengeID string    `db:"challenge_id" json:"challengeId"`
	Nonce       string    `db:"nonce" json:"nonce"`
	Hash        string    `db:"hash" json:"hash"`
	Attempts    int       `db:"attempts" json:"attempts"`
	SolveTimeMs int64     `db:"solve_time_ms" json:"solveTimeMs"`
	Verified    bool      `db:"verified" json:"verified"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
}

type Connection struct {
	ID                  string     `db:"id" json:"id"`
	ClientID            string     `db:"client_id" json:"clientId"`
	RemoteAddr          string     `db:"remote_addr" json:"remoteAddr"`
	Status              string     `db:"status" json:"status"`
	Algorithm           string     `db:"algorithm" json:"algorithm"`
	ConnectedAt         time.Time  `db:"connected_at" json:"connectedAt"`
	DisconnectedAt      *time.Time `db:"disconnected_at" json:"disconnectedAt,omitempty"`
	ChallengesAttempted int        `db:"challenges_attempted" json:"challengesAttempted"`
	ChallengesCompleted int        `db:"challenges_completed" json:"challengesCompleted"`
	TotalSolveTimeMs    int64      `db:"total_solve_time_ms" json:"totalSolveTimeMs"`
}

func (d *Database) CreateChallenge(challenge *Challenge) error {
	query := `
		INSERT INTO challenges (seed, difficulty, algorithm, client_id, status, expires_at, argon2_time, argon2_memory, argon2_threads, argon2_keylen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	err := d.Postgres.QueryRow(query,
		challenge.Seed, challenge.Difficulty, challenge.Algorithm, challenge.ClientID,
		challenge.Status, challenge.ExpiresAt, challenge.Argon2Time, challenge.Argon2Memory,
		challenge.Argon2Threads, challenge.Argon2Keylen).Scan(&challenge.ID, &challenge.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create challenge: %w", err)
	}

	// Cache in Redis for quick lookup
	key := fmt.Sprintf("challenge:%s", challenge.ID)
	err = d.Redis.HMSet(d.ctx, key,
		"seed", challenge.Seed,
		"difficulty", challenge.Difficulty,
		"algorithm", challenge.Algorithm,
		"client_id", challenge.ClientID,
		"status", challenge.Status,
	).Err()

	if err != nil {
		log.Printf("Warning: failed to cache challenge in Redis: %v", err)
	}

	d.Redis.Expire(d.ctx, key, 5*time.Minute)

	return nil
}

func (d *Database) GetChallenge(id string) (*Challenge, error) {
	// Try Redis first
	key := fmt.Sprintf("challenge:%s", id)
	cached := d.Redis.HGetAll(d.ctx, key).Val()

	if len(cached) > 0 && cached["seed"] != "" {
		challenge := &Challenge{
			ID:        id,
			Seed:      cached["seed"],
			Algorithm: cached["algorithm"],
			ClientID:  cached["client_id"],
			Status:    cached["status"],
		}

		if difficulty, ok := cached["difficulty"]; ok {
			fmt.Sscanf(difficulty, "%d", &challenge.Difficulty)
		}

		return challenge, nil
	}

	// Fallback to PostgreSQL
	var challenge Challenge
	query := `SELECT * FROM challenges WHERE id = $1`
	err := d.Postgres.Get(&challenge, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("challenge not found")
		}
		return nil, fmt.Errorf("failed to get challenge: %w", err)
	}

	return &challenge, nil
}

func (d *Database) UpdateChallengeStatus(id string, status string, solvedAt *time.Time) error {
	query := `UPDATE challenges SET status = $1, solved_at = $2 WHERE id = $3`
	_, err := d.Postgres.Exec(query, status, solvedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update challenge status: %w", err)
	}

	// Update cache
	key := fmt.Sprintf("challenge:%s", id)
	d.Redis.HSet(d.ctx, key, "status", status)

	return nil
}

func (d *Database) CreateSolution(solution *Solution) error {
	query := `
		INSERT INTO solutions (challenge_id, nonce, hash, attempts, solve_time_ms, verified)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	err := d.Postgres.QueryRow(query,
		solution.ChallengeID, solution.Nonce, solution.Hash,
		solution.Attempts, solution.SolveTimeMs, solution.Verified).
		Scan(&solution.ID, &solution.CreatedAt)

	return err
}

func (d *Database) CreateConnection(conn *Connection) error {
	query := `
		INSERT INTO connections (client_id, remote_addr, status, algorithm)
		VALUES ($1, $2, $3, $4)
		RETURNING id, connected_at`

	err := d.Postgres.QueryRow(query,
		conn.ClientID, conn.RemoteAddr, conn.Status, conn.Algorithm).
		Scan(&conn.ID, &conn.ConnectedAt)

	return err
}

func (d *Database) UpdateConnection(id string, status string, challengesAttempted, challengesCompleted int, totalSolveTimeMs int64, disconnectedAt *time.Time) error {
	query := `
		UPDATE connections 
		SET status = $1, challenges_attempted = $2, challenges_completed = $3, 
		    total_solve_time_ms = $4, disconnected_at = $5
		WHERE id = $1`

	_, err := d.Postgres.Exec(query, status, challengesAttempted, challengesCompleted, totalSolveTimeMs, disconnectedAt)
	return err
}

func (d *Database) RecordMetric(name string, value float64, labels map[string]interface{}) error {
	query := `INSERT INTO metrics (metric_name, metric_value, labels) VALUES ($1, $2, $3)`

	var labelsJSON []byte
	var err error
	if labels != nil {
		labelsJSON, err = json.Marshal(labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}
	} else {
		labelsJSON = []byte("{}")
	}

	_, err = d.Postgres.Exec(query, name, value, labelsJSON)
	return err
}

// Stats and analytics
func (d *Database) GetChallengeStats(hours int) ([]map[string]interface{}, error) {
	query := `
		SELECT algorithm, difficulty, status, 
		       COUNT(*) as count,
		       AVG(EXTRACT(EPOCH FROM (solved_at - created_at)) * 1000) as avg_solve_time_ms
		FROM challenges 
		WHERE created_at >= NOW() - INTERVAL '%d hours'
		GROUP BY algorithm, difficulty, status
		ORDER BY algorithm, difficulty, status`

	rows, err := d.Postgres.Query(fmt.Sprintf(query, hours))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var algorithm, status string
		var difficulty, count int
		var avgSolveTime sql.NullFloat64

		err := rows.Scan(&algorithm, &difficulty, &status, &count, &avgSolveTime)
		if err != nil {
			return nil, err
		}

		result := map[string]interface{}{
			"algorithm":         algorithm,
			"difficulty":        difficulty,
			"status":            status,
			"count":             count,
			"avg_solve_time_ms": nil,
		}

		if avgSolveTime.Valid {
			result["avg_solve_time_ms"] = avgSolveTime.Float64
		}

		results = append(results, result)
	}

	return results, nil
}
