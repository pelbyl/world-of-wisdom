-- name: CreateConnection :one
INSERT INTO connections (
    client_id, remote_addr, status, algorithm
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetConnection :one
SELECT * FROM connections WHERE id = $1;

-- name: GetConnectionByClientID :one
SELECT * FROM connections 
WHERE client_id = $1 AND status IN ('connected', 'solving')
ORDER BY connected_at DESC 
LIMIT 1;

-- name: UpdateConnectionStatus :one
UPDATE connections 
SET status = $2, 
    disconnected_at = CASE WHEN $2 = 'disconnected' THEN NOW() ELSE disconnected_at END
WHERE id = $1 
RETURNING *;

-- name: UpdateConnectionStats :one
UPDATE connections 
SET challenges_attempted = challenges_attempted + $2,
    challenges_completed = challenges_completed + $3,
    total_solve_time_ms = total_solve_time_ms + $4
WHERE id = $1 
RETURNING *;

-- name: GetActiveConnections :many
SELECT * FROM connections 
WHERE status IN ('connected', 'solving')
ORDER BY connected_at DESC;

-- name: GetRecentConnections :many
SELECT * FROM connections 
WHERE connected_at >= NOW() - INTERVAL '1 hour'
ORDER BY connected_at DESC
LIMIT $1;

-- name: GetConnectionStats :one
SELECT 
    COUNT(*) as total_connections,
    COUNT(CASE WHEN status = 'connected' THEN 1 END) as active_connections,
    AVG(challenges_completed) as avg_challenges_completed,
    AVG(total_solve_time_ms) as avg_solve_time_ms
FROM connections 
WHERE connected_at >= NOW() - INTERVAL '24 hours';