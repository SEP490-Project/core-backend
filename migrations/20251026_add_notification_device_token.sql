-- Migration: Add Notification and DeviceToken tables
-- Date: 2025-10-26
-- Feature: 002-notification-integrations
-- Create notifications table
CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    
-- JSONB columns for flexible metadata
    delivery_attempts JSONB NOT NULL DEFAULT '[]'::jsonb,
    recipient_info JSONB NOT NULL,
    content_data JSONB NOT NULL,
    platform_config JSONB,
    error_details JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
-- Foreign key constraint
    CONSTRAINT fk_notifications_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- Create device_tokens table
CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL,
    registered_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_valid BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
-- Foreign key constraint
    CONSTRAINT fk_device_tokens_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE
);


-- Alter users table to add notification preference columns
alter table users 
    add column if not exists email_enabled boolean default true not null,
    add column if not exists push_enabled boolean default true not null;


-- ========== Notifications Indexes ==========
-- Standard B-tree indexes
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications(status);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications(type);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON notifications(deleted_at);

-- GIN indexes for JSONB columns (enables efficient JSONB queries)
CREATE INDEX IF NOT EXISTS idx_notifications_delivery_attempts ON notifications USING GIN (delivery_attempts);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_info ON notifications USING GIN (recipient_info);
CREATE INDEX IF NOT EXISTS idx_notifications_error_details ON notifications USING GIN (error_details);

-- ========== DeviceTokens Indexes ==========
-- Standard B-tree indexes
CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(token) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_device_tokens_is_valid ON device_tokens(is_valid);
CREATE INDEX IF NOT EXISTS idx_device_tokens_last_used_at ON device_tokens(last_used_at);
CREATE INDEX IF NOT EXISTS idx_device_tokens_deleted_at ON device_tokens(deleted_at);

-- ========== Comments for Documentation ==========
COMMENT ON TABLE notifications IS 'Stores all notification attempts (email and push) with flexible JSONB metadata';
COMMENT ON COLUMN notifications.type IS 'Notification type: EMAIL or PUSH';
COMMENT ON COLUMN notifications.status IS 'Delivery status: PENDING, SENT, FAILED, RETRYING';
COMMENT ON COLUMN notifications.delivery_attempts IS 'Array of delivery attempts with timestamps and results';
COMMENT ON COLUMN notifications.recipient_info IS 'Email address or FCM device tokens';
COMMENT ON COLUMN notifications.content_data IS 'Notification content (subject, body, template data)';
COMMENT ON COLUMN notifications.platform_config IS 'iOS/Android specific push notification settings';
COMMENT ON COLUMN notifications.error_details IS 'Last error information if delivery failed';

COMMENT ON TABLE device_tokens IS 'Stores FCM device tokens for push notifications';
COMMENT ON COLUMN device_tokens.token IS 'Firebase Cloud Messaging device token';
COMMENT ON COLUMN device_tokens.platform IS 'Mobile platform: IOS or ANDROID';
COMMENT ON COLUMN device_tokens.is_valid IS 'Whether token is still valid (false if FCM reports invalid)';
COMMENT ON COLUMN device_tokens.last_used_at IS 'Last time a notification was sent to this token';

-- ========== Verification Queries ==========
-- Verify tables created
select table_name, table_type
from information_schema.tables
where table_schema = 'public' and table_name in ('notifications', 'device_tokens')
;

-- Verify indexes created
select indexname, indexdef
from pg_indexes
where schemaname = 'public' and tablename in ('notifications', 'device_tokens')
order by tablename, indexname
;

-- Verify foreign keys
select
    tc.constraint_name,
    tc.table_name,
    kcu.column_name,
    ccu.table_name as foreign_table_name,
    ccu.column_name as foreign_column_name
from information_schema.table_constraints as tc
join
    information_schema.key_column_usage as kcu
    on tc.constraint_name = kcu.constraint_name
join
    information_schema.constraint_column_usage as ccu
    on ccu.constraint_name = tc.constraint_name
where
    tc.constraint_type = 'FOREIGN KEY'
    and tc.table_name in ('notifications', 'device_tokens')
;

