-- Migration: Create Content and Blog Management Tables
-- Date: 2025-10-22
-- Description: Creates contents and blogs tables with FSM state support

-- Create content_status enum type
CREATE TYPE content_status AS ENUM (
    'DRAFT',
    'AWAIT_STAFF',
    'AWAIT_BRAND',
    'REJECTED',
    'APPROVED',
    'POSTED'
);

-- Create content_type enum type
CREATE TYPE content_type AS ENUM (
    'POST',
    'VIDEO'
);

-- Create contents table
CREATE TABLE contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID,
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    type content_type NOT NULL,
    status content_status NOT NULL DEFAULT 'DRAFT',
    publish_date TIMESTAMP,
    affiliate_link VARCHAR(1000),
    ai_generated_text TEXT,
    rejection_feedback TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    
    CONSTRAINT fk_contents_task FOREIGN KEY (task_id) 
        REFERENCES tasks(id) ON DELETE SET NULL
);

-- Create blogs table (weak entity)
CREATE TABLE blogs (
    content_id UUID PRIMARY KEY,
    author_id UUID NOT NULL,
    tags JSONB,
    excerpt TEXT,
    read_time INTEGER,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_blogs_content FOREIGN KEY (content_id) 
        REFERENCES contents(id) ON DELETE CASCADE,
    CONSTRAINT fk_blogs_author FOREIGN KEY (author_id) 
        REFERENCES users(id)
);

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
