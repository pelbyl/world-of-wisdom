-- name: GetChallengeStats :one
-- Get aggregated challenge statistics for the API
SELECT 
    COUNT(*) FILTER (WHERE status = 'pending') as pending_count,
    COUNT(*) FILTER (WHERE status = 'solving') as solving_count,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_count,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_count,
    COUNT(*) FILTER (WHERE status = 'expired') as expired_count,
    COUNT(*) as total_count,
    AVG(CASE 
        WHEN status = 'completed' AND solved_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (solved_at - created_at)) * 1000 
        ELSE NULL 
    END)::FLOAT as avg_solve_time_ms,
    COUNT(*) FILTER (WHERE algorithm = 'sha256') as sha256_count,
    COUNT(*) FILTER (WHERE algorithm = 'argon2') as argon2_count
FROM challenges
WHERE created_at >= NOW() - INTERVAL '24 hours';


-- name: GetChallengeDistribution :many
-- Get distribution of challenges by difficulty and algorithm
SELECT 
    difficulty,
    algorithm,
    COUNT(*) as count,
    AVG(CASE 
        WHEN status = 'completed' AND solved_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (solved_at - created_at)) * 1000 
        ELSE NULL 
    END)::FLOAT as avg_solve_time_ms
FROM challenges
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY difficulty, algorithm
ORDER BY difficulty, algorithm;

-- name: GetClientStats :many
-- Get statistics per client ID
SELECT 
    client_id,
    COUNT(*) as total_challenges,
    COUNT(*) FILTER (WHERE status = 'completed') as completed_challenges,
    AVG(CASE 
        WHEN status = 'completed' AND solved_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (solved_at - created_at)) * 1000 
        ELSE NULL 
    END)::FLOAT as avg_solve_time_ms,
    MAX(created_at) as last_challenge_time
FROM challenges
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY client_id
ORDER BY total_challenges DESC
LIMIT 100;

-- name: GetHashRateHistory :many
-- Get historical hash rate data for charts (using solutions table)
SELECT 
    time_bucket('5 minutes', s.created_at) as time_bucket,
    COUNT(*) as solution_count,
    SUM(s.attempts) as total_attempts,
    AVG(s.attempts)::FLOAT as avg_attempts_per_solution,
    AVG(c.difficulty)::FLOAT as avg_difficulty
FROM solutions s
JOIN challenges c ON s.challenge_id = c.id
WHERE s.created_at >= NOW() - INTERVAL '24 hours'
GROUP BY time_bucket
ORDER BY time_bucket DESC;