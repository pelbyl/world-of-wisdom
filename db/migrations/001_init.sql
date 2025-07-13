-- Enable TimescaleDB extension
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

-- Create enum types
CREATE TYPE pow_algorithm AS ENUM ('sha256', 'argon2');
CREATE TYPE challenge_status AS ENUM ('pending', 'solving', 'completed', 'failed', 'expired');
CREATE TYPE connection_status AS ENUM ('connected', 'solving', 'disconnected', 'failed');

-- Challenges table
CREATE TABLE challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seed VARCHAR(64) NOT NULL,
    difficulty INTEGER NOT NULL CHECK (difficulty >= 1 AND difficulty <= 6),
    algorithm pow_algorithm NOT NULL DEFAULT 'argon2',
    client_id VARCHAR(255) NOT NULL,
    status challenge_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    solved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '5 minutes',
    
    -- Argon2 parameters
    argon2_time INTEGER,
    argon2_memory INTEGER,
    argon2_threads SMALLINT,
    argon2_keylen INTEGER
);

-- Solutions table
CREATE TABLE solutions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    nonce TEXT NOT NULL,
    hash VARCHAR(64) NOT NULL,
    attempts INTEGER DEFAULT 1,
    solve_time_ms BIGINT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Connections table for tracking client connections
CREATE TABLE connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(255) NOT NULL,
    remote_addr INET NOT NULL,
    status connection_status NOT NULL DEFAULT 'connected',
    algorithm pow_algorithm NOT NULL DEFAULT 'argon2',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    challenges_attempted INTEGER DEFAULT 0,
    challenges_completed INTEGER DEFAULT 0,
    total_solve_time_ms BIGINT DEFAULT 0
);

-- Blocks table for blockchain-like storage
CREATE TABLE blocks (
    id SERIAL PRIMARY KEY,
    block_index INTEGER NOT NULL,
    challenge_id UUID REFERENCES challenges(id),
    solution_id UUID REFERENCES solutions(id),
    quote TEXT,
    previous_hash VARCHAR(64),
    block_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Metrics table (hypertable for time-series data)
CREATE TABLE metrics (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    labels JSONB DEFAULT '{}',
    server_instance VARCHAR(100) DEFAULT 'default'
);

-- Convert metrics table to hypertable
SELECT create_hypertable('metrics', 'time');

-- Create indexes
CREATE INDEX idx_challenges_client_id ON challenges(client_id);
CREATE INDEX idx_challenges_status ON challenges(status);
CREATE INDEX idx_challenges_created_at ON challenges(created_at);
CREATE INDEX idx_challenges_algorithm ON challenges(algorithm);

CREATE INDEX idx_solutions_challenge_id ON solutions(challenge_id);
CREATE INDEX idx_solutions_verified ON solutions(verified);
CREATE INDEX idx_solutions_solve_time ON solutions(solve_time_ms);

CREATE INDEX idx_connections_client_id ON connections(client_id);
CREATE INDEX idx_connections_status ON connections(status);
CREATE INDEX idx_connections_connected_at ON connections(connected_at);

CREATE INDEX idx_blocks_challenge_id ON blocks(challenge_id);
CREATE INDEX idx_blocks_created_at ON blocks(created_at);

CREATE INDEX idx_metrics_time_name ON metrics(time, metric_name);
CREATE INDEX idx_metrics_labels ON metrics USING GIN(labels);

-- Create views for common queries
CREATE VIEW challenge_stats AS
SELECT 
    algorithm,
    difficulty,
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (solved_at - created_at)) * 1000) as avg_solve_time_ms
FROM challenges 
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY algorithm, difficulty, status;

CREATE VIEW connection_stats AS
SELECT 
    algorithm,
    status,
    COUNT(*) as count,
    AVG(challenges_completed) as avg_challenges_completed,
    AVG(total_solve_time_ms) as avg_total_solve_time_ms
FROM connections 
WHERE connected_at >= NOW() - INTERVAL '24 hours'
GROUP BY algorithm, status;

-- Functions for metrics
CREATE OR REPLACE FUNCTION record_metric(
    p_metric_name VARCHAR(100),
    p_metric_value DOUBLE PRECISION,
    p_labels JSONB DEFAULT '{}'
) RETURNS VOID AS $$
BEGIN
    INSERT INTO metrics (metric_name, metric_value, labels)
    VALUES (p_metric_name, p_metric_value, p_labels);
END;
$$ LANGUAGE plpgsql;

-- Retention policy for metrics (keep 30 days)
SELECT add_retention_policy('metrics', INTERVAL '30 days');

-- Initial data
INSERT INTO metrics (metric_name, metric_value, labels) VALUES
('server_started', 1, '{"version": "1.0.0", "algorithm": "argon2"}');