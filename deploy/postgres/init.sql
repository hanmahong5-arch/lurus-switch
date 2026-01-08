-- PostgreSQL initialization for Lurus Switch

-- Create databases for each service
CREATE DATABASE IF NOT EXISTS provider_service;
CREATE DATABASE IF NOT EXISTS billing_service;
CREATE DATABASE IF NOT EXISTS sync_service;

-- Connect to provider_service database
\c provider_service;

-- Provider table
CREATE TABLE IF NOT EXISTS providers (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    api_url VARCHAR(500) NOT NULL,
    api_key VARCHAR(500) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    level INTEGER DEFAULT 1,
    site VARCHAR(255),
    icon VARCHAR(255),
    tint VARCHAR(50),
    accent VARCHAR(50),
    supported_models JSONB DEFAULT '{}',
    model_mapping JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_providers_platform ON providers(platform);
CREATE INDEX idx_providers_enabled ON providers(enabled);

-- Connect to billing_service database
\c billing_service;

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(100) PRIMARY KEY,
    newapi_user_id VARCHAR(100),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    avatar_url VARCHAR(500),
    plan VARCHAR(50) DEFAULT 'free',
    quota_total DECIMAL(20, 6) DEFAULT 0,
    quota_used DECIMAL(20, 6) DEFAULT 0,
    daily_limit DECIMAL(20, 6),
    daily_used DECIMAL(20, 6) DEFAULT 0,
    is_admin BOOLEAN DEFAULT false,
    is_disabled BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL REFERENCES users(id),
    type VARCHAR(50) NOT NULL,
    amount DECIMAL(20, 6) NOT NULL,
    balance_before DECIMAL(20, 6) NOT NULL,
    balance_after DECIMAL(20, 6) NOT NULL,
    description TEXT,
    trace_id VARCHAR(100),
    platform VARCHAR(50),
    model VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_trace_id ON transactions(trace_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);

-- Connect to sync_service database
\c sync_service;

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    title VARCHAR(500) NOT NULL,
    summary TEXT,
    model VARCHAR(100),
    provider VARCHAR(100),
    message_count INTEGER DEFAULT 0,
    token_count INTEGER DEFAULT 0,
    cost DECIMAL(20, 6) DEFAULT 0,
    is_pinned BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    last_message_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_last_message_at ON sessions(last_message_at);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR(100) PRIMARY KEY,
    session_id VARCHAR(100) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(100) NOT NULL,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    content_type VARCHAR(50) DEFAULT 'text',
    model VARCHAR(100),
    provider VARCHAR(100),
    tokens_input INTEGER DEFAULT 0,
    tokens_output INTEGER DEFAULT 0,
    tokens_reasoning INTEGER DEFAULT 0,
    cost DECIMAL(20, 6) DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,
    finish_reason VARCHAR(50),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_session_id ON messages(session_id);
CREATE INDEX idx_messages_user_id ON messages(user_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    device_id VARCHAR(255) NOT NULL,
    device_name VARCHAR(255),
    device_type VARCHAR(50),
    client_version VARCHAR(50),
    last_seen_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_ip VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_devices_user_id ON devices(user_id);
CREATE UNIQUE INDEX idx_devices_user_device ON devices(user_id, device_id);
