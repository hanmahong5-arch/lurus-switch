-- ClickHouse initialization for Lurus Switch Log Service

CREATE DATABASE IF NOT EXISTS lurus_logs;

USE lurus_logs;

-- Request log table (main log storage)
CREATE TABLE IF NOT EXISTS request_log (
    id String,
    trace_id String,
    request_id String,
    user_id String,

    -- Request metadata
    platform LowCardinality(String),
    model LowCardinality(String),
    provider LowCardinality(String),
    provider_model LowCardinality(String),
    request_method LowCardinality(String),
    request_path String,
    is_stream UInt8,
    user_agent String,
    client_ip String,

    -- Response details
    http_code UInt16,
    duration_sec Float32,
    finish_reason LowCardinality(String),

    -- Token usage
    input_tokens UInt32,
    output_tokens UInt32,
    cache_create_tokens UInt32,
    cache_read_tokens UInt32,
    reasoning_tokens UInt32,

    -- Costs (USD)
    input_cost Decimal64(6),
    output_cost Decimal64(6),
    cache_create_cost Decimal64(6),
    cache_read_cost Decimal64(6),
    total_cost Decimal64(6),

    -- Error information
    error_type LowCardinality(String),
    error_message String,
    provider_error_code LowCardinality(String),

    created_at DateTime64(3)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (platform, provider, created_at, trace_id)
TTL created_at + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- Materialized view for hourly stats
CREATE MATERIALIZED VIEW IF NOT EXISTS hourly_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, platform, provider, model)
AS SELECT
    toStartOfHour(created_at) AS hour,
    platform,
    provider,
    model,
    count() AS request_count,
    countIf(http_code >= 200 AND http_code < 400) AS success_count,
    sum(input_tokens + output_tokens) AS total_tokens,
    sum(total_cost) AS total_cost,
    avg(duration_sec * 1000) AS avg_duration_ms
FROM request_log
GROUP BY hour, platform, provider, model;

-- Materialized view for daily stats
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, platform, provider, model)
AS SELECT
    toDate(created_at) AS date,
    platform,
    provider,
    model,
    count() AS request_count,
    countIf(http_code >= 200 AND http_code < 400) AS success_count,
    sum(input_tokens + output_tokens) AS total_tokens,
    sum(total_cost) AS total_cost,
    avg(duration_sec * 1000) AS avg_duration_ms
FROM request_log
GROUP BY date, platform, provider, model;

-- Materialized view for user daily stats
CREATE MATERIALIZED VIEW IF NOT EXISTS user_daily_stats_mv
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, user_id, platform)
AS SELECT
    toDate(created_at) AS date,
    user_id,
    platform,
    count() AS request_count,
    sum(input_tokens + output_tokens) AS total_tokens,
    sum(total_cost) AS total_cost
FROM request_log
GROUP BY date, user_id, platform;

-- Create indexes
ALTER TABLE request_log ADD INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 4;
ALTER TABLE request_log ADD INDEX idx_user_id user_id TYPE bloom_filter GRANULARITY 4;
ALTER TABLE request_log ADD INDEX idx_error_type error_type TYPE set(100) GRANULARITY 4;
