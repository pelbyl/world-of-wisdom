-- Create logs table for activity logs
CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    level VARCHAR(20) NOT NULL CHECK (level IN ('info', 'success', 'warning', 'error')),
    message TEXT NOT NULL,
    icon VARCHAR(10),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for efficient querying
CREATE INDEX idx_logs_timestamp ON logs(timestamp DESC);
CREATE INDEX idx_logs_level ON logs(level);
CREATE INDEX idx_logs_created_at ON logs(created_at DESC);

-- Create a view for recent logs
CREATE VIEW recent_logs AS
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
SELECT add_retention_policy('logs', INTERVAL '7 days');