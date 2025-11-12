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
    add column external_post_id varchar(255),
    add column metrics jsonb;

    alter table users
    add column is_facebook_oauth boolean not null default false,
    add column is_tiktok_oauth boolean not null default false,
    add column oauth_metadata jsonb;

    END
$$
;

