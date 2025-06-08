-- Create audit logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id VARCHAR(50) PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    user_id VARCHAR(100),
    user_email VARCHAR(255),
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(100),
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    ip_address INET NOT NULL,
    user_agent TEXT,
    request_body JSONB,
    response_body JSONB,
    error TEXT,
    duration BIGINT NOT NULL, -- Duration in nanoseconds
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_status_code ON audit_logs(status_code);

-- Composite indexes for filtering
CREATE INDEX idx_audit_logs_user_timestamp ON audit_logs(user_id, timestamp DESC);
CREATE INDEX idx_audit_logs_resource_timestamp ON audit_logs(resource_type, resource_id, timestamp DESC);

-- Partition by month for better performance at scale
-- This is commented out by default, enable if you expect high volume
-- CREATE TABLE audit_logs_y2024m01 PARTITION OF audit_logs 
--     FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Add comments
COMMENT ON TABLE audit_logs IS 'Audit trail of all API operations';
COMMENT ON COLUMN audit_logs.duration IS 'Request duration in nanoseconds';
COMMENT ON COLUMN audit_logs.metadata IS 'Additional context like request_id, trace_id, etc';