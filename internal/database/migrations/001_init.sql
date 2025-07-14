-- Word of Wisdom Database Schema
-- Simple, clean schema for proof-of-work service

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Enum types
CREATE TYPE pow_algorithm AS ENUM ('sha256', 'argon2');
CREATE TYPE challenge_status AS ENUM ('pending', 'solving', 'completed', 'failed', 'expired');
CREATE TYPE connection_status AS ENUM ('connected', 'solving', 'disconnected', 'failed');

-- Core tables
CREATE TABLE challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    seed VARCHAR(64) NOT NULL,
    difficulty INTEGER NOT NULL CHECK (difficulty >= 1 AND difficulty <= 6),
    algorithm pow_algorithm NOT NULL DEFAULT 'argon2',
    client_id VARCHAR(255) NOT NULL,
    status challenge_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    solved_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '5 minutes',
    -- Argon2 specific parameters
    argon2_time INTEGER,
    argon2_memory INTEGER,
    argon2_threads SMALLINT,
    argon2_keylen INTEGER
);

CREATE TABLE solutions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    challenge_id UUID NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    nonce TEXT NOT NULL,
    hash VARCHAR(64),
    attempts INTEGER DEFAULT 1,
    solve_time_ms BIGINT NOT NULL,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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

-- Time-series tables
CREATE TABLE metrics (
    time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    labels JSONB DEFAULT '{}',
    server_instance VARCHAR(100) DEFAULT 'api-server'
);

CREATE TABLE logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(20) NOT NULL CHECK (level IN ('info', 'success', 'warning', 'error')),
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Convert metrics to hypertable for time-series optimization
SELECT create_hypertable('metrics', 'time', if_not_exists => TRUE);

-- Essential indexes only
CREATE INDEX idx_challenges_client_id ON challenges(client_id);
CREATE INDEX idx_challenges_status ON challenges(status);
CREATE INDEX idx_challenges_created_at ON challenges(created_at DESC);
CREATE INDEX idx_solutions_challenge_id ON solutions(challenge_id);
CREATE INDEX idx_connections_client_id ON connections(client_id);
CREATE INDEX idx_connections_connected_at ON connections(connected_at DESC);
CREATE INDEX idx_metrics_time_name ON metrics(time DESC, metric_name);
CREATE INDEX idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX idx_logs_level ON logs(level);

-- Simple retention policy (keep metrics for 30 days)
SELECT add_retention_policy('metrics', INTERVAL '30 days', if_not_exists => TRUE);