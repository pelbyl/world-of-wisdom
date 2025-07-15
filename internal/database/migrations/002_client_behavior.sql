-- Add client behavior tracking table
CREATE TABLE IF NOT EXISTS client_behaviors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ip_address INET NOT NULL UNIQUE,
    connection_count INTEGER DEFAULT 0,
    failure_rate FLOAT DEFAULT 0.0,
    avg_solve_time_ms BIGINT DEFAULT 0,
    last_connection TIMESTAMPTZ,
    reconnect_rate FLOAT DEFAULT 0.0,
    difficulty INTEGER DEFAULT 1,
    total_challenges INTEGER DEFAULT 0,
    successful_challenges INTEGER DEFAULT 0,
    failed_challenges INTEGER DEFAULT 0,
    total_solve_time_ms BIGINT DEFAULT 0,
    suspicious_activity_score FLOAT DEFAULT 0.0,
    reputation_score FLOAT DEFAULT 50.0, -- 0-100, starts at neutral
    last_reputation_update TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient lookups
CREATE INDEX idx_client_behaviors_ip ON client_behaviors(ip_address);
CREATE INDEX idx_client_behaviors_difficulty ON client_behaviors(difficulty);
CREATE INDEX idx_client_behaviors_reputation ON client_behaviors(reputation_score);
CREATE INDEX idx_client_behaviors_last_connection ON client_behaviors(last_connection);

-- Add trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_client_behaviors_updated_at BEFORE UPDATE
    ON client_behaviors FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Add connection timestamps table for tracking reconnect patterns
CREATE TABLE IF NOT EXISTS connection_timestamps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_behavior_id UUID NOT NULL REFERENCES client_behaviors(id) ON DELETE CASCADE,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    disconnected_at TIMESTAMPTZ,
    challenge_completed BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_connection_timestamps_client ON connection_timestamps(client_behavior_id);
CREATE INDEX idx_connection_timestamps_connected ON connection_timestamps(connected_at);

-- Function to calculate reconnect rate
CREATE OR REPLACE FUNCTION calculate_reconnect_rate(p_client_behavior_id UUID)
RETURNS FLOAT AS $$
DECLARE
    v_reconnect_count INTEGER;
    v_total_connections INTEGER;
    v_avg_interval INTERVAL;
BEGIN
    -- Count rapid reconnects (within 5 seconds of disconnect)
    SELECT COUNT(*) INTO v_reconnect_count
    FROM connection_timestamps t1
    JOIN connection_timestamps t2 ON t1.client_behavior_id = t2.client_behavior_id
    WHERE t1.client_behavior_id = p_client_behavior_id
    AND t2.connected_at > t1.disconnected_at
    AND t2.connected_at - t1.disconnected_at < INTERVAL '5 seconds';
    
    SELECT COUNT(*) INTO v_total_connections
    FROM connection_timestamps
    WHERE client_behavior_id = p_client_behavior_id;
    
    IF v_total_connections <= 1 THEN
        RETURN 0.0;
    END IF;
    
    RETURN v_reconnect_count::FLOAT / v_total_connections::FLOAT;
END;
$$ LANGUAGE plpgsql;

-- Function to update client difficulty based on behavior
CREATE OR REPLACE FUNCTION calculate_adaptive_difficulty(
    p_failure_rate FLOAT,
    p_avg_solve_time_ms BIGINT,
    p_reconnect_rate FLOAT,
    p_connection_count INTEGER,
    p_reputation_score FLOAT,
    p_current_difficulty INTEGER
) RETURNS INTEGER AS $$
DECLARE
    v_difficulty INTEGER;
    v_difficulty_adjustment INTEGER := 0;
BEGIN
    -- Start with current difficulty
    v_difficulty := p_current_difficulty;
    
    -- High failure rate increases difficulty
    IF p_failure_rate > 0.5 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 2;
    ELSIF p_failure_rate > 0.3 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 1;
    END IF;
    
    -- Primary goal: Ensure solve times stay in 10-30 second range
    -- Decrease difficulty if solving takes too long (prioritize giving wisdom)
    IF p_avg_solve_time_ms > 30000 THEN
        v_difficulty_adjustment := v_difficulty_adjustment - 3; -- Way too slow, help them out
    ELSIF p_avg_solve_time_ms > 20000 THEN
        v_difficulty_adjustment := v_difficulty_adjustment - 2; -- Too slow, make easier
    ELSIF p_avg_solve_time_ms > 15000 THEN
        v_difficulty_adjustment := v_difficulty_adjustment - 1; -- Slightly slow
    END IF;
    
    -- Gradually increase difficulty only after multiple successes (anti-spam)
    IF p_connection_count >= 10 AND p_failure_rate <= 0.1 AND p_avg_solve_time_ms < 10000 THEN
        -- Many successful, fast solves = increase difficulty gradually
        v_difficulty_adjustment := v_difficulty_adjustment + 1;
    ELSIF p_connection_count >= 20 AND p_failure_rate <= 0.2 THEN
        -- High activity with good success = potential spam
        v_difficulty_adjustment := v_difficulty_adjustment + 1;
    END IF;
    
    -- Bot detection (only extreme cases)
    IF p_avg_solve_time_ms > 0 AND p_avg_solve_time_ms < 100 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 3; -- Extremely fast = clear bot
    ELSIF p_avg_solve_time_ms < 1000 AND p_connection_count > 50 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 2; -- Fast + high volume = bot
    END IF;
    
    -- Massive spam detection
    IF p_connection_count > 100 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 2; -- Clear spam
    ELSIF p_reconnect_rate > 0.8 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 2; -- Rapid reconnect spam
    END IF;
    
    -- Reward good behavior patterns
    IF p_connection_count >= 3 AND p_avg_solve_time_ms >= 10000 AND p_avg_solve_time_ms <= 30000 THEN
        -- Perfect solve time range = reward with easier difficulty
        v_difficulty_adjustment := v_difficulty_adjustment - 1;
    END IF;
    
    -- Reputation adjustments - be generous
    IF p_reputation_score < 10 THEN
        v_difficulty_adjustment := v_difficulty_adjustment + 1; -- Only very bad reputation
    ELSIF p_reputation_score > 80 THEN
        v_difficulty_adjustment := v_difficulty_adjustment - 1; -- Good reputation gets bonus
    END IF;
    
    -- Apply adjustment
    v_difficulty := v_difficulty + v_difficulty_adjustment;
    
    -- Clamp to valid range
    IF v_difficulty < 1 THEN
        v_difficulty := 1;
    ELSIF v_difficulty > 6 THEN
        v_difficulty := 6;
    END IF;
    
    RETURN v_difficulty;
END;
$$ LANGUAGE plpgsql;

-- Function to decay reputation over time (good behavior improves reputation)
CREATE OR REPLACE FUNCTION update_reputation_score(
    p_client_behavior_id UUID,
    p_challenge_success BOOLEAN
) RETURNS VOID AS $$
DECLARE
    v_current_reputation FLOAT;
    v_last_update TIMESTAMPTZ;
    v_hours_since_update FLOAT;
    v_new_reputation FLOAT;
BEGIN
    SELECT reputation_score, last_reputation_update 
    INTO v_current_reputation, v_last_update
    FROM client_behaviors
    WHERE id = p_client_behavior_id;
    
    -- Calculate hours since last update
    v_hours_since_update := EXTRACT(EPOCH FROM (NOW() - v_last_update)) / 3600.0;
    
    -- Start with current reputation
    v_new_reputation := v_current_reputation;
    
    -- Natural reputation recovery over time (1 point per hour, max 50)
    IF v_current_reputation < 50 THEN
        v_new_reputation := LEAST(50, v_current_reputation + v_hours_since_update);
    END IF;
    
    -- Adjust based on challenge result
    IF p_challenge_success THEN
        -- Successful challenge improves reputation
        v_new_reputation := LEAST(100, v_new_reputation + 5);
    ELSE
        -- Failed challenge decreases reputation
        v_new_reputation := GREATEST(0, v_new_reputation - 10);
    END IF;
    
    -- Update the reputation
    UPDATE client_behaviors
    SET reputation_score = v_new_reputation,
        last_reputation_update = NOW()
    WHERE id = p_client_behavior_id;
END;
$$ LANGUAGE plpgsql;