-- ========================================
-- Migration: Affiliate Link Tracking System
-- Date: 2025-10-29
-- Feature: specs/003-affiliate-link-tracking
-- ========================================
--
-- This migration implements:
-- 1. TimescaleDB extension enablement
-- 2. kpi_metrics hypertable conversion
-- 3. New tables: affiliate_links, click_events (hypertable)
-- 4. Enum extension: reference_type + 'AFFILIATE_LINK'
-- 5. Continuous aggregate: click_events_hourly
-- 6. Admin config: CTR aggregation settings
--
-- CRITICAL: Run backup_kpi_metrics.sh before applying this migration!
-- ========================================
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;
CREATE EXTENSION IF NOT EXISTS pgcrypto CASCADE;

begin
;

-- ========================================
-- STEP 1: Enable TimescaleDB Extension
-- ========================================
-- This must be done by a superuser
-- Verify with: SELECT extname, extversion FROM pg_extension WHERE extname =
-- 'timescaledb';
-- ========================================
-- STEP 2: Convert kpi_metrics to Hypertable
-- ========================================
-- IMPORTANT: This requires kpi_metrics to have:
-- 1. Composite primary key: (id, recorded_date)
-- 2. recorded_date must be NOT NULL
--
-- If kpi_metrics doesn't have composite PK, we need to alter it first:
-- Check current structure
DO
$$
    BEGIN
        -- Check if kpi_metrics already has composite primary key
        IF NOT EXISTS (SELECT 1
                       FROM pg_constraint
                       WHERE conname = 'kpi_metrics_pkey'
                         AND conrelid = 'kpi_metrics'::regclass
                         AND array_length(conkey, 1) = 2) THEN
            -- Drop existing primary key
            RAISE NOTICE 'Altering kpi_metrics table to add composite primary key';
            ALTER TABLE kpi_metrics
                DROP CONSTRAINT IF EXISTS kpi_metrics_pkey;

            -- Add composite primary key
            ALTER TABLE kpi_metrics
                ADD PRIMARY KEY (id, recorded_date);

            RAISE NOTICE 'Composite primary key added successfully';
        ELSE
            RAISE NOTICE 'kpi_metrics already has composite primary key - skipping alteration';
        END IF;
    END
$$;

-- Ensure recorded_date is NOT NULL
ALTER TABLE kpi_metrics
    ALTER COLUMN recorded_date SET NOT NULL;

-- ========================================
-- STEP 3: Extend reference_type Enum
-- ========================================
-- Add AFFILIATE_LINK to existing reference_type enum
-- Check if value already exists (PostgreSQL 9.6+)
DO
$$
    DECLARE
        _constraint_name text;
    BEGIN
        -- STEP 1: Create new ENUM type if not exists
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'reference_type') THEN
            CREATE TYPE reference_type AS ENUM ('CONTENT', 'CAMPAIGN', 'AFFILIATE_LINK');
            RAISE NOTICE '✅ Created new ENUM type reference_type';
        ELSE
            RAISE NOTICE 'reference_type enum already exists';
        END IF;

        -- STEP 2: Find and drop the old check constraint if it exists
        alter table kpi_metrics
            drop constraint kpi_metrics_reference_type_check;

        -- STEP 3: Alter column to ENUM type using safe cast
        ALTER TABLE kpi_metrics
            ALTER COLUMN reference_type TYPE reference_type
                USING reference_type::reference_type;

        RAISE NOTICE '✅ Converted kpi_metrics.reference_type to enum type successfully';
    END
$$;


-- Convert to hypertable (skip if already a hypertable)
select
    create_hypertable(
        'kpi_metrics',
        'recorded_date',
        chunk_time_interval => interval '7 days',
        if_not_exists => true,
        migrate_data => true
    )
;

ALTER TABLE kpi_metrics
    SET (timescaledb.compress = true);

ALTER TABLE kpi_metrics
    SET (
        timescaledb.compress_orderby = 'recorded_date DESC',
        timescaledb.compress_segmentby = 'id'
        );

-- Add compression policy (compress chunks older than 7 days)
select add_compression_policy('kpi_metrics', interval '7 days', if_not_exists => true)
;

-- Add retention policy (drop chunks older than 1 year)
select add_retention_policy('kpi_metrics', interval '365 days', if_not_exists => true)
;

-- Verify conversion
DO
$$
    DECLARE
        hypertable_count INTEGER;
    BEGIN
        SELECT COUNT(*)
        INTO hypertable_count
        FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'kpi_metrics';

        IF hypertable_count = 0 THEN
            RAISE EXCEPTION 'kpi_metrics hypertable conversion failed';
        ELSE
            RAISE NOTICE '✅ kpi_metrics successfully converted to hypertable';
        END IF;
    END
$$;

-- ========================================
-- STEP 4: Create affiliate_links Table
-- ========================================
CREATE TABLE IF NOT EXISTS affiliate_links
(
    id           UUID PRIMARY KEY            DEFAULT gen_random_uuid(),
    hash         VARCHAR(16) UNIQUE NOT NULL,
    contract_id  UUID               NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    content_id   UUID               NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    channel_id   UUID               NOT NULL REFERENCES channels (id) ON DELETE RESTRICT,
    tracking_url TEXT               NOT NULL,
    status       VARCHAR(20)        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'expired')),
    created_at   TIMESTAMPTZ                 DEFAULT NOW(),
    updated_at   TIMESTAMPTZ                 DEFAULT NOW(),
    deleted_at   TIMESTAMPTZ,

-- Unique constraint: one affiliate link per content+channel+contract combination
    CONSTRAINT unique_affiliate_combination UNIQUE (contract_id, content_id, channel_id)
);

-- Indexes for affiliate_links
CREATE INDEX IF NOT EXISTS idx_affiliate_links_contract_id ON affiliate_links (contract_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_affiliate_links_content_id ON affiliate_links (content_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_affiliate_links_channel_id ON affiliate_links (channel_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_affiliate_links_status ON affiliate_links (status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_affiliate_links_deleted_at ON affiliate_links (deleted_at);

-- Comments for documentation
COMMENT ON TABLE affiliate_links IS 'Stores unique trackable affiliate links for content+channel combinations';
COMMENT ON COLUMN affiliate_links.hash IS 'Base62 SHA-256 hash (16 chars) for public URL shortening';
COMMENT ON COLUMN affiliate_links.tracking_url IS 'Original affiliate product URL from contract ScopeOfWork';
COMMENT ON COLUMN affiliate_links.status IS 'active: clickable, inactive: paused, expired: contract/content ended';

-- ========================================
-- STEP 5: Create click_events Table (Hypertable)
-- ========================================
CREATE TABLE IF NOT EXISTS click_events
(
    id                UUID                 DEFAULT gen_random_uuid(),
    affiliate_link_id UUID        NOT NULL REFERENCES affiliate_links (id) ON DELETE CASCADE,
    user_id           UUID        REFERENCES users (id) ON DELETE SET NULL,
    clicked_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address        INET,
    user_agent        TEXT,
    referrer_url      TEXT,
    session_id        VARCHAR(255),

-- Composite primary key required for TimescaleDB hypertable
    PRIMARY KEY (id, clicked_at)
);

-- Convert to TimescaleDB hypertable
select
    create_hypertable(
        'click_events',
        'clicked_at',
        chunk_time_interval => interval '1 day',
        if_not_exists => true
    )
;

ALTER TABLE click_events
    SET (timescaledb.compress = true);

ALTER TABLE click_events
    SET (
        timescaledb.compress_orderby = 'clicked_at DESC',
        timescaledb.compress_segmentby = 'id'
        );

-- Add compression policy (compress chunks older than 7 days)
select add_compression_policy('click_events', interval '7 days', if_not_exists => true)
;

-- Add retention policy (drop chunks older than 90 days)
select add_retention_policy('click_events', interval '90 days', if_not_exists => true)
;

-- Indexes for click_events (create AFTER hypertable conversion)
CREATE INDEX IF NOT EXISTS idx_click_events_affiliate_link_id
    ON click_events (affiliate_link_id, clicked_at DESC);

CREATE INDEX IF NOT EXISTS idx_click_events_user_id
    ON click_events (user_id, clicked_at DESC) WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_click_events_session_id
    ON click_events (session_id, clicked_at DESC) WHERE session_id IS NOT NULL;

-- Comments for documentation
COMMENT ON TABLE click_events IS 'TimescaleDB hypertable storing individual click events with 90-day retention';
COMMENT ON COLUMN click_events.clicked_at IS 'Partition key for TimescaleDB - DO NOT UPDATE after insert';
COMMENT ON COLUMN click_events.ip_address IS 'Anonymized for privacy - store hashed or truncated version';
COMMENT ON COLUMN click_events.user_agent IS 'Browser user agent for bot detection';

-- Verify hypertable creation
DO
$$
    DECLARE
        hypertable_count INTEGER;
    BEGIN
        SELECT COUNT(*)
        INTO hypertable_count
        FROM timescaledb_information.hypertables
        WHERE hypertable_name = 'click_events';

        IF hypertable_count = 0 THEN
            RAISE EXCEPTION 'click_events hypertable creation failed';
        ELSE
            RAISE NOTICE '✅ click_events successfully created as hypertable';
        END IF;
    END
$$;

-- ========================================
-- STEP 6: Create Continuous Aggregate (Materialized View)
-- ========================================
-- Pre-compute hourly click statistics for faster analytics queries
CREATE MATERIALIZED VIEW IF NOT EXISTS click_events_hourly
            WITH (timescaledb.continuous) AS
SELECT time_bucket('1 hour', clicked_at)                                AS hour,
       affiliate_link_id,
       COUNT(*)                                                         AS total_clicks,
       COUNT(DISTINCT COALESCE(user_id::text, ip_address::text))        AS unique_users,
       COUNT(DISTINCT session_id) FILTER (WHERE session_id IS NOT NULL) AS unique_sessions
FROM click_events
GROUP BY hour, affiliate_link_id
WITH NO DATA;

-- Add refresh policy (refresh every hour, keep last 30 days)
select
    add_continuous_aggregate_policy(
        'click_events_hourly',
        start_offset => interval '30 days',
        end_offset => interval '1 hour',
        schedule_interval => interval '1 hour',
        if_not_exists => true
    )
;

-- ========================================
-- STEP 7: Update Admin Config
-- ========================================
-- Add CTR aggregation settings to config table
-- Check if config table exists
DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'configs') THEN
            RAISE EXCEPTION 'configs table does not exist - cannot add admin configs entries';
        END IF;
    END
$$;

-- Migrate value_type to using type instead of check
DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'value_type') THEN
            CREATE TYPE value_type AS ENUM ('STRING', 'NUMBER', 'BOOLEAN', 'JSON', 'ARRAY', 'TIME');
            RAISE NOTICE '✅ Created new ENUM type value_type';
        ELSE
            RAISE NOTICE 'value_type enum already exists';
        END IF;
        alter table configs
            drop constraint configs_type_check;
        ALTER TABLE configs
            ALTER COLUMN value_type TYPE value_type
                USING value_type::value_type;
        alter table configs
            alter column value_type set default 'STRING';

        INSERT INTO configs (key, value, value_type, description, created_at, updated_at)
        VALUES ('ctr_aggregation_interval_minutes',
                '5',
                'NUMBER',
                'Interval in minutes for CTR aggregation cron job (default: 5 minutes)',
                NOW(),
                NOW()),
               ('ctr_aggregation_enabled',
                'true',
                'BOOLEAN',
                'Enable/disable automatic CTR aggregation from click_events to kpi_metrics',
                NOW(),
                NOW())
        ON CONFLICT (key) DO NOTHING;
    END;
$$;
-- Insert admin configs entries (skip if already exist)
-- Verify admin configs insertion
DO
$$
    DECLARE
        config_count INTEGER;
    BEGIN
        SELECT COUNT(*)
        INTO config_count
        FROM configs
        WHERE key IN ('ctr_aggregation_interval_minutes', 'ctr_aggregation_enabled');

        IF config_count < 2 THEN
            RAISE WARNING 'Not all admin config entries were inserted - may already exist';
        ELSE
            RAISE NOTICE '✅ Admin config entries added successfully';
        END IF;
    END
$$;
commit
;
call refresh_continuous_aggregate('click_events_hourly', null, null)
;

