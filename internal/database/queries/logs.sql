-- name: CreateLog :one
INSERT INTO logs (
    timestamp, level, message, metadata
) VALUES (
    COALESCE($1, NOW()), $2, $3, $4
) RETURNING *;

-- name: GetRecentLogs :many
SELECT * FROM logs
ORDER BY timestamp DESC
LIMIT $1;

-- name: GetLogsPaginated :many
SELECT * FROM logs
WHERE ($1::timestamptz IS NULL OR timestamp < $1)
ORDER BY timestamp DESC
LIMIT $2;

-- name: GetLogsByLevel :many
SELECT * FROM logs
WHERE level = $1
ORDER BY timestamp DESC
LIMIT $2;

-- name: GetLogsInTimeRange :many
SELECT * FROM logs
WHERE timestamp >= $1 AND timestamp <= $2
ORDER BY timestamp DESC;

-- name: CountLogsByLevel :many
SELECT level, COUNT(*) as count
FROM logs
WHERE timestamp >= NOW() - INTERVAL '24 hours'
GROUP BY level;

-- name: DeleteOldLogs :exec
DELETE FROM logs
WHERE timestamp < NOW() - INTERVAL '7 days';