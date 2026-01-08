-- ClickHouse initialization script for Lurus Switch Log Service
-- OLAP database for request log analytics

-- Create database
CREATE DATABASE IF NOT EXISTS lurus_logs;

-- Request log table with MergeTree engine for high-performance analytics
CREATE TABLE IF NOT EXISTS lurus_logs.request_log (
    trace_id String,
    user_id String,
    platform LowCardinality(String),
    model LowCardinality(String),
    provider LowCardinality(String),
    input_tokens UInt32,
    output_tokens UInt32,
    total_tokens UInt32 MATERIALIZED input_tokens + output_tokens,
    total_cost Decimal64(6),
    duration_ms UInt32,
    status_code UInt16,
    error_type LowCardinality(String),
    is_stream UInt8,
    client_ip String,
    user_agent String,
    created_at DateTime64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (platform, provider, model, created_at)
TTL created_at + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- Daily aggregation table for faster dashboard queries
CREATE TABLE IF NOT EXISTS lurus_logs.daily_stats (
    date Date,
    user_id String,
    platform LowCardinality(String),
    model LowCardinality(String),
    provider LowCardinality(String),
    request_count UInt64,
    total_input_tokens UInt64,
    total_output_tokens UInt64,
    total_cost Decimal64(6),
    avg_duration_ms Float64,
    error_count UInt64
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, user_id, platform, model, provider)
TTL date + INTERVAL 730 DAY;

-- Materialized view to auto-populate daily stats
CREATE MATERIALIZED VIEW IF NOT EXISTS lurus_logs.mv_daily_stats
TO lurus_logs.daily_stats
AS SELECT
    toDate(created_at) as date,
    user_id,
    platform,
    model,
    provider,
    count() as request_count,
    sum(input_tokens) as total_input_tokens,
    sum(output_tokens) as total_output_tokens,
    sum(total_cost) as total_cost,
    avg(duration_ms) as avg_duration_ms,
    countIf(status_code >= 400) as error_count
FROM lurus_logs.request_log
GROUP BY date, user_id, platform, model, provider;

-- Hourly stats for real-time monitoring
CREATE TABLE IF NOT EXISTS lurus_logs.hourly_stats (
    hour DateTime,
    platform LowCardinality(String),
    provider LowCardinality(String),
    request_count UInt64,
    error_count UInt64,
    avg_duration_ms Float64,
    p95_duration_ms Float64,
    total_tokens UInt64
)
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(hour)
ORDER BY (hour, platform, provider)
TTL hour + INTERVAL 30 DAY;

-- Model usage ranking view
CREATE VIEW IF NOT EXISTS lurus_logs.model_ranking AS
SELECT
    model,
    platform,
    count() as usage_count,
    sum(input_tokens + output_tokens) as total_tokens,
    sum(total_cost) as total_cost,
    avg(duration_ms) as avg_latency_ms
FROM lurus_logs.request_log
WHERE created_at >= now() - INTERVAL 7 DAY
GROUP BY model, platform
ORDER BY usage_count DESC;

-- Provider performance view
CREATE VIEW IF NOT EXISTS lurus_logs.provider_performance AS
SELECT
    provider,
    platform,
    count() as request_count,
    countIf(status_code >= 400) / count() * 100 as error_rate,
    avg(duration_ms) as avg_latency_ms,
    quantile(0.95)(duration_ms) as p95_latency_ms,
    quantile(0.99)(duration_ms) as p99_latency_ms
FROM lurus_logs.request_log
WHERE created_at >= now() - INTERVAL 24 HOUR
GROUP BY provider, platform
ORDER BY request_count DESC;

-- User usage summary view
CREATE VIEW IF NOT EXISTS lurus_logs.user_usage AS
SELECT
    user_id,
    count() as request_count,
    sum(input_tokens) as input_tokens,
    sum(output_tokens) as output_tokens,
    sum(total_cost) as total_cost,
    min(created_at) as first_request,
    max(created_at) as last_request
FROM lurus_logs.request_log
WHERE created_at >= now() - INTERVAL 30 DAY
GROUP BY user_id
ORDER BY total_cost DESC;

-- Create index for trace_id lookups
ALTER TABLE lurus_logs.request_log
    ADD INDEX idx_trace_id trace_id TYPE bloom_filter(0.01) GRANULARITY 4;

ALTER TABLE lurus_logs.request_log
    ADD INDEX idx_user_id user_id TYPE bloom_filter(0.01) GRANULARITY 4;
