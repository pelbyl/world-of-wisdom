-- name: CreateSolution :one
INSERT INTO solutions (
    challenge_id, nonce, hash, attempts, solve_time_ms, verified
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetSolution :one
SELECT * FROM solutions WHERE id = $1;

-- name: GetSolutionsByChallenge :many
SELECT * FROM solutions 
WHERE challenge_id = $1
ORDER BY created_at ASC;

-- name: VerifySolution :one
UPDATE solutions 
SET verified = $2
WHERE id = $1 
RETURNING *;

-- name: GetRecentSolutions :many
SELECT s.*, c.difficulty, c.algorithm 
FROM solutions s
JOIN challenges c ON s.challenge_id = c.id
WHERE s.created_at >= NOW() - INTERVAL '1 hour'
ORDER BY s.created_at DESC
LIMIT $1;

-- name: GetSolutionStats :one
SELECT 
    COUNT(*) as total_solutions,
    AVG(solve_time_ms) as avg_solve_time_ms,
    MIN(solve_time_ms) as min_solve_time_ms,
    MAX(solve_time_ms) as max_solve_time_ms,
    COUNT(CASE WHEN verified = true THEN 1 END) as verified_count
FROM solutions 
WHERE created_at >= NOW() - INTERVAL '24 hours';