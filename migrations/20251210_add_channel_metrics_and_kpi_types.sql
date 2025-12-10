-- Migration: Add metrics column to channels and update KPI reference types
-- Date: 2025-12-10
-- Description: 
--   1. Add metrics JSONB column to channels table for page/user level metrics
--   2. Update kpi_metrics reference_type CHECK constraint to include CHANNEL and CONTENT_CHANNEL

-- Add metrics column to channels table
ALTER TABLE channels ADD column IF NOT EXISTS metrics JSONB DEFAULT '{}'::jsonb ;

-- Create GIN index for efficient JSONB querying on channel metrics
CREATE INDEX IF NOT EXISTS idx_channels_metrics ON channels USING GIN (metrics) ;

-- Update kpi_metrics reference_type CHECK constraint to include new types
-- First drop the existing constraint if it exists
ALTER TABLE kpi_metrics DROP CONSTRAINT IF EXISTS kpi_metrics_reference_type_check ;

-- Add updated constraint with new reference types
ALTER TYPE reference_type ADD VALUE 'CHANNEL' ;
ALTER TYPE reference_type ADD VALUE 'CONTENT_CHANNEL' ;

-- Create composite index for kpi_metrics by reference_type and reference_id
CREATE INDEX IF NOT EXISTS idx_kpi_metrics_reference_type_id ON kpi_metrics (reference_type,
reference_id) ;

-- Add comment for documentation
COMMENT ON COLUMN channels.metrics IS 'JSONB column storing page/user level metrics from social platforms (Facebook fan_count, TikTok followers, etc.)' ;
