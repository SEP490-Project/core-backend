-- ========================================
-- Migration: Make Affiliate Link Fields Optional
-- Date: 2025-12-08
-- Description: Allow contract_id, content_id, channel_id to be nullable and add
-- metadata column
-- ========================================
begin
;

-- 1. Drop the unique constraint first as it depends on these columns
ALTER TABLE affiliate_links DROP CONSTRAINT IF EXISTS unique_affiliate_combination;

-- 2. Make columns nullable
ALTER TABLE affiliate_links ALTER COLUMN contract_id DROP NOT NULL;
ALTER TABLE affiliate_links ALTER COLUMN content_id DROP NOT NULL;
ALTER TABLE affiliate_links ALTER COLUMN channel_id DROP NOT NULL;

-- 3. Add metadata column
ALTER TABLE affiliate_links ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;

alter table tasks
    alter column scope_of_work_item_id type varchar(150);

-- 4. Add comment for metadata
COMMENT ON COLUMN affiliate_links.metadata IS 'Flexible storage for additional context (e.g. campaign_id, user_id) when standard relations are not used';

commit
;

