-- Migration: Create content_schedules table for scheduling content publishing
-- Uses RabbitMQ delayed message plugin for execution

-- Create schedule status enum
DO $$ BEGIN
    CREATE TYPE schedule_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

-- Create content_schedules table
CREATE TABLE IF NOT EXISTS content_schedules (
id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
content_channel_id UUID NOT NULL REFERENCES content_channels (id) ON DELETE CASCADE,
scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
status VARCHAR (30) NOT NULL DEFAULT 'PENDING',
retry_count INT NOT NULL DEFAULT 0,
last_error TEXT,
executed_at TIMESTAMP WITH TIME ZONE,
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
deleted_at TIMESTAMP WITH TIME ZONE,
created_by UUID NOT NULL REFERENCES users (id) ON DELETE SET NULL
) ;

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_content_schedules_status ON content_schedules (status) WHERE deleted_at IS NULL ;
CREATE INDEX IF NOT EXISTS idx_content_schedules_scheduled_at ON content_schedules (scheduled_at) WHERE deleted_at IS NULL ;
CREATE INDEX IF NOT EXISTS idx_content_schedules_content_channel_id ON content_schedules (content_channel_id) WHERE deleted_at IS NULL ;
CREATE INDEX IF NOT EXISTS idx_content_schedules_created_by ON content_schedules (created_by) ;
CREATE INDEX IF NOT EXISTS idx_content_schedules_pending ON content_schedules (scheduled_at) WHERE status = 'PENDING' AND deleted_at IS NULL ;

-- Comments
COMMENT ON TABLE content_schedules IS 'Stores scheduled content publishing jobs processed via RabbitMQ delayed messages' ;
COMMENT ON COLUMN content_schedules.content_channel_id IS 'Reference to the content channel to be published' ;
COMMENT ON COLUMN content_schedules.scheduled_at IS 'The time when content should be published' ;
COMMENT ON COLUMN content_schedules.status IS 'Current status: PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED' ;
COMMENT ON COLUMN content_schedules.retry_count IS 'Number of retry attempts after failures' ;
COMMENT ON COLUMN content_schedules.last_error IS 'Error message from the last failed attempt' ;
COMMENT ON COLUMN content_schedules.executed_at IS 'Timestamp when the schedule was actually executed' ;
