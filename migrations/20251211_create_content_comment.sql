-- Create content_schedules table
CREATE TABLE IF NOT EXISTS content_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_channel_id UUID NOT NULL REFERENCES content_channels (
        id
    ) ON DELETE CASCADE,
    comment TEXT NOT NULL,
    reactions JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    created_by UUID NOT NULL REFERENCES users (id) ON DELETE SET NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_by UUID REFERENCES users (id) ON DELETE SET NULL,
    is_censored BOOLEAN DEFAULT FALSE,
    censor_reason TEXT
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_content_comments_content_channel_id ON content_comments (
    content_channel_id
);
CREATE INDEX IF NOT EXISTS idx_content_comments_created_by ON content_comments (
    created_by
);
create index if not exists idx_content_comments_reactions_type on content_comments using gin ((reactions - > 'type')) ;

-- Comments
COMMENT ON TABLE content_comments IS 'Stores comments for content' ;
comment ON COLUMN content_comments.content_channel_id IS 'Reference to the content channel being commented on' ;
COMMENT ON COLUMN content_comments.comment IS 'The text of the comment' ;
COMMENT ON COLUMN content_comments.reactions IS 'JSONB field storing reactions to the comment' ;
COMMENT ON COLUMN content_comments.is_censored IS 'Indicates if the comment has been censored' ;
COMMENT ON COLUMN content_comments.censor_reason IS 'Reason for censoring the comment, if applicable' ;
COMMENT ON COLUMN content_comments.created_by IS 'User who created the comment' ;
COMMENT ON COLUMN content_comments.updated_by IS 'User who last updated the comment' ;
