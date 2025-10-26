-- Migration: Add Notification and DeviceToken tables
-- Date: 2025-10-26
-- Feature: 002-notification-integrations
-- Create notifications table


-- ================================================================
--  Notifications Table Migration Script
--  Safely migrate old notifications schema → new JSONB-based schema
-- ================================================================

BEGIN;

-- 1️⃣  Backup the existing table (safety net)
CREATE TABLE IF NOT EXISTS notifications_backup AS TABLE notifications;
COMMENT ON TABLE notifications_backup IS 'Backup of original notifications table before migration';

-- 2️⃣  Create the new target table (temporary name: notifications_new)
CREATE TABLE IF NOT EXISTS notifications_new (
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

    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);

-- 3️⃣  Migrate and transform data
INSERT INTO notifications_new (
    id,
    user_id,
    type,
    status,
    delivery_attempts,
    recipient_info,
    content_data,
    platform_config,
    error_details,
    created_at,
    updated_at
)
SELECT
    id,
    user_id,
    type,
    status,
    '[]'::jsonb AS delivery_attempts,
    jsonb_build_object(
        'user_id', user_id
    ) AS recipient_info,
    jsonb_build_object(
        'message', message,
        'data', message_data
    ) AS content_data,
    jsonb_build_object(
        'channel', channel,
        'related_id', related_id,
        'send_time', send_time
    ) AS platform_config,
    '{}'::jsonb AS error_details,
    created_at,
    updated_at
FROM notifications;

-- 4️⃣  Recreate all indexes from new DDL
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications_new (user_id);
CREATE INDEX IF NOT EXISTS idx_notifications_status ON notifications_new (status);
CREATE INDEX IF NOT EXISTS idx_notifications_type ON notifications_new (type);
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications_new (created_at);
CREATE INDEX IF NOT EXISTS idx_notifications_deleted_at ON notifications_new (deleted_at);

-- JSONB GIN indexes
CREATE INDEX IF NOT EXISTS idx_notifications_delivery_attempts ON notifications_new USING GIN (delivery_attempts);
CREATE INDEX IF NOT EXISTS idx_notifications_recipient_info ON notifications_new USING GIN (recipient_info);
CREATE INDEX IF NOT EXISTS idx_notifications_error_details ON notifications_new USING GIN (error_details);

-- 5️⃣  Swap the tables
ALTER TABLE notifications RENAME TO notifications_old;
ALTER TABLE notifications_new RENAME TO notifications;

-- 6️⃣  Optional: Verify migrated rows count
DO $$
DECLARE
    old_count INT;
    new_count INT;
BEGIN
    SELECT COUNT(*) INTO old_count FROM notifications_old;
    SELECT COUNT(*) INTO new_count FROM notifications;
    RAISE NOTICE 'Old table rows: %, New table rows: %', old_count, new_count;
END $$;

drop table if exists notifications_old;
drop table if exists notifications_backup;

-- ✅ Commit transaction if all went well
COMMIT;

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

