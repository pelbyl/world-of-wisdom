-- name: RecordMetric :exec
INSERT INTO metrics (metric_name, metric_value, labels, server_instance)
VALUES ($1, $2, $3, $4);

-- name: GetMetricsByName :many
SELECT * FROM metrics 
WHERE metric_name = $1 
  AND time >= NOW() - INTERVAL '1 hour'
ORDER BY time DESC
LIMIT $2;

-- name: GetRecentMetrics :many
SELECT DISTINCT ON (metric_name) 
    metric_name, metric_value, labels, time
FROM metrics 
WHERE time >= NOW() - INTERVAL '1 hour'
ORDER BY metric_name, time DESC;

-- name: GetMetricHistory :many
SELECT 
    time_bucket('1 minute', time) AS bucket,
    metric_name,
    AVG(metric_value) as avg_value,
    MIN(metric_value) as min_value,
    MAX(metric_value) as max_value,
    COUNT(*) as sample_count
FROM metrics 
WHERE metric_name = $1 
  AND time >= NOW() - INTERVAL '1 hour'
GROUP BY bucket, metric_name
ORDER BY bucket DESC;

-- name: GetSystemMetrics :many
SELECT 
    metric_name,
    metric_value,
    labels,
    time
FROM metrics 
WHERE metric_name IN ('current_difficulty', 'active_connections', 'total_challenges', 'hash_rate')
  AND time >= NOW() - INTERVAL '5 minutes'
ORDER BY metric_name, time DESC;

-- name: CountDifficultyAdjustments :one
SELECT COUNT(*) as adjustment_count
FROM metrics 
WHERE metric_name = 'difficulty_adjustment'
  AND time >= NOW() - INTERVAL '1 hour';