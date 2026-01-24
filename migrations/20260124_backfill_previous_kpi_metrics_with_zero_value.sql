BEGIN;

-- 1. PREPARE THE INSERT
WITH distinct_metrics_per_post AS (
    -- Identify which metric types (VIEWS, LIKES, etc.) exist for each post
    -- We only want to insert 0s for types that actually exist
    SELECT DISTINCT
        km.reference_id,
        km.type
    FROM kpi_metrics km
    WHERE km.reference_type = 'CONTENT_CHANNEL'
),

missing_zero_targets AS (
    -- Calculate the "Safe Zero Date" (Publish Date - 2 Days)
    SELECT
        dm.reference_id,
        dm.type,
        (cc.published_at - INTERVAL '2 days') AS zero_date
    FROM distinct_metrics_per_post dm
    JOIN content_channels cc ON cc.id = dm.reference_id
    WHERE cc.published_at IS NOT NULL
)

INSERT INTO kpi_metrics (
    id,
    reference_id,
    reference_type,
    type,
    value,
    recorded_date
)
SELECT
    gen_random_uuid(),          -- Generate ID
    mzt.reference_id,
    'CONTENT_CHANNEL',
    mzt.type,
    0,                          -- The Value is 0
    mzt.zero_date              -- The Date is 2 days before publish
FROM missing_zero_targets mzt
WHERE NOT EXISTS (
    -- IDEMPOTENCY CHECK:
    -- Don't insert if we already have a record for this ID+Type
    -- at roughly this time (or exactly this time) to prevent duplicates.
        SELECT 1
        FROM kpi_metrics km
        WHERE
            km.reference_id = mzt.reference_id
            AND km.type = mzt.type
            AND km.value = 0
            AND km.recorded_date = mzt.zero_date
    );

-- 2. VERIFY THE DATA (Check what we just did)
-- This shows the newly created "Zero" records
SELECT
    km.recorded_date,
    km.type,
    km.value,
    km.reference_id,
    'NEW_ZERO_ANCHOR' AS status
FROM kpi_metrics km
WHERE km.value = 0
ORDER BY km.recorded_date ASC
LIMIT 20;

-- 3. ROLLBACK or COMMIT
-- Run with ROLLBACK first. If the output looks correct, change to COMMIT.
ROLLBACK;
