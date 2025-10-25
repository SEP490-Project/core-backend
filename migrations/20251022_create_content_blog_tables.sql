-- Migration: Create Content and Blog Management Tables
-- Date: 2025-10-22
-- Description: Creates contents and blogs tables with FSM state support

-- Indexes for contents table
CREATE INDEX idx_contents_task_id ON contents(task_id);
CREATE INDEX idx_contents_status ON contents(status);
CREATE INDEX idx_contents_type ON contents(type);
CREATE INDEX idx_contents_created_at ON contents(created_at DESC);
CREATE INDEX idx_contents_deleted_at ON contents(deleted_at);
CREATE INDEX idx_contents_title_search ON contents USING gin(to_tsvector('english', title));
CREATE INDEX idx_contents_body_search ON contents USING gin(to_tsvector('english', body));

-- Indexes for blogs table
CREATE INDEX idx_blogs_author_id ON blogs(author_id);
CREATE INDEX idx_blogs_created_at ON blogs(created_at DESC);
CREATE INDEX idx_blogs_tags ON blogs USING GIN (tags);

-- ContentChannel table modifications (if table already exists)
-- Ensure unique constraint on content_id + channel_id
DO $$
BEGIN
    -- Add unique constraint if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'uq_content_channels_content_channel'
    ) THEN
        ALTER TABLE content_channels 
            ADD CONSTRAINT uq_content_channels_content_channel 
            UNIQUE (content_id, channel_id);
    END IF;
END $$;

-- Comments for documentation
COMMENT ON TABLE contents IS 'Primary content entity for blog posts and videos';
COMMENT ON TABLE blogs IS 'Weak entity extending Content for blog-specific attributes (Type=POST only)';
COMMENT ON COLUMN contents.status IS 'FSM state: DRAFT → AWAIT_STAFF/AWAIT_BRAND → APPROVED → POSTED';
COMMENT ON COLUMN contents.affiliate_link IS 'Required for AFFILIATE contracts, optional for ADVERTISING';
COMMENT ON COLUMN blogs.content_id IS 'Primary key = Foreign key (weak entity pattern)';
COMMENT ON COLUMN blogs.tags IS 'JSONB array of tag strings for categorization';
COMMENT ON COLUMN blogs.read_time IS 'Estimated read time in minutes';
