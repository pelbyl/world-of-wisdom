-- Add HMAC key storage table
CREATE TABLE IF NOT EXISTS hmac_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_version INTEGER NOT NULL,
    encrypted_key TEXT NOT NULL,
    previous_encrypted_key TEXT,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    rotated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'
);

-- Ensure only one active key at a time
CREATE UNIQUE INDEX idx_hmac_keys_active ON hmac_keys (is_active) WHERE is_active = true;

-- Index for key version lookup
CREATE INDEX idx_hmac_keys_version ON hmac_keys (key_version);

-- Index for time-based queries
CREATE INDEX idx_hmac_keys_created_at ON hmac_keys (created_at);