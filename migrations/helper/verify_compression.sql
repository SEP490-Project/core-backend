-- TimescaleDB Compression Verification and Statistics
-- Run this query to check compression status and ratios

-- 1. Overall compression statistics
SELECT
    hypertable_schema,
    hypertable_name,
    total_chunks,
    number_compressed_chunks,
    ROUND((number_compressed_chunks::numeric / NULLIF(total_chunks, 0)) * 100, 2) as compression_percentage,
    pg_size_pretty(before_compression_total_bytes) as uncompressed_size,
    pg_size_pretty(after_compression_total_bytes) as compressed_size,
    ROUND((1 - after_compression_total_bytes::numeric / NULLIF(before_compression_total_bytes, 0)) * 100, 2) as space_saved_percentage,
    ROUND(before_compression_total_bytes::numeric / NULLIF(after_compression_total_bytes, 1), 2) as compression_ratio
FROM timescaledb_information.hypertable_compression_stats
WHERE hypertable_name IN ('click_events', 'kpi_metrics');

-- 2. Detailed chunk-level compression status
SELECT
    h.hypertable_name,
    c.chunk_name,
    c.range_start,
    c.range_end,
    c.is_compressed,
    pg_size_pretty(c.total_bytes) as chunk_size,
    CASE
        WHEN c.is_compressed THEN 'Compressed'
        WHEN c.range_end < NOW() - INTERVAL '7 days' THEN 'Eligible for compression'
        ELSE 'Too recent to compress'
    END as status
FROM timescaledb_information.chunks c
JOIN timescaledb_information.hypertables h ON h.hypertable_name = c.hypertable_name
WHERE h.hypertable_name IN ('click_events', 'kpi_metrics')
ORDER BY h.hypertable_name, c.range_start DESC
LIMIT 20;

-- 3. Compression policies status
SELECT
    application_name,
    hypertable_name,
    older_than,
    schedule_interval,
    max_runtime,
    CASE
        WHEN proc_name LIKE '%compress%' THEN 'Active'
        ELSE 'Not configured'
    END as policy_status
FROM timescaledb_information.jobs j
JOIN timescaledb_information.hypertables h ON j.hypertable_name = h.hypertable_name
WHERE h.hypertable_name IN ('click_events', 'kpi_metrics')
  AND j.proc_name LIKE '%compress%';

-- 4. Uncompressed chunks that should be compressed
SELECT
    hypertable_name,
    chunk_name,
    range_start,
    range_end,
    pg_size_pretty(total_bytes) as size,
    EXTRACT(DAY FROM NOW() - range_end) as days_old
FROM timescaledb_information.chunks
WHERE hypertable_name IN ('click_events', 'kpi_metrics')
  AND is_compressed = FALSE
  AND range_end < NOW() - INTERVAL '7 days'
ORDER BY range_end ASC;

-- 5. Compression job execution history
SELECT
    hypertable_name,
    job_id,
    last_run_started_at,
    last_successful_finish,
    last_run_status,
    total_runs,
    total_successes,
    total_failures
FROM timescaledb_information.job_stats
WHERE hypertable_name IN ('click_events', 'kpi_metrics')
  AND proc_name LIKE '%compress%'
ORDER BY last_run_started_at DESC;

-- 6. Table sizes comparison (compressed vs uncompressed)
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as indexes_size,
    ROUND(100.0 * pg_relation_size(schemaname||'.'||tablename) / 
          NULLIF(pg_total_relation_size(schemaname||'.'||tablename), 0), 2) as table_pct
FROM pg_tables
WHERE tablename IN ('click_events', 'kpi_metrics')
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- 7. Row count per chunk
SELECT
    hypertable_name,
    chunk_name,
    is_compressed,
    num_rows,
    pg_size_pretty(uncompressed_heap_size) as heap_size,
    pg_size_pretty(uncompressed_index_size) as index_size,
    CASE
        WHEN is_compressed THEN pg_size_pretty(compressed_heap_size)
        ELSE NULL
    END as compressed_heap_size
FROM timescaledb_information.chunks_detailed_size
WHERE hypertable_name IN ('click_events', 'kpi_metrics')
ORDER BY hypertable_name, range_start DESC
LIMIT 10;

-- 8. Compression effectiveness by column
SELECT
    hypertable_name,
    attname as column_name,
    pg_size_pretty(before_compression_total_bytes) as before_compression,
    pg_size_pretty(after_compression_total_bytes) as after_compression,
    ROUND((1 - after_compression_total_bytes::numeric / NULLIF(before_compression_total_bytes, 0)) * 100, 2) as compression_ratio
FROM timescaledb_information.compression_settings
WHERE hypertable_name IN ('click_events', 'kpi_metrics')
ORDER BY hypertable_name, before_compression_total_bytes DESC;

-- 9. Manual compression trigger (if needed)
-- Uncomment to manually compress eligible chunks
/*
SELECT compress_chunk(c.chunk_name)
FROM timescaledb_information.chunks c
WHERE c.hypertable_name = 'click_events'
  AND c.is_compressed = FALSE
  AND c.range_end < NOW() - INTERVAL '7 days'
LIMIT 5;
*/

-- 10. Recommendations
WITH compression_stats AS (
    SELECT
        hypertable_name,
        total_chunks,
        number_compressed_chunks,
        before_compression_total_bytes,
        after_compression_total_bytes,
        ROUND((1 - after_compression_total_bytes::numeric / NULLIF(before_compression_total_bytes, 0)) * 100, 2) as savings
    FROM timescaledb_information.hypertable_compression_stats
    WHERE hypertable_name IN ('click_events', 'kpi_metrics')
)
SELECT
    hypertable_name,
    CASE
        WHEN number_compressed_chunks = 0 THEN 'WARNING: No chunks compressed yet'
        WHEN savings < 50 THEN 'INFO: Compression ratio below 50%, review column types'
        WHEN savings < 70 THEN 'OK: Decent compression ratio'
        WHEN savings >= 70 THEN 'EXCELLENT: High compression ratio (>70%)'
    END as compression_assessment,
    CASE
        WHEN total_chunks - number_compressed_chunks > 5 THEN 
            'ACTION NEEDED: ' || (total_chunks - number_compressed_chunks)::text || ' chunks eligible for compression'
        ELSE 'No action needed'
    END as action_required
FROM compression_stats;
