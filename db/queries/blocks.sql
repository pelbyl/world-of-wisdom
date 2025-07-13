-- name: CreateBlock :one
INSERT INTO blocks (
    block_index, challenge_id, solution_id, quote, previous_hash, block_hash
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetBlock :one
SELECT * FROM blocks WHERE id = $1;

-- name: GetBlockByIndex :one
SELECT * FROM blocks WHERE block_index = $1;

-- name: GetLatestBlock :one
SELECT * FROM blocks 
ORDER BY block_index DESC 
LIMIT 1;

-- name: GetRecentBlocks :many
SELECT b.*, c.difficulty, c.algorithm, s.solve_time_ms 
FROM blocks b
LEFT JOIN challenges c ON b.challenge_id = c.id
LEFT JOIN solutions s ON b.solution_id = s.id
WHERE b.created_at >= NOW() - INTERVAL '1 hour'
ORDER BY b.block_index DESC
LIMIT $1;

-- name: GetBlockchain :many
SELECT b.*, c.difficulty, c.algorithm, s.solve_time_ms 
FROM blocks b
LEFT JOIN challenges c ON b.challenge_id = c.id
LEFT JOIN solutions s ON b.solution_id = s.id
ORDER BY b.block_index ASC;