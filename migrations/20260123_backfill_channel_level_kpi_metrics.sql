-- Migration: Backfill channel-level aggregated KPI metrics
-- Purpose: Before the poller job fix, aggregated metrics for Facebook/TikTok channels
-- were not being persisted to kpi_metrics with reference_type = 'CHANNEL'.
-- This migration aggregates existing CONTENT_CHANNEL level metrics into CHANNEL level.
--
-- Data Patterns:
-- - Website Views/UniqueViews: Incremental (each row = 1 view) -> SUM
-- - All other metrics: Cumulative snapshots -> DISTINCT ON (latest per content_channel), then SUM
--
-- Note: This creates one aggregated record per channel per day per metric type.
-- After running this migration, the poller job will continue adding new CHANNEL-level metrics.

DO $$
BEGIN
    -- 1. Backfill aggregated metrics from CONTENT_CHANNEL level to CHANNEL level
    
    INSERT INTO kpi_metrics (reference_id, reference_type, type, value, recorded_date)
    
    -- Part A: Handle cumulative metrics (Social channels)
    -- Get the last snapshot of the day for every metric
    WITH latest_content_channel_daily_metrics AS (
        SELECT DISTINCT ON (km.reference_id, km.type, km.recorded_date::date)
            cc.channel_id,
            km.type,
            km.value,
            km.recorded_date::date as metric_date
        FROM kpi_metrics km
        JOIN content_channels cc ON cc.id = km.reference_id
        JOIN channels ch ON ch.id = cc.channel_id
        WHERE km.reference_type = 'CONTENT_CHANNEL'
          AND ch.code != 'WEBSITE'
          AND cc.auto_post_status = 'POSTED'
          AND km.type IN ('VIEWS', 'LIKES', 'COMMENTS', 'SHARES', 'ENGAGEMENT', 'REACH')
        ORDER BY km.reference_id, km.type, km.recorded_date::date, km.recorded_date DESC
    ),
    social_aggregated AS (
        SELECT 
            channel_id,
            type,
            SUM(value) as total_value,
            metric_date
        FROM latest_content_channel_daily_metrics
        GROUP BY channel_id, type, metric_date
    ),
    
    -- Part B: Handle incremental metrics (Website Views)
    website_views AS (
        SELECT 
            cc.channel_id,
            km.type,
            SUM(km.value) as total_value,
            km.recorded_date::date as metric_date
        FROM kpi_metrics km
        JOIN content_channels cc ON cc.id = km.reference_id
        JOIN channels ch ON ch.id = cc.channel_id
        WHERE km.reference_type = 'CONTENT_CHANNEL'
          AND ch.code = 'WEBSITE'
          AND cc.auto_post_status = 'POSTED'
          AND km.type IN ('VIEWS', 'UNIQUE_VIEWS')
        GROUP BY cc.channel_id, km.type, km.recorded_date::date
    ),
    
    -- Part C: Handle cumulative metrics for Website (Engagement)
    website_engagement AS (
        SELECT DISTINCT ON (km.reference_id, km.type, km.recorded_date::date)
            cc.channel_id,
            km.type,
            km.value,
            km.recorded_date::date as metric_date
        FROM kpi_metrics km
        JOIN content_channels cc ON cc.id = km.reference_id
        JOIN channels ch ON ch.id = cc.channel_id
        WHERE km.reference_type = 'CONTENT_CHANNEL'
          AND ch.code = 'WEBSITE'
          AND cc.auto_post_status = 'POSTED'
          AND km.type IN ('LIKES', 'COMMENTS', 'SHARES', 'ENGAGEMENT')
        ORDER BY km.reference_id, km.type, km.recorded_date::date, km.recorded_date DESC
    ),
    website_engagement_aggregated AS (
        SELECT 
            channel_id,
            type,
            SUM(value) as total_value,
            metric_date
        FROM website_engagement
        GROUP BY channel_id, type, metric_date
    ),
    
    -- Part D: Combine all raw aggregates calculated so far
    raw_channel_metrics AS (
        SELECT channel_id, type, total_value, metric_date FROM social_aggregated
        UNION ALL
        SELECT channel_id, type, total_value, metric_date FROM website_views
        UNION ALL
        SELECT channel_id, type, total_value, metric_date FROM website_engagement_aggregated
    ),

    -- Part E: Special Logic for VIEWS
    -- We separate Views and Engagement to handle the fallback logic
    daily_engagement AS (
        SELECT channel_id, metric_date, total_value 
        FROM raw_channel_metrics WHERE type = 'ENGAGEMENT'
    ),
    daily_existing_views AS (
        SELECT channel_id, metric_date, total_value 
        FROM raw_channel_metrics WHERE type = 'VIEWS'
    ),
    
    -- Calculate Final Views: Use existing VIEWS if > 0, otherwise use ENGAGEMENT
    resolved_views AS (
        SELECT
            COALESCE(v.channel_id, e.channel_id) as channel_id,
            COALESCE(v.metric_date, e.metric_date) as metric_date,
            'VIEWS' as type,
            CASE 
                WHEN COALESCE(v.total_value, 0) > 0 THEN v.total_value
                ELSE e.total_value 
            END as total_value
        FROM daily_existing_views v
        FULL OUTER JOIN daily_engagement e 
          ON v.channel_id = e.channel_id AND v.metric_date = e.metric_date
    ),

    -- Part F: Final Dataset
    -- Combine all non-views metrics + the resolved views
    final_dataset AS (
        -- 1. All metrics except VIEWS
        SELECT channel_id, type, total_value, metric_date
        FROM raw_channel_metrics
        WHERE type != 'VIEWS'
        
        UNION ALL
        
        -- 2. The resolved VIEWS (either real or backfilled from engagement)
        SELECT channel_id, type, total_value, metric_date
        FROM resolved_views
        WHERE total_value > 0 -- Ensure we don't insert 0s if both were 0
    )
    
    -- Insert into kpi_metrics
    SELECT 
        fd.channel_id as reference_id,
        'CHANNEL' as reference_type,
        fd.type,
        fd.total_value as value,
        fd.metric_date + TIME '23:59:59' as recorded_date
    FROM final_dataset fd
    WHERE NOT EXISTS (
          SELECT 1 FROM kpi_metrics km
          WHERE km.reference_id = fd.channel_id
            AND km.reference_type = 'CHANNEL'
            AND km.type = fd.type
            AND km.recorded_date::date = fd.metric_date
      );
      
    RAISE NOTICE 'Backfilled channel-level KPI metrics successfully (including Views fallback).';
END $$ ;

-- Verify the migration (optional - can be run separately to check results)
-- SELECT 
--     ch.name as channel_name,
--     km.type,
--     COUNT(*) as metric_count,
--     MIN(km.recorded_date) as earliest,
--     MAX(km.recorded_date) as latest,
--     SUM(km.value) as total_value
-- FROM kpi_metrics km
-- JOIN channels ch ON ch.id = km.reference_id
-- WHERE km.reference_type = 'CHANNEL'
-- GROUP BY ch.name, km.type
-- ORDER BY ch.name, km.type;
