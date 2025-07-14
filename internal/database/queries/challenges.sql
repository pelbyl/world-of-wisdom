-- name: CreateChallenge :one
INSERT INTO challenges (
    seed, difficulty, algorithm, client_id, status,
    argon2_time, argon2_memory, argon2_threads, argon2_keylen
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetChallenge :one
SELECT * FROM challenges WHERE id = $1;

-- name: GetChallengeByClientID :one
SELECT * FROM challenges 
WHERE client_id = $1 AND status = 'pending'
ORDER BY created_at DESC 
LIMIT 1;

-- name: UpdateChallengeStatus :one
UPDATE challenges 
SET status = @status::challenge_status, solved_at = CASE WHEN @status::challenge_status = 'completed' THEN NOW() ELSE solved_at END
WHERE id = @id 
RETURNING *;

-- name: GetRecentChallenges :many
SELECT * FROM challenges 
WHERE created_at >= NOW() - INTERVAL '1 hour'
ORDER BY created_at DESC
LIMIT $1;

-- name: GetChallengesByDifficulty :many
SELECT * FROM challenges 
WHERE difficulty = $1 AND created_at >= NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- name: GetChallengesByAlgorithm :many
SELECT * FROM challenges 
WHERE algorithm = $1 AND created_at >= NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC;

-- name: GetChallengesFiltered :many
-- Get challenges with multiple filter options for API endpoint
SELECT 
    c.id,
    c.seed,
    c.difficulty,
    c.algorithm,
    c.client_id,
    c.status,
    c.created_at,
    c.solved_at,
    c.expires_at,
    CASE 
        WHEN c.status = 'completed' AND c.solved_at IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (c.solved_at - c.created_at)) * 1000 
        ELSE NULL 
    END::BIGINT as solve_time_ms
FROM challenges c
WHERE 
    (@status::challenge_status IS NULL OR c.status = @status)
    AND (@algorithm::pow_algorithm IS NULL OR c.algorithm = @algorithm)
    AND c.created_at >= NOW() - INTERVAL '24 hours'
ORDER BY c.created_at DESC
LIMIT @limit_count;