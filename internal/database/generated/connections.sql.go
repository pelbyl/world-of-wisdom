// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: connections.sql

package db

import (
	"context"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
)

const createConnection = `-- name: CreateConnection :one
INSERT INTO connections (
    client_id, remote_addr, status, algorithm
) VALUES (
    $1, $2, $3, $4
) RETURNING id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms
`

type CreateConnectionParams struct {
	ClientID   string           `json:"client_id"`
	RemoteAddr netip.Addr       `json:"remote_addr"`
	Status     ConnectionStatus `json:"status"`
	Algorithm  PowAlgorithm     `json:"algorithm"`
}

func (q *Queries) CreateConnection(ctx context.Context, db DBTX, arg CreateConnectionParams) (Connection, error) {
	row := db.QueryRow(ctx, createConnection,
		arg.ClientID,
		arg.RemoteAddr,
		arg.Status,
		arg.Algorithm,
	)
	var i Connection
	err := row.Scan(
		&i.ID,
		&i.ClientID,
		&i.RemoteAddr,
		&i.Status,
		&i.Algorithm,
		&i.ConnectedAt,
		&i.DisconnectedAt,
		&i.ChallengesAttempted,
		&i.ChallengesCompleted,
		&i.TotalSolveTimeMs,
	)
	return i, err
}

const getActiveConnections = `-- name: GetActiveConnections :many
SELECT id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms FROM connections 
WHERE status IN ('connected', 'solving')
ORDER BY connected_at DESC
`

func (q *Queries) GetActiveConnections(ctx context.Context, db DBTX) ([]Connection, error) {
	rows, err := db.Query(ctx, getActiveConnections)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var i Connection
		if err := rows.Scan(
			&i.ID,
			&i.ClientID,
			&i.RemoteAddr,
			&i.Status,
			&i.Algorithm,
			&i.ConnectedAt,
			&i.DisconnectedAt,
			&i.ChallengesAttempted,
			&i.ChallengesCompleted,
			&i.TotalSolveTimeMs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getConnection = `-- name: GetConnection :one
SELECT id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms FROM connections WHERE id = $1
`

func (q *Queries) GetConnection(ctx context.Context, db DBTX, id pgtype.UUID) (Connection, error) {
	row := db.QueryRow(ctx, getConnection, id)
	var i Connection
	err := row.Scan(
		&i.ID,
		&i.ClientID,
		&i.RemoteAddr,
		&i.Status,
		&i.Algorithm,
		&i.ConnectedAt,
		&i.DisconnectedAt,
		&i.ChallengesAttempted,
		&i.ChallengesCompleted,
		&i.TotalSolveTimeMs,
	)
	return i, err
}

const getConnectionByClientID = `-- name: GetConnectionByClientID :one
SELECT id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms FROM connections 
WHERE client_id = $1 AND status IN ('connected', 'solving')
ORDER BY connected_at DESC 
LIMIT 1
`

func (q *Queries) GetConnectionByClientID(ctx context.Context, db DBTX, clientID string) (Connection, error) {
	row := db.QueryRow(ctx, getConnectionByClientID, clientID)
	var i Connection
	err := row.Scan(
		&i.ID,
		&i.ClientID,
		&i.RemoteAddr,
		&i.Status,
		&i.Algorithm,
		&i.ConnectedAt,
		&i.DisconnectedAt,
		&i.ChallengesAttempted,
		&i.ChallengesCompleted,
		&i.TotalSolveTimeMs,
	)
	return i, err
}

const getConnectionStats = `-- name: GetConnectionStats :one
SELECT 
    COUNT(*) as total_connections,
    COUNT(CASE WHEN status IN ('connected', 'solving') THEN 1 END) as active_connections,
    AVG(challenges_completed) as avg_challenges_completed,
    AVG(total_solve_time_ms) as avg_solve_time_ms
FROM connections 
WHERE connected_at >= NOW() - INTERVAL '24 hours'
`

type GetConnectionStatsRow struct {
	TotalConnections       int64   `json:"total_connections"`
	ActiveConnections      int64   `json:"active_connections"`
	AvgChallengesCompleted float64 `json:"avg_challenges_completed"`
	AvgSolveTimeMs         float64 `json:"avg_solve_time_ms"`
}

func (q *Queries) GetConnectionStats(ctx context.Context, db DBTX) (GetConnectionStatsRow, error) {
	row := db.QueryRow(ctx, getConnectionStats)
	var i GetConnectionStatsRow
	err := row.Scan(
		&i.TotalConnections,
		&i.ActiveConnections,
		&i.AvgChallengesCompleted,
		&i.AvgSolveTimeMs,
	)
	return i, err
}

const getConnectionsFiltered = `-- name: GetConnectionsFiltered :many
SELECT 
    id,
    client_id,
    remote_addr,
    status,
    algorithm,
    connected_at,
    disconnected_at,
    challenges_attempted,
    challenges_completed,
    total_solve_time_ms
FROM connections
WHERE 
    ($1::connection_status IS NULL OR status = $1)
    AND connected_at >= NOW() - INTERVAL '24 hours'
ORDER BY connected_at DESC
LIMIT 100
`

// Get connections with optional status filter for API endpoint
func (q *Queries) GetConnectionsFiltered(ctx context.Context, db DBTX, status ConnectionStatus) ([]Connection, error) {
	rows, err := db.Query(ctx, getConnectionsFiltered, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var i Connection
		if err := rows.Scan(
			&i.ID,
			&i.ClientID,
			&i.RemoteAddr,
			&i.Status,
			&i.Algorithm,
			&i.ConnectedAt,
			&i.DisconnectedAt,
			&i.ChallengesAttempted,
			&i.ChallengesCompleted,
			&i.TotalSolveTimeMs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getRecentConnections = `-- name: GetRecentConnections :many
SELECT id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms FROM connections 
WHERE connected_at >= NOW() - INTERVAL '1 hour'
ORDER BY connected_at DESC
LIMIT $1
`

func (q *Queries) GetRecentConnections(ctx context.Context, db DBTX, limit int32) ([]Connection, error) {
	rows, err := db.Query(ctx, getRecentConnections, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Connection{}
	for rows.Next() {
		var i Connection
		if err := rows.Scan(
			&i.ID,
			&i.ClientID,
			&i.RemoteAddr,
			&i.Status,
			&i.Algorithm,
			&i.ConnectedAt,
			&i.DisconnectedAt,
			&i.ChallengesAttempted,
			&i.ChallengesCompleted,
			&i.TotalSolveTimeMs,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const updateConnectionStats = `-- name: UpdateConnectionStats :one
UPDATE connections 
SET challenges_attempted = challenges_attempted + $2,
    challenges_completed = challenges_completed + $3,
    total_solve_time_ms = total_solve_time_ms + $4
WHERE id = $1 
RETURNING id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms
`

type UpdateConnectionStatsParams struct {
	ID                  pgtype.UUID `json:"id"`
	ChallengesAttempted pgtype.Int4 `json:"challenges_attempted"`
	ChallengesCompleted pgtype.Int4 `json:"challenges_completed"`
	TotalSolveTimeMs    pgtype.Int8 `json:"total_solve_time_ms"`
}

func (q *Queries) UpdateConnectionStats(ctx context.Context, db DBTX, arg UpdateConnectionStatsParams) (Connection, error) {
	row := db.QueryRow(ctx, updateConnectionStats,
		arg.ID,
		arg.ChallengesAttempted,
		arg.ChallengesCompleted,
		arg.TotalSolveTimeMs,
	)
	var i Connection
	err := row.Scan(
		&i.ID,
		&i.ClientID,
		&i.RemoteAddr,
		&i.Status,
		&i.Algorithm,
		&i.ConnectedAt,
		&i.DisconnectedAt,
		&i.ChallengesAttempted,
		&i.ChallengesCompleted,
		&i.TotalSolveTimeMs,
	)
	return i, err
}

const updateConnectionStatus = `-- name: UpdateConnectionStatus :one
UPDATE connections 
SET status = $1::connection_status, 
    disconnected_at = CASE WHEN $1::connection_status = 'disconnected' THEN NOW() ELSE disconnected_at END
WHERE id = $2 
RETURNING id, client_id, remote_addr, status, algorithm, connected_at, disconnected_at, challenges_attempted, challenges_completed, total_solve_time_ms
`

type UpdateConnectionStatusParams struct {
	Status ConnectionStatus `json:"status"`
	ID     pgtype.UUID      `json:"id"`
}

func (q *Queries) UpdateConnectionStatus(ctx context.Context, db DBTX, arg UpdateConnectionStatusParams) (Connection, error) {
	row := db.QueryRow(ctx, updateConnectionStatus, arg.Status, arg.ID)
	var i Connection
	err := row.Scan(
		&i.ID,
		&i.ClientID,
		&i.RemoteAddr,
		&i.Status,
		&i.Algorithm,
		&i.ConnectedAt,
		&i.DisconnectedAt,
		&i.ChallengesAttempted,
		&i.ChallengesCompleted,
		&i.TotalSolveTimeMs,
	)
	return i, err
}
