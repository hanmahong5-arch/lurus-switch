-- PostgreSQL initialization script for Lurus Switch
-- Creates databases and initial schemas for each service

-- Provider Service database (using main database)
CREATE SCHEMA IF NOT EXISTS provider;

-- Billing Service tables
CREATE TABLE IF NOT EXISTS billing_users (
    id VARCHAR(64) PRIMARY KEY,
    email VARCHAR(255),
    name VARCHAR(255),
    plan VARCHAR(32) DEFAULT 'free',
    balance DECIMAL(15,6) DEFAULT 0,
    quota_limit BIGINT DEFAULT 1000000,
    quota_used BIGINT DEFAULT 0,
    quota_reset_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_users_email ON billing_users(email);
CREATE INDEX IF NOT EXISTS idx_billing_users_quota_reset ON billing_users(quota_reset_at);

CREATE TABLE IF NOT EXISTS billing_usage (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(64),
    platform VARCHAR(32),
    model VARCHAR(128),
    provider VARCHAR(128),
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    total_cost DECIMAL(15,6) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_usage_user_id ON billing_usage(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_usage_trace_id ON billing_usage(trace_id);
CREATE INDEX IF NOT EXISTS idx_billing_usage_created_at ON billing_usage(created_at);

-- Provider Service tables
CREATE TABLE IF NOT EXISTS providers (
    id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_url VARCHAR(512) NOT NULL,
    api_key VARCHAR(512),
    platform VARCHAR(32) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    supported_models TEXT[],
    model_mapping JSONB,
    rate_limit INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_providers_platform ON providers(platform);
CREATE INDEX IF NOT EXISTS idx_providers_enabled ON providers(enabled);

-- Sync Service tables (if needed)
CREATE TABLE IF NOT EXISTS sync_sessions (
    id VARCHAR(64) PRIMARY KEY,
    user_id VARCHAR(64) NOT NULL,
    device_id VARCHAR(64),
    platform VARCHAR(32),
    last_sync_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sync_sessions_user_id ON sync_sessions(user_id);

-- Insert some default test data
INSERT INTO billing_users (id, email, name, plan, balance, quota_limit, quota_used, quota_reset_at)
VALUES ('test-user-1', 'test@example.com', 'Test User', 'free', 0, 1000000, 0, NOW() + INTERVAL '30 days')
ON CONFLICT (id) DO NOTHING;

COMMENT ON TABLE billing_users IS 'User billing information';
COMMENT ON TABLE billing_usage IS 'Usage records for billing';
COMMENT ON TABLE providers IS 'AI provider configurations';
COMMENT ON TABLE sync_sessions IS 'User sync sessions';
