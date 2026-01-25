DO
$$
    BEGIN

    alter table channels
    add column channel_code varchar(255),
    add column external_id varchar(255),
    add column account_name varchar(255),
    add column hashed_access_token text,
    add column hashed_refresh_token text,
    add column access_token_expires_at timestamp with time zone,
    add column refresh_token_expires_at timestamp with time zone,
    add column last_synced_at timestamp with time zone;


    alter table content_channels
    add column if not exists external_post_id varchar(255),
    add column if not exists external_post_url text,
    add column if not exists metrics jsonb,
    add column if not exists last_error text,
    add column if not exists published_at timestamp with time zone;

    DO
    $$
        BEGIN
-- STEP 1: Create new ENUM type if not exists
            IF NOT EXISTS (SELECT 1
                           FROM pg_type
                           WHERE typname = 'auto_post_status') THEN
                CREATE TYPE auto_post_status AS ENUM ('PENDING', 'POSTED', 'FAILED', 'SKIPPED');
raise notice '✅ Created new ENUM type auto_post_status'
;
else raise notice 'auto_post_status enum already exists'
;
end if
;

-- STEP 2: Find and drop the old check constraint if it exists
            ALTER TABLE content_channels
                DROP CONSTRAINT content_channels_auto_post_status_check;

-- STEP 3: Alter column to ENUM type using safe cast
            ALTER TABLE content_channels
                ALTER COLUMN auto_post_status TYPE auto_post_status
                    USING auto_post_status::auto_post_status;

raise notice '✅ Converted orders.order_status to enum type successfully'
;
end
$$
    ;

    alter table users
    add column is_facebook_oauth boolean not null default false,
    add column is_tiktok_oauth boolean not null default false,
    add column oauth_metadata jsonb;

    END
$$
;

