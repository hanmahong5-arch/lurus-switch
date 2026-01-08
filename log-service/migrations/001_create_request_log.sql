-- Create request_log table for ClickHouse
-- This table stores all LLM request logs for analysis

CREATE TABLE IF NOT EXISTS request_log (
    -- Primary identifiers
    id String,
    trace_id String,
    request_id String,
    user_id String,

    -- Request metadata
    platform LowCardinality(String),      -- claude, codex, gemini
    model LowCardinality(String),         -- requested model name
    provider LowCardinality(String),      -- provider used
    provider_model LowCardinality(String), -- actual model used at provider

    -- HTTP details
    request_method LowCardinality(String),
    request_path String,
    is_stream UInt8,                      -- 0: false, 1: true
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

    -- Cost (in USD with 6 decimal precision)
    input_cost Decimal64(6),
    output_cost Decimal64(6),
    cache_create_cost Decimal64(6),
    cache_read_cost Decimal64(6),
    total_cost Decimal64(6),

    -- Error information
    error_type LowCardinality(String),
    error_message String,
    provider_error_code String,

    -- Timestamp
    created_at DateTime64(3)
)
ENGINE = MergeTree()
PARTITION BY toYYYYMM(created_at)
ORDER BY (platform, provider, created_at, trace_id)
TTL created_at + INTERVAL 365 DAY
SETTINGS index_granularity = 8192;

-- Create indexes for common queries
ALTER TABLE request_log ADD INDEX idx_user_id user_id TYPE bloom_filter GRANULARITY 1;
ALTER TABLE request_log ADD INDEX idx_trace_id trace_id TYPE bloom_filter GRANULARITY 1;
ALTER TABLE request_log ADD INDEX idx_model model TYPE bloom_filter GRANULARITY 1;

-- Materialized view for hourly stats (pre-aggregated)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_hourly_stats
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
    sum(duration_sec) AS total_duration_sec
FROM request_log
GROUP BY hour, platform, provider, model;

-- Materialized view for daily stats (pre-aggregated)
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_stats
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
    sum(duration_sec) AS total_duration_sec
FROM request_log
GROUP BY date, platform, provider, model;
