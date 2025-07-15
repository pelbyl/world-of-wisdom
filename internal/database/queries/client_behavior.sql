-- name: GetClientBehaviorByIP :one
SELECT * FROM client_behaviors
WHERE ip_address = $1;

-- name: CreateClientBehavior :one
INSERT INTO client_behaviors (
    ip_address,
    connection_count,
    difficulty,
    last_connection
) VALUES (
    $1, 1, 2, CURRENT_TIMESTAMP
) RETURNING *;

-- name: UpdateClientBehavior :one
UPDATE client_behaviors
SET 
    connection_count = connection_count + 1,
    last_connection = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = $1
RETURNING *;

-- name: UpdateClientChallengeStats :exec
UPDATE client_behaviors
SET 
    total_challenges = total_challenges + 1,
    successful_challenges = CASE WHEN @is_successful::boolean THEN successful_challenges + 1 ELSE successful_challenges END,
    failed_challenges = CASE WHEN NOT @is_successful::boolean THEN failed_challenges + 1 ELSE failed_challenges END,
    total_solve_time_ms = total_solve_time_ms + @solve_time_ms::bigint,
    avg_solve_time_ms = (total_solve_time_ms + @solve_time_ms::bigint) / NULLIF(successful_challenges + CASE WHEN @is_successful::boolean THEN 1 ELSE 0 END, 0),
    failure_rate = failed_challenges::FLOAT / NULLIF(total_challenges + 1, 0),
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = @ip_address;

-- name: UpdateClientDifficulty :exec
UPDATE client_behaviors
SET 
    difficulty = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = $1;

-- name: UpdateClientReconnectRate :exec
UPDATE client_behaviors
SET 
    reconnect_rate = calculate_reconnect_rate(id),
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = $1;

-- name: CalculateAndUpdateClientDifficulty :one
UPDATE client_behaviors
SET 
    difficulty = calculate_adaptive_difficulty(
        failure_rate,
        avg_solve_time_ms,
        reconnect_rate,
        connection_count,
        reputation_score,
        difficulty
    ),
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = $1
RETURNING difficulty;

-- name: UpdateClientReputation :exec
SELECT update_reputation_score(
    (SELECT id FROM client_behaviors WHERE ip_address = @ip_address),
    @challenge_success::boolean
);

-- name: CreateConnectionTimestamp :one
INSERT INTO connection_timestamps (
    client_behavior_id,
    connected_at
) VALUES (
    (SELECT id FROM client_behaviors WHERE ip_address = $1),
    CURRENT_TIMESTAMP
) RETURNING *;

-- name: UpdateConnectionTimestamp :exec
UPDATE connection_timestamps
SET 
    disconnected_at = CURRENT_TIMESTAMP,
    challenge_completed = @challenge_completed::boolean
WHERE id = @id;

-- name: GetActiveClients :many
SELECT 
    cb.*,
    COUNT(c.id) FILTER (WHERE c.status = 'connected') as active_connections
FROM client_behaviors cb
LEFT JOIN connections c ON c.remote_addr = cb.ip_address AND c.status = 'connected'
WHERE cb.last_connection > NOW() - INTERVAL '1 hour'
GROUP BY cb.id
ORDER BY cb.difficulty DESC, cb.connection_count DESC
LIMIT $1;

-- name: GetClientBehaviorStats :many
SELECT 
    cb.*,
    COUNT(ch.id) as recent_challenges,
    AVG(s.solve_time_ms) FILTER (WHERE s.verified = true) as recent_avg_solve_time
FROM client_behaviors cb
LEFT JOIN challenges ch ON ch.client_id IN (
    SELECT client_id FROM connections WHERE remote_addr = cb.ip_address
) AND ch.created_at > NOW() - INTERVAL '10 minutes'
LEFT JOIN solutions s ON s.challenge_id = ch.id
GROUP BY cb.id
ORDER BY cb.suspicious_activity_score DESC
LIMIT $1;

-- name: UpdateSuspiciousActivityScore :exec
UPDATE client_behaviors
SET 
    suspicious_activity_score = CASE
        WHEN failure_rate > 0.8 THEN 90
        WHEN failure_rate > 0.6 AND reconnect_rate > 0.5 THEN 80
        WHEN avg_solve_time_ms < 500 AND avg_solve_time_ms > 0 THEN 85
        WHEN connection_count > 50 AND reputation_score < 30 THEN 75
        WHEN reconnect_rate > 0.7 THEN 70
        ELSE GREATEST(0, suspicious_activity_score - 5) -- Decay over time
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE ip_address = $1;

-- name: GetTopAggressiveClients :many
SELECT 
    ip_address,
    difficulty,
    connection_count,
    failure_rate,
    avg_solve_time_ms,
    reconnect_rate,
    reputation_score,
    suspicious_activity_score,
    last_connection,
    successful_challenges,
    failed_challenges,
    total_challenges
FROM client_behaviors
WHERE suspicious_activity_score > 50
   OR reputation_score < 20
   OR difficulty >= 5
ORDER BY suspicious_activity_score DESC, reputation_score ASC
LIMIT $1;