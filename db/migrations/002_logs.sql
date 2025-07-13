-- Create logs table for activity logs (time-series data)
CREATE TABLE IF NOT EXISTS logs (
    id UUID DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(20) NOT NULL CHECK (level IN ('info', 'success', 'warning', 'error')),
    message TEXT NOT NULL,
    icon VARCHAR(10),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Convert logs table to hypertable (for time-series optimization)
SELECT create_hypertable('logs', 'timestamp', if_not_exists => TRUE);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_logs_level ON logs(level);
CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at DESC);

-- Create a view for recent logs
CREATE OR REPLACE VIEW recent_logs AS
SELECT 
    id,
    timestamp,
    level,
    message,
    icon,
    metadata
FROM logs
WHERE created_at >= NOW() - INTERVAL '24 hours'
ORDER BY timestamp DESC
LIMIT 1000;

-- Add retention policy for logs (keep 7 days)
SELECT add_retention_policy('logs', INTERVAL '7 days', if_not_exists => TRUE);