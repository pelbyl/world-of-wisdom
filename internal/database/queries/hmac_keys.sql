-- name: GetActiveHMACKey :one
SELECT * FROM hmac_keys
WHERE is_active = true
LIMIT 1;

-- name: CreateHMACKey :one
INSERT INTO hmac_keys (
    key_version,
    encrypted_key,
    previous_encrypted_key,
    metadata
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: DeactivateHMACKeys :exec
UPDATE hmac_keys
SET is_active = false
WHERE is_active = true;

-- name: GetHMACKeyByVersion :one
SELECT * FROM hmac_keys
WHERE key_version = $1
LIMIT 1;

-- name: GetLatestHMACKeys :many
SELECT * FROM hmac_keys
ORDER BY created_at DESC
LIMIT $1;