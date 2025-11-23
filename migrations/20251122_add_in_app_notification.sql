-- Add is_read column to notifications table
ALTER TABLE notifications ADD COLUMN is_read BOOLEAN DEFAULT FALSE NOT NULL;

-- Create index for is_read to optimize unread count queries
CREATE INDEX idx_notifications_is_read ON notifications(is_read) WHERE is_read = FALSE;

-- Update comment for type column to include IN_APP
COMMENT ON COLUMN notifications.type IS 'Notification type: EMAIL, PUSH, or IN_APP';

ALTER TYPE auto_post_status ADD VALUE 'IN_PROGRESS'
;


DO
$$
    BEGIN
        -- STEP 1: Create new ENUM type if not exists
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'external_post_type') THEN
            CREATE TYPE external_post_type AS ENUM ('TEXT', 'SINGLE_IMAGE', 'MULTI_IMAGE', 'VIDEO', 'LONG_VIDEO');
            raise notice '✅ Created new ENUM type external_post_type';
        else
            raise notice 'external_post_type enum already exists';
        end if;


        -- STEP 3: Alter column to ENUM type using safe cast
        ALTER TABLE content_channels
            ADD COLUMN if not exists external_post_type external_post_type;

        raise notice '✅ Converted orders.order_status to enum type successfully';
    end
$$

