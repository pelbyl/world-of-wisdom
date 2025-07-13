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
SET status = $2, solved_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE solved_at END
WHERE id = $1 
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