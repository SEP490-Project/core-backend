-- Database Index Verification for Affiliate Link Tracking
-- Run this query to verify all required indexes are present

-- 1. Check indexes on affiliate_links table
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'affiliate_links'
ORDER BY indexname;

-- Expected indexes:
-- - idx_affiliate_links_hash (unique on hash)
-- - idx_affiliate_links_contract_id
-- - idx_affiliate_links_content_id
-- - idx_affiliate_links_channel_id
-- - idx_affiliate_links_status
-- - idx_affiliate_links_tracking_url_context (unique composite)

-- 2. Check indexes on click_events hypertable
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'click_events'
ORDER BY indexname;

-- Expected indexes:
-- - idx_click_events_affiliate_link_id_clicked_at (DESC for recent queries)
-- - idx_click_events_user_id_clicked_at (WHERE user_id IS NOT NULL)
-- - idx_click_events_ip_address (for analytics)

-- 3. Check indexes on kpi_metrics hypertable
SELECT
    schemaname,
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'kpi_metrics'
ORDER BY indexname;

-- Expected indexes:
-- - idx_kpi_metrics_reference (composite on reference_type, reference_id, type)
-- - idx_kpi_metrics_created_at (for time-range queries)

-- 4. Verify index usage statistics
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan as index_scans,
    idx_tup_read as tuples_read,
    idx_tup_fetch as tuples_fetched
FROM pg_stat_user_indexes
WHERE tablename IN ('affiliate_links', 'click_events', 'kpi_metrics')
ORDER BY tablename, indexname;

-- 5. Find missing indexes (tables without indexes on foreign keys)
SELECT
    t.tablename,
    STRING_AGG(a.attname, ', ') as columns_without_indexes
FROM pg_tables t
CROSS JOIN LATERAL (
    SELECT a.attname
    FROM pg_attribute a
    WHERE a.attrelid = (t.schemaname || '.' || t.tablename)::regclass
      AND a.attname LIKE '%_id'
      AND NOT EXISTS (
          SELECT 1
          FROM pg_index i
          WHERE i.indrelid = a.attrelid
            AND a.attnum = ANY(i.indkey)
      )
) a
WHERE t.tablename IN ('affiliate_links', 'click_events', 'kpi_metrics')
GROUP BY t.tablename;

-- 6. Check for unused indexes (consider dropping if idx_scan = 0)
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size
FROM pg_stat_user_indexes
WHERE tablename IN ('affiliate_links', 'click_events', 'kpi_metrics')
  AND idx_scan = 0
ORDER BY pg_relation_size(indexrelid) DESC;

-- 7. Check TimescaleDB chunk indexes
SELECT
    chunk_name,
    hypertable_name,
    range_start,
    range_end,
    is_compressed
FROM timescaledb_information.chunks
WHERE hypertable_name IN ('click_events', 'kpi_metrics')
ORDER BY hypertable_name, range_start DESC
LIMIT 10;

-- 8. Verify query performance with EXPLAIN ANALYZE
-- Test query 1: Get recent clicks for an affiliate link
EXPLAIN ANALYZE
SELECT *
FROM click_events
WHERE affiliate_link_id = '00000000-0000-0000-0000-000000000000'
  AND clicked_at >= NOW() - INTERVAL '7 days'
ORDER BY clicked_at DESC
LIMIT 100;

-- Test query 2: Get KPI metrics for a contract
EXPLAIN ANALYZE
SELECT *
FROM kpi_metrics
WHERE reference_type = 'AFFILIATE_LINK'
  AND reference_id IN (
      SELECT id FROM affiliate_links WHERE contract_id = '00000000-0000-0000-0000-000000000000'
  )
  AND created_at >= NOW() - INTERVAL '30 days'
ORDER BY created_at DESC;

-- 9. Index bloat check
SELECT
    schemaname,
    tablename,
    indexname,
    pg_size_pretty(pg_relation_size(indexrelid)) as index_size,
    CASE
        WHEN pg_relation_size(indexrelid) > 0 THEN
            ROUND((pg_relation_size(indexrelid)::numeric - 
                   pg_relation_size(indexrelid, 'main')::numeric) * 100 / 
                  pg_relation_size(indexrelid)::numeric, 2)
        ELSE 0
    END as bloat_percentage
FROM pg_stat_user_indexes
WHERE tablename IN ('affiliate_links', 'click_events', 'kpi_metrics')
  AND pg_relation_size(indexrelid) > 1000000 -- Only indexes > 1MB
ORDER BY pg_relation_size(indexrelid) DESC;

-- 10. Recommendations based on query patterns
-- If you see high sequential scans, consider adding indexes
SELECT
    schemaname,
    tablename,
    seq_scan,
    seq_tup_read,
    idx_scan,
    idx_tup_fetch,
    CASE
        WHEN seq_scan > 0 AND idx_scan = 0 THEN 'Consider adding indexes'
        WHEN seq_scan > idx_scan THEN 'Sequential scans dominate'
        ELSE 'Good index usage'
    END as recommendation
FROM pg_stat_user_tables
WHERE tablename IN ('affiliate_links', 'click_events', 'kpi_metrics');
