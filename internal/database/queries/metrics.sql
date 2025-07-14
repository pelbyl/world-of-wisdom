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

-- name: GetMetricsByTimeRange :many
-- Get metrics within a specific time range with optional metric name filter
SELECT 
    time,
    metric_name,
    metric_value,
    labels
FROM metrics
WHERE time >= @start_time::TIMESTAMPTZ
  AND time <= @end_time::TIMESTAMPTZ
  AND (@metric_name::VARCHAR IS NULL OR metric_name = @metric_name)
ORDER BY time DESC
LIMIT 1000;

-- name: GetAggregatedMetrics :many
-- Get aggregated metrics with configurable time bucket
SELECT 
    time_bucket(@interval::INTERVAL, time) as bucket,
    metric_name,
    AVG(metric_value) as avg_value,
    MAX(metric_value) as max_value,
    MIN(metric_value) as min_value,
    COUNT(*) as sample_count
FROM metrics
WHERE time >= @start_time::TIMESTAMPTZ
  AND time <= @end_time::TIMESTAMPTZ
  AND (@metric_name::VARCHAR IS NULL OR metric_name = @metric_name)
GROUP BY bucket, metric_name
ORDER BY bucket DESC
LIMIT 500;

-- name: GetMetricsFromContinuousAggregate :many
-- Get metrics aggregated by time bucket (simplified for clean architecture)
SELECT 
    time_bucket('5 minutes', time) as time,
    metric_name,
    AVG(metric_value) as avg_value,
    MAX(metric_value) as max_value,
    MIN(metric_value) as min_value,
    COUNT(*) as sample_count,
    labels
FROM metrics
WHERE time >= @start_time::TIMESTAMPTZ
  AND time <= @end_time::TIMESTAMPTZ
  AND (@metric_name::VARCHAR IS NULL OR metric_name = @metric_name)
GROUP BY time_bucket('5 minutes', time), metric_name, labels
ORDER BY time DESC
LIMIT 500;