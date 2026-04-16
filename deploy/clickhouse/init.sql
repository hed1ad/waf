-- ClickHouse schema for WAF events
-- Engine: MergeTree partitioned by day, sorted by timestamp + client_ip

CREATE DATABASE IF NOT EXISTS waf;

CREATE TABLE IF NOT EXISTS waf.events
(
    -- When
    timestamp          DateTime64(3, 'UTC'),
    -- Transaction identity
    transaction_id     String,
    -- Client
    client_ip          String,
    client_port        UInt16,
    -- Request
    method             LowCardinality(String),
    uri                String,
    protocol           LowCardinality(String),
    host               String,
    -- Response
    status             UInt16,
    -- ModSecurity decision
    action             LowCardinality(String),  -- 'pass' | 'block' | 'redirect' | 'allow'
    anomaly_score      UInt16,
    -- Matched rules (stored as parallel arrays for fast querying)
    rule_ids           Array(UInt32),
    rule_msgs          Array(String),
    rule_tags          Array(String),
    rule_severities    Array(LowCardinality(String)),
    -- GeoIP enrichment (filled by ingester)
    country_code       LowCardinality(String),
    country_name       LowCardinality(String),
    city               String,
    asn                UInt32,
    asn_org            String,
    -- Raw audit log parts (for drill-down)
    request_headers    String,
    request_body       String,
    response_headers   String,
    -- Metadata
    ingested_at        DateTime DEFAULT now()
)
ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (timestamp, client_ip, transaction_id)
TTL toDateTime(timestamp) + INTERVAL 90 DAY
SETTINGS index_granularity = 8192;

-- Materialized view: per-minute aggregates for the dashboard timeseries chart
CREATE TABLE IF NOT EXISTS waf.events_1m
(
    minute             DateTime,
    action             LowCardinality(String),
    total              UInt64,
    blocked            UInt64,
    unique_ips         UInt64
)
ENGINE = SummingMergeTree((total, blocked))
PARTITION BY toYYYYMM(minute)
ORDER BY (minute, action);

CREATE MATERIALIZED VIEW IF NOT EXISTS waf.events_1m_mv
TO waf.events_1m AS
SELECT
    toStartOfMinute(timestamp) AS minute,
    action,
    count()                    AS total,
    countIf(action = 'block')  AS blocked,
    uniq(client_ip)            AS unique_ips
FROM waf.events
GROUP BY minute, action;

-- Materialized view: top attacking IPs (rolling, deduplicated by ReplacingMergeTree)
CREATE TABLE IF NOT EXISTS waf.top_ips
(
    client_ip     String,
    country_code  LowCardinality(String),
    total_hits    UInt64,
    blocked_hits  UInt64,
    last_seen     DateTime
)
ENGINE = ReplacingMergeTree(last_seen)
ORDER BY client_ip;

CREATE MATERIALIZED VIEW IF NOT EXISTS waf.top_ips_mv
TO waf.top_ips AS
SELECT
    client_ip,
    anyLast(country_code)       AS country_code,
    count()                     AS total_hits,
    countIf(action = 'block')   AS blocked_hits,
    max(timestamp)              AS last_seen
FROM waf.events
GROUP BY client_ip;
