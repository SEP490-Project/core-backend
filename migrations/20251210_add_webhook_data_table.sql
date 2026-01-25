-- Migration: Add webhook_data table for storing raw webhook payloads
-- Description: Creates a table to store raw webhook data for audit/debugging purposes

-- Create webhook_data table
CREATE TABLE IF NOT EXISTS webhook_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Source of webhook: 'payos', 'facebook', 'tiktok', etc.
    source VARCHAR(50) NOT NULL,
    -- Type of event: 'payment.success', 'content.posted', etc.
    event_type VARCHAR(100),
    -- External reference ID from the source
    external_id VARCHAR(255),
    raw_query JSONB,                       -- Raw query string if applicable
    raw_payload JSONB NOT NULL,            -- Raw webhook payload
    -- Whether this webhook has been processed
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP WITH TIME ZONE, -- When it was processed
    error_message TEXT,                    -- Error message if processing failed
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),

    -- Indexes
    constraint CHK_WEBHOOK_SOURCE CHECK (
        source IN ('payos', 'facebook', 'tiktok', 'ghn', 'other')
    )
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_webhook_data_source ON webhook_data (source);
CREATE INDEX IF NOT EXISTS idx_webhook_data_external_id ON webhook_data (
    external_id
);
CREATE INDEX IF NOT EXISTS idx_webhook_data_created_at ON webhook_data (
    created_at DESC
);
CREATE INDEX IF NOT EXISTS idx_webhook_data_processed ON webhook_data (
    processed
) WHERE processed = FALSE ;
CREATE INDEX IF NOT EXISTS idx_webhook_data_raw_payload ON webhook_data USING GIN (raw_payload) ;

-- Comments
COMMENT ON TABLE webhook_data IS 'Stores raw webhook payloads for audit and debugging purposes' ;
COMMENT ON COLUMN webhook_data.source IS 'Source of the webhook: payos, facebook, tiktok, ghn, other' ;
COMMENT ON COLUMN webhook_data.event_type IS 'Type of event from the webhook source' ;
COMMENT ON COLUMN webhook_data.external_id IS 'External reference ID from the webhook source' ;
COMMENT ON COLUMN webhook_data.raw_query IS 'Raw query string received with the webhook, if applicable' ;
COMMENT ON COLUMN webhook_data.raw_payload IS 'Complete raw JSON payload received from webhook' ;
COMMENT ON COLUMN webhook_data.processed IS 'Whether this webhook has been successfully processed' ;
COMMENT ON COLUMN webhook_data.processed_at IS 'Timestamp when the webhook was processed' ;
COMMENT ON COLUMN webhook_data.error_message IS 'Error message if webhook processing failed' ;
