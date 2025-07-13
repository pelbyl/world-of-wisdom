-- Clear all application data while preserving schema
-- This script truncates all data tables but keeps the database structure intact

-- Disable foreign key constraints temporarily for faster truncation
SET session_replication_role = replica;

-- Clear all data tables (order matters due to foreign key constraints)
TRUNCATE TABLE blocks RESTART IDENTITY CASCADE;
TRUNCATE TABLE solutions RESTART IDENTITY CASCADE;  
TRUNCATE TABLE challenges RESTART IDENTITY CASCADE;
TRUNCATE TABLE connections RESTART IDENTITY CASCADE;
TRUNCATE TABLE metrics RESTART IDENTITY CASCADE;

-- Re-enable foreign key constraints
SET session_replication_role = DEFAULT;

-- Reset any sequences
ALTER SEQUENCE blocks_id_seq RESTART WITH 1;

-- Add initial server started metric
INSERT INTO metrics (metric_name, metric_value, labels) VALUES
('server_started', 1, '{"version": "1.0.0", "algorithm": "argon2", "cleaned": true}');