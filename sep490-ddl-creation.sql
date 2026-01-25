CREATE EXTENSION IF NOT EXISTS pg_trgm ;
CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE ;
CREATE EXTENSION IF NOT EXISTS pgcrypto CASCADE ;

-- Unknown how to generate base type type

create type concept_status as enum ('UNPUBLISHED', 'DRAFT', 'PUBLISHED') ;

create type content_status as enum ('DRAFT',
'AWAIT_STAFF',
'AWAIT_BRAND',
'REJECTED',
'APPROVED',
'POSTED') ;

create type content_type as enum ('POST', 'VIDEO') ;

create type reference_type as enum ('CONTENT',
'CAMPAIGN',
'AFFILIATE_LINK',
'CHANNEL',
'CONTENT_CHANNEL',
'SCHEDULE',
'MILESTONE',
'CONTRACT',
'ORDER',
'USER',
'BRAND',
'OTHER',
'NOTIFICATION') ;

create type value_type as enum ('STRING',
'NUMBER',
'BOOLEAN',
'JSON',
'ARRAY',
'TIME',
'TIPTAP_JSON') ;

create type order_status as enum ('PENDING',
'PAID',
'REFUNDED',
'CONFIRMED',
'CANCELLED',
'SHIPPED',
'IN_TRANSIT',
'DELIVERED',
'RECEIVED',
'AWAITING_PICKUP',
'REFUND_REQUEST',
'COMPENSATE_REQUEST',
'COMPENSATED') ;

create type campaign_status as enum ('RUNNING',
'COMPLETED',
'CANCELLED',
'DRAFT') ;

create type payment_transactions_status as enum ('PENDING',
'COMPLETED',
'FAILED',
'CANCELLED',
'EXPIRED') ;

create type auto_post_status as enum ('PENDING',
'POSTED',
'FAILED',
'SKIPPED',
'IN_PROGRESS') ;

create type external_post_type as enum ('TEXT',
'SINGLE_IMAGE',
'MULTI_IMAGE',
'VIDEO',
'LONG_VIDEO') ;

create type pre_order_status as enum ('PENDING',
'PAID',
'PRE_ORDERED',
'CANCELLED',
'AWAITING_PICKUP',
'IN_TRANSIT',
'DELIVERED',
'RECEIVED',
'COMPENSATE_REQUEST',
'COMPENSATED',
'REFUND_REQUEST',
'REFUNDED') ;

create type schedule_status as enum ('PENDING',
'PROCESSING',
'COMPLETED',
'FAILED',
'CANCELLED') ;

create type alert_type as enum ('WARNING', 'ERROR', 'INFO') ;

create type alert_category as enum ('CONTENT_REJECTED',
'LOW_CTR',
'LOW_ENGAGEMENT',
'SCHEDULE_FAILED',
'PENDING_APPROVAL',
'DEADLINE_APPROACHING',
'CAMPAIGN_DEADLINE',
'BUDGET_EXCEEDED',
'ORDER_ISSUE',
'PAYMENT_OVERDUE',
'SYSTEM_HEALTH',
'SECURITY_ISSUE') ;

create type alert_severity as enum ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL') ;

create type alert_status as enum ('ACTIVE', 'RESOLVED', 'EXPIRED') ;

create type contract_payments_status as enum ('PENDING',
'PAID',
'OVERDUE',
'TERMINATED') ;

create type notification_severity as enum ('INFO', 'WARN', 'ERROR', 'SUCCESS') ;

create table if not exists users
(
    id                  uuid                     default gen_random_uuid() not null
        primary key,
    username            varchar(255)                                       not null
        unique,
    email               varchar(255)                                       not null
        unique,
    password_hash       text                                               not null,
    full_name           varchar(255)                                       not null,
    phone               varchar(20),
    role                varchar(50)                                        not null
        constraint users_role_check
            check ((role)::text = ANY
                   (ARRAY [('ADMIN'::character varying)::text, ('MARKETING_STAFF'::character varying)::text, ('CONTENT_STAFF'::character varying)::text, ('SALES_STAFF'::character varying)::text, ('CUSTOMER'::character varying)::text, ('BRAND_PARTNER'::character varying)::text]))
        constraint chk_roles_users
            check ((role)::text = ANY
                   (ARRAY [('ADMIN'::character varying)::text, ('MARKETING_STAFF'::character varying)::text, ('CONTENT_STAFF'::character varying)::text, ('SALES_STAFF'::character varying)::text, ('CUSTOMER'::character varying)::text, ('BRAND_PARTNER'::character varying)::text])),
    created_at          timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at          timestamp with time zone default CURRENT_TIMESTAMP,
    last_login          timestamp with time zone,
    is_active           boolean                  default true,
    deleted_at          timestamp with time zone,
    profile_data        jsonb,
    date_of_birth       date,
    avatar_url          text,
    email_enabled       boolean                  default true              not null,
    push_enabled        boolean                  default true              not null,
    is_facebook_oauth   boolean                  default false             not null,
    is_tiktok_oauth     boolean                  default false             not null,
    oauth_metadata      jsonb,
    bank_account        text,
    bank_name           text,
    bank_account_holder text
);

create index if not exists idx_users_role
    on users (role);

create index if not exists idx_users_is_active
    on users (is_active);

create index if not exists idx_users_profile_data_gin
    on users using gin (profile_data);

create table if not exists logged_sessions
(
    id                 uuid                     default gen_random_uuid() not null
        primary key,
    user_id            uuid                                               not null
        references users
            on delete cascade,
    refresh_token_hash text                                               not null,
    device_fingerprint varchar(255)                                       not null,
    expiry_at          timestamp with time zone                           not null,
    is_revoked         boolean                  default false,
    last_used_at       timestamp with time zone,
    created_at         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at         timestamp with time zone
);

create index if not exists idx_logged_sessions_user_id
    on logged_sessions (user_id);

create index if not exists idx_logged_sessions_refresh_token_hash
    on logged_sessions (refresh_token_hash);

create table if not exists shipping_addresses
(
    id              uuid                     default gen_random_uuid() not null
        primary key,
    user_id         uuid                                               not null
        references users
            on delete cascade,
    type            varchar(50)                                        not null
        constraint shipping_addresses_type_check
            check ((type)::text = ANY
                   (ARRAY [('BILLING'::character varying)::text, ('SHIPPING'::character varying)::text])),
    full_name       varchar(255)                                       not null,
    phone_number    varchar(20),
    email           varchar(255),
    street          varchar(255)                                       not null,
    address_line2   varchar(255),
    city            varchar(255)                                       not null,
    postal_code     varchar(20)                                        not null,
    country         varchar(255),
    is_default      boolean                  default false,
    created_at      timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at      timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at      timestamp with time zone,
    ghn_province_id integer,
    ghn_district_id integer,
    ghn_ward_code   varchar(16),
    province_name   varchar(255),
    district_name   varchar(255),
    ward_name       varchar(255)
);

create index if not exists idx_shipping_addresses_user_id
    on shipping_addresses (user_id);

create table if not exists brands
(
    id                        uuid                     default gen_random_uuid() not null
        primary key,
    user_id                   uuid
        references users
            on delete restrict,
    name                      varchar(255)                                       not null,
    description               text,
    contact_email             varchar(255),
    contact_phone             varchar(20),
    website                   varchar(255),
    logo_url                  text,
    status                    varchar(50)                                        not null
        constraint brands_status_check
            check ((status)::text = ANY
                   (ARRAY [('ACTIVE'::character varying)::text, ('INACTIVE'::character varying)::text])),
    created_at                timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at                timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at                timestamp with time zone,
    address                   varchar(255),
    tax_number                varchar(100),
    representative_name       varchar(255),
    representative_role       varchar(100),
    representative_email      varchar(255),
    representative_phone      varchar(25),
    representative_citizen_id varchar(100)
);

create index if not exists idx_brands_name_trgm
    on brands using gin (name gin_trgm_ops);

create index if not exists idx_brands_status
    on brands (status)
    where (deleted_at IS NULL);

comment on index idx_brands_status is 'Optimizes active brand count queries and brand filtering in revenue calculations';

create table if not exists contracts
(
    id                                 uuid                     default gen_random_uuid()          not null
        primary key,
    brand_id                           uuid                                                        not null
        references brands,
    title                              varchar(255),
    type                               varchar(50)                                                 not null,
    start_date                         date,
    end_date                           date,
    status                             varchar(50)              default 'DRAFT'::character varying not null,
    contract_file_url                  text,
    proposal_file_url                  text,
    parent_contract_id                 uuid
                                                                                                   references contracts
                                                                                                       on delete set null,
    created_at                         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at                         timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at                         timestamp with time zone,
    brand_bank_name                    varchar(255),
    brand_account_number               varchar(255),
    contract_number                    varchar(100)                                                not null
        unique,
    representative_name                varchar(255),
    representative_role                varchar(100),
    representative_phone               varchar(20),
    representative_email               varchar(255),
    signed_date                        date,
    signed_location                    text,
    currency                           varchar(3)               default 'VND'::character varying,
    financial_terms                    jsonb                    default '{}'::jsonb                not null,
    scope_of_work                      jsonb                    default '{}'::jsonb                not null,
    legal_terms                        jsonb                    default '{}'::jsonb                not null,
    created_by                         uuid
                                                                                                   references users
                                                                                                       on delete set null,
    updated_by                         uuid
                                                                                                   references users
                                                                                                       on delete set null,
    representative_tax_number          varchar(100),
    representative_bank_name           varchar(100),
    representative_bank_account_number varchar(50),
    representative_bank_account_holder varchar(100),
    deposit_percent                    integer                  default 0
        constraint contracts_deposit_percent_check
            check ((deposit_percent >= 0) AND (deposit_percent <= 100)),
    deposit_amount                     numeric                  default 0
        constraint contracts_deposit_amount_check
            check (deposit_amount >= (0)::numeric),
    brand_account_holder               varchar(100),
    is_deposit_paid                    boolean                  default false,
    reject_reason                      text
);

comment on column contracts.title is 'Contract title/name for easy identification';

comment on column contracts.parent_contract_id is 'Reference to parent contract for amendments or related contracts';

comment on column contracts.brand_bank_name is 'Brand bank name for contract payments';

comment on column contracts.brand_account_number is 'Brand bank account number for contract payments';

comment on column contracts.representative_name is 'KOL/Influencer representative name';

comment on column contracts.representative_role is 'KOL/Influencer representative role';

comment on column contracts.representative_phone is 'KOL/Influencer representative phone';

comment on column contracts.representative_email is 'KOL/Influencer representative email';

create index if not exists idx_contracts_brand_id
    on contracts (brand_id);

create index if not exists idx_contracts_status_type
    on contracts (status, type);

create index if not exists idx_contracts_type_status
    on contracts (type, status)
    where (deleted_at IS NULL);

comment on index idx_contracts_type_status is 'Optimizes revenue breakdown queries by contract type (ADVERTISING, AFFILIATE, etc.)';

create table if not exists contract_payments
(
    id                     uuid                     default gen_random_uuid() not null
        primary key,
    contract_id            uuid                                               not null
        references contracts
            on delete cascade,
    installment_percentage numeric(5, 2)
        constraint contract_payments_installment_percentage_check
            check ((installment_percentage >= (0)::numeric) AND (installment_percentage <= (100)::numeric)),
    amount                 numeric(15, 2)                                     not null,
    status                 contract_payments_status                           not null,
    due_date               date                                               not null,
    paid_date              date,
    payment_method         varchar(50)
        constraint contract_payments_payment_method_check
            check ((payment_method)::text = ANY
                   (ARRAY [('BANK_TRANSFER'::character varying)::text, ('CASH'::character varying)::text, ('CHECK'::character varying)::text])),
    note                   text,
    created_at             timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at             timestamp with time zone default CURRENT_TIMESTAMP,
    created_by             uuid
                                                                              references users
                                                                                  on delete set null,
    updated_by             uuid
                                                                              references users
                                                                                  on delete set null,
    deleted_at             timestamp with time zone,
    is_deposit             boolean                  default false,
    period_start           timestamp with time zone,
    period_end             timestamp with time zone,
    calculated_at          timestamp with time zone,
    calculation_breakdown  jsonb,
    locked_amount          numeric(15, 2),
    locked_at              timestamp with time zone,
    locked_clicks          bigint,
    locked_revenue         numeric(15, 2),
    base_amount            numeric(15, 2)           default 0,
    performance_amount     numeric(15, 2)           default 0
);

comment on column contract_payments.period_start is 'Start of the payment period (inclusive) for AFFILIATE/CO_PRODUCING contracts';

comment on column contract_payments.period_end is 'End of the payment period (exclusive) for AFFILIATE/CO_PRODUCING contracts';

comment on column contract_payments.calculated_at is 'Timestamp of last payment amount calculation';

comment on column contract_payments.calculation_breakdown is 'Detailed breakdown of payment calculation (tier breakdown for AFFILIATE, revenue breakdown for CO_PRODUCING)';

comment on column contract_payments.locked_amount is 'Locked payment amount when payment link was created';

comment on column contract_payments.locked_at is 'Timestamp when payment amount was locked for payment processing';

comment on column contract_payments.locked_clicks is 'Locked total clicks count at time of locking (AFFILIATE contracts only)';

comment on column contract_payments.locked_revenue is 'Locked total revenue at time of locking (CO_PRODUCING contracts only)';

create index if not exists idx_contract_payments_contract_id
    on contract_payments (contract_id);

create index if not exists idx_contract_payments_period
    on contract_payments (contract_id, period_start, period_end)
    where (deleted_at IS NULL);

create index if not exists idx_contract_payments_locked
    on contract_payments (locked_at)
    where ((locked_at IS NOT NULL) AND (deleted_at IS NULL));

create index if not exists idx_contract_payments_status_due_date
    on contract_payments (status, due_date)
    where (deleted_at IS NULL);

comment on index idx_contract_payments_status_due_date is 'Optimizes revenue queries filtering by payment status and due date for time-range analytics';

create table if not exists tags
(
    id          uuid                     default gen_random_uuid() not null
        primary key,
    name        varchar(255)                                       not null
        unique,
    description text,
    usage_count integer                  default 0,
    created_at  timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at  timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at  timestamp with time zone,
    created_by  uuid
        references users,
    updated_by  uuid
        references users
);

create index if not exists idx_tags_name_trgm
    on tags using gin (name gin_trgm_ops);

create table if not exists channels
(
    id                       uuid                     default gen_random_uuid() not null
        primary key,
    name                     varchar(50)                                        not null,
    description              text,
    created_at               timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at               timestamp with time zone default CURRENT_TIMESTAMP,
    is_active                boolean                  default true,
    deleted_at               timestamp with time zone,
    home_page_url            text,
    external_id              varchar(255),
    account_name             varchar(255),
    hashed_access_token      text,
    hashed_refresh_token     text,
    access_token_expires_at  timestamp with time zone,
    refresh_token_expires_at timestamp with time zone,
    last_synced_at           timestamp with time zone,
    code                     varchar(100),
    vault_path               varchar(255),
    metrics                  jsonb                    default '{}'::jsonb
);

comment on column channels.metrics is 'JSONB column storing page/user level metrics from social platforms (Facebook fan_count, TikTok followers, etc.)';

create index if not exists idx_channels_name
    on channels (name);

create index if not exists idx_channels_metrics
    on channels using gin (metrics);

create table if not exists product_categories
(
    id                 uuid                     default gen_random_uuid() not null
        primary key,
    name               varchar(255)                                       not null,
    description        text,
    parent_category_id uuid
                                                                          references product_categories
                                                                              on delete set null,
    created_at         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at         timestamp with time zone,
    icon_url           text
);

create index if not exists idx_product_categories_parent_id
    on product_categories (parent_category_id);

create table if not exists variant_attributes
(
    id          uuid                     default gen_random_uuid() not null
        primary key,
    ingredient  varchar(255)                                       not null,
    description text,
    created_at  timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at  timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at  timestamp with time zone,
    created_by  uuid
                                                                   references users
                                                                       on delete set null,
    updated_by  uuid
                                                                   references users
                                                                       on delete set null
);

create table if not exists carts
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    user_id    uuid
        references users
            on delete cascade,
    created_at timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at timestamp with time zone default CURRENT_TIMESTAMP
);

create index if not exists idx_carts_user_id
    on carts (user_id);

create table if not exists files
(
    id           uuid                     default gen_random_uuid()            not null
        primary key,
    file_name    varchar(255)                                                  not null,
    alt_text     varchar(255),
    url          text,
    mime_type    varchar(100)                                                  not null,
    size         bigint                                                        not null,
    uploaded_at  timestamp with time zone default CURRENT_TIMESTAMP,
    uploaded_by  uuid
                                                                               references users
                                                                                   on delete set null,
    storage_key  text                     default ''::text                     not null,
    status       varchar(50)              default 'PENDING'::character varying not null,
    error_reason text,
    created_at   timestamp with time zone default now(),
    updated_at   timestamp with time zone default now(),
    deleted_at   timestamp with time zone,
    metadata     jsonb
);

create index if not exists idx_files_uploaded_by
    on files (uploaded_by);

create index if not exists idx_files_deleted_at
    on files (deleted_at);

create index if not exists idx_files_status
    on files (status)
    where (deleted_at IS NULL);

create index if not exists idx_files_storage_key
    on files (storage_key)
    where (deleted_at IS NULL);

create table if not exists modified_histories
(
    id             uuid                     default gen_random_uuid()                not null
        primary key,
    reference_id   uuid,
    reference_type varchar(50)                                                       not null
        constraint modified_histories_reference_type_chk
            check ((reference_type)::text = ANY
                   (ARRAY [('CONTRACT'::character varying)::text, ('CAMPAIGN'::character varying)::text, ('MILESTONE'::character varying)::text, ('TASK'::character varying)::text, ('CONTENT'::character varying)::text, ('PRODUCT'::character varying)::text, ('BLOG'::character varying)::text])),
    operation      varchar(50)                                                       not null
        constraint modified_histories_operation_chk
            check ((operation)::text = ANY
                   (ARRAY [('CREATE'::character varying)::text, ('UPDATE'::character varying)::text, ('DELETE'::character varying)::text])),
    description    text                                                              not null,
    changed_by     uuid
                                                                                     references users
                                                                                         on delete set null,
    changed_at     timestamp with time zone default CURRENT_TIMESTAMP,
    status         varchar(20)              default 'IN_PROGRESS'::character varying not null
        constraint modified_histories_status_check
            check ((status)::text = ANY
                   (ARRAY [('IN_PROGRESS'::character varying)::text, ('COMPLETED'::character varying)::text, ('FAILED'::character varying)::text]))
);

create index if not exists idx_modified_histories_changed_by
    on modified_histories (changed_by);

create index if not exists idx_modified_histories_poly_ref
    on modified_histories (reference_id, reference_type);

create table if not exists concepts
(
    id              uuid                     default gen_random_uuid() not null
        primary key,
    name            varchar(255)                                       not null,
    description     text,
    created_at      timestamp with time zone default now()             not null,
    updated_at      timestamp with time zone default now()             not null,
    banner_url      text,
    video_thumbnail text
);

create table if not exists notifications
(
    id                uuid                     default gen_random_uuid() not null
        constraint notifications_new_pkey
            primary key,
    user_id           uuid                                               not null
        constraint fk_notifications_user
            references users
            on update cascade on delete cascade,
    type              varchar(50)                                        not null,
    status            varchar(50)                                        not null,
    delivery_attempts jsonb                    default '[]'::jsonb       not null,
    recipient_info    jsonb                                              not null,
    content_data      jsonb                                              not null,
    platform_config   jsonb,
    error_details     jsonb,
    created_at        timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at        timestamp with time zone,
    is_read           boolean                  default false             not null,
    severity          notification_severity    default 'INFO'::notification_severity
);

comment on table notifications is 'Stores all notification attempts (email and push) with flexible JSONB metadata';

comment on column notifications.type is 'Notification type: EMAIL, PUSH, or IN_APP';

comment on column notifications.status is 'Delivery status: PENDING, SENT, FAILED, RETRYING';

comment on column notifications.delivery_attempts is 'Array of delivery attempts with timestamps and results';

comment on column notifications.recipient_info is 'Email address or FCM device tokens';

comment on column notifications.content_data is 'Notification content (subject, body, template data)';

comment on column notifications.platform_config is 'iOS/Android specific push notification settings';

comment on column notifications.error_details is 'Last error information if delivery failed';

create index if not exists idx_notifications_type
    on notifications (type);

create index if not exists idx_notifications_created_at
    on notifications (created_at);

create index if not exists idx_notifications_deleted_at
    on notifications (deleted_at);

create index if not exists idx_notifications_delivery_attempts
    on notifications using gin (delivery_attempts);

create index if not exists idx_notifications_recipient_info
    on notifications using gin (recipient_info);

create index if not exists idx_notifications_error_details
    on notifications using gin (error_details);

create index if not exists idx_notifications_is_read
    on notifications (is_read)
    where (is_read = false);

create table if not exists device_tokens
(
    id                uuid                     default gen_random_uuid() not null
        primary key,
    user_id           uuid                                               not null
        constraint fk_device_tokens_user
            references users
            on update cascade on delete cascade,
    token             varchar(255)                                       not null,
    platform          varchar(50)                                        not null,
    registered_at     timestamp with time zone default CURRENT_TIMESTAMP not null,
    last_used_at      timestamp with time zone,
    is_valid          boolean                  default true              not null,
    created_at        timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at        timestamp with time zone,
    logged_session_id uuid
        references logged_sessions
            on delete cascade
);

comment on table device_tokens is 'Stores FCM device tokens for push notifications';

comment on column device_tokens.token is 'Firebase Cloud Messaging device token';

comment on column device_tokens.platform is 'Mobile platform: IOS or ANDROID';

comment on column device_tokens.last_used_at is 'Last time a notification was sent to this token';

comment on column device_tokens.is_valid is 'Whether token is still valid (false if FCM reports invalid)';

create index if not exists idx_device_tokens_user_id
    on device_tokens (user_id);

create unique index if not exists idx_device_tokens_token
    on device_tokens (token)
    where (deleted_at IS NULL);

create index if not exists idx_device_tokens_is_valid
    on device_tokens (is_valid);

create index if not exists idx_device_tokens_last_used_at
    on device_tokens (last_used_at);

create index if not exists idx_device_tokens_deleted_at
    on device_tokens (deleted_at);

create index if not exists idx_device_tokens_logged_session_id
    on device_tokens (logged_session_id);

create table if not exists provinces
(
    id             integer                                not null
        primary key,
    name           varchar(255)                           not null,
    country_id     integer,
    code           varchar(64),
    region_id      integer,
    region_cpn     integer,
    is_enable      integer,
    can_update_cod boolean,
    status         integer,
    created_at     timestamp with time zone default now() not null,
    updated_at     timestamp with time zone default now() not null,
    deleted_at     timestamp with time zone
);

create index if not exists idx_provinces_name
    on provinces (name);

create index if not exists idx_provinces_deleted_at
    on provinces (deleted_at);

create table if not exists districts
(
    id              integer                                not null
        primary key,
    province_id     integer                                not null
        references provinces
            on update cascade on delete cascade,
    name            varchar(255)                           not null,
    code            varchar(64),
    type            integer,
    support_type    integer,
    pick_type       integer,
    deliver_type    integer,
    government_code varchar(64),
    is_enable       integer,
    can_update_cod  boolean,
    status          integer,
    created_at      timestamp with time zone default now() not null,
    updated_at      timestamp with time zone default now() not null,
    deleted_at      timestamp with time zone
);

create index if not exists idx_districts_province_id
    on districts (province_id);

create index if not exists idx_districts_name
    on districts (name);

create index if not exists idx_districts_deleted_at
    on districts (deleted_at);

create table if not exists wards
(
    code            varchar(32)                            not null
        primary key,
    district_id     integer                                not null
        references districts
            on update cascade on delete cascade,
    name            varchar(255)                           not null,
    support_type    integer,
    pick_type       integer,
    deliver_type    integer,
    government_code varchar(64),
    is_enable       integer,
    can_update_cod  boolean,
    status          integer,
    created_at      timestamp with time zone default now() not null,
    updated_at      timestamp with time zone default now() not null,
    deleted_at      timestamp with time zone
);

create index if not exists idx_wards_district_id
    on wards (district_id);

create index if not exists idx_wards_name
    on wards (name);

create index if not exists idx_wards_deleted_at
    on wards (deleted_at);

create table if not exists configs
(
    id          uuid                     default gen_random_uuid()    not null
        primary key,
    key         varchar(255)                                          not null
        unique,
    value       text                                                  not null,
    description text,
    created_at  timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at  timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at  timestamp with time zone,
    value_type  value_type               default 'STRING'::value_type not null,
    updated_by  uuid
                                                                      references users
                                                                          on delete set null
);

create table if not exists campaigns
(
    id            uuid                     default gen_random_uuid() not null
        primary key,
    contract_id   uuid                                               not null
        references contracts
            on delete cascade,
    name          varchar(255)                                       not null,
    description   text,
    start_date    date                                               not null,
    end_date      date                                               not null,
    status        campaign_status                                    not null,
    type          varchar(50)                                        not null,
    created_at    timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at    timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at    timestamp with time zone,
    created_by    uuid
                                                                     references users
                                                                         on delete set null,
    updated_by    uuid
                                                                     references users
                                                                         on delete set null,
    reject_reason text
);

create index if not exists idx_campaigns_contract_id
    on campaigns (contract_id);

create index if not exists idx_campaigns_status
    on campaigns (status);

create index if not exists idx_campaigns_status_end_date
    on campaigns (status, end_date)
    where (deleted_at IS NULL);

comment on index idx_campaigns_status_end_date is 'Optimizes queries for campaigns approaching deadline, filtered by status and end date';

create table if not exists milestones
(
    id                    uuid                     default gen_random_uuid() not null
        primary key,
    campaign_id           uuid                                               not null
        references campaigns
            on delete cascade,
    description           text                                               not null,
    due_date              date                                               not null,
    completed_at          timestamp with time zone,
    completion_percentage integer                  default 0
        constraint milestones_completion_percent_check
            check ((completion_percentage >= 0) AND (completion_percentage <= 100)),
    status                varchar(50)                                        not null
        constraint milestones_status_check
            check ((status)::text = ANY
                   (ARRAY [('NOT_STARTED'::character varying)::text, ('ON_GOING'::character varying)::text, ('CANCELLED'::character varying)::text, ('COMPLETED'::character varying)::text])),
    behind_schedule       boolean                  default false,
    created_at            timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at            timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at            timestamp with time zone,
    created_by            uuid
                                                                             references users
                                                                                 on delete set null,
    updated_by            uuid
                                                                             references users
                                                                                 on delete set null
);

create index if not exists idx_milestones_campaign_id
    on milestones (campaign_id);

create table if not exists tasks
(
    id                    uuid                     default gen_random_uuid() not null
        primary key,
    milestone_id          uuid                                               not null
        references milestones
            on delete cascade,
    name                  varchar(255)                                       not null,
    description           jsonb,
    deadline              date                                               not null,
    type                  varchar(50)                                        not null,
    status                varchar(50)                                        not null
        constraint tasks_status_check
            check ((status)::text = ANY
                   (ARRAY [('TODO'::character varying)::text, ('IN_PROGRESS'::character varying)::text, ('CANCELLED'::character varying)::text, ('SUBMITTED'::character varying)::text, ('REVISION_REQUESTED'::character varying)::text, ('APPROVED'::character varying)::text, ('ON_RELEASE'::character varying)::text, ('RECAP'::character varying)::text, ('DONE'::character varying)::text])),
    assigned_to           uuid
                                                                             references users
                                                                                 on delete set null,
    created_at            timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at            timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at            timestamp with time zone,
    created_by            uuid
                                                                             references users
                                                                                 on delete set null,
    updated_by            uuid
                                                                             references users
                                                                                 on delete set null,
    scope_of_work_item_id varchar(150)
);

create index if not exists idx_tasks_milestone_id
    on tasks (milestone_id);

create index if not exists idx_tasks_assigned_to
    on tasks (assigned_to);

create index if not exists idx_tasks_status
    on tasks (status);

create table if not exists contents
(
    id                 uuid                     default gen_random_uuid()       not null
        primary key,
    task_id            uuid
                                                                                references tasks
                                                                                    on delete set null,
    title              varchar(255)                                             not null,
    type               content_type                                             not null,
    body               jsonb                                                    not null,
    publish_date       timestamp with time zone,
    status             content_status           default 'DRAFT'::content_status not null,
    ai_generated_text  text,
    created_at         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at         timestamp with time zone,
    created_by         uuid
                                                                                references users
                                                                                    on delete set null,
    updated_by         uuid
                                                                                references users
                                                                                    on delete set null,
    thumbnail_url      text,
    rejection_feedback text,
    description        text,
    tags               jsonb                    default '[]'::jsonb
);

create index if not exists idx_contents_task_id
    on contents (task_id);

create index if not exists idx_contents_title_trgm
    on contents using gin (title gin_trgm_ops);

create index if not exists idx_contents_body_gin
    on contents using gin (body);

create index if not exists idx_contents_status
    on contents (status);

create table if not exists blogs
(
    content_id uuid                                               not null
        primary key
        references contents
            on delete cascade,
    author_id  uuid                                               not null
        references users
            on delete set null,
    tags       jsonb,
    excerpt    text,
    read_time  integer,
    created_by uuid
        references users,
    updated_by uuid
        references users,
    created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
    updated_at timestamp with time zone
);

create index if not exists idx_blogs_author_id
    on blogs (author_id);

create table if not exists blog_tags
(
    blog_id uuid not null
        references blogs
            on delete cascade,
    tag_id  uuid not null
        references tags
            on delete cascade,
    primary key (blog_id, tag_id)
);

create index if not exists idx_blog_tags_tag_id
    on blog_tags (tag_id);

create table if not exists products
(
    id             uuid                     default gen_random_uuid()          not null
        primary key,
    brand_id       uuid
                                                                               references brands
                                                                                   on delete set null,
    category_id    uuid
                                                                               references product_categories
                                                                                   on delete set null,
    name           varchar(255)                                                not null,
    description    text                                                        not null,
    type           varchar(50)                                                 not null
        constraint products_type_check
            check ((type)::text = ANY
                   (ARRAY [('STANDARD'::character varying)::text, ('LIMITED'::character varying)::text])),
    created_at     timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at     timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at     timestamp with time zone,
    task_id        uuid
                                                                               references tasks
                                                                                   on delete set null,
    status         varchar(50)              default 'DRAFT'::character varying not null
        constraint products_status_check
            check ((status)::text = ANY
                   (ARRAY [('DRAFT'::character varying)::text, ('SUBMITTED'::character varying)::text, ('REVISION'::character varying)::text, ('APPROVED'::character varying)::text, ('ACTIVED'::character varying)::text, ('INACTIVED'::character varying)::text])),
    is_active      boolean                  default false                      not null,
    created_by     uuid
                                                                               references users
                                                                                   on delete set null,
    updated_by     uuid
                                                                               references users
                                                                                   on delete set null,
    average_rating numeric(3, 2)            default 0
);

create index if not exists idx_products_brand_id
    on products (brand_id);

create index if not exists idx_products_type
    on products (type);

create index if not exists idx_products_name_trgm
    on products using gin (name gin_trgm_ops);

create table if not exists product_variants
(
    id                uuid                     default gen_random_uuid() not null
        primary key,
    product_id        uuid                                               not null
        references products
            on delete cascade,
    price             numeric(15, 2),
    current_stock     integer,
    capacity          numeric(10, 2),
    capacity_unit     varchar(20),
    container_type    varchar(50),
    dispenser_type    varchar(50),
    uses              varchar(255),
    manufactring_date date,
    expiry_date       date,
    instructions      text,
    is_default        boolean                  default false,
    created_at        timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at        timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at        timestamp with time zone,
    is_active         boolean                  default true              not null,
    created_by        uuid
                                                                         references users
                                                                             on delete set null,
    updated_by        uuid
                                                                         references users
                                                                             on delete set null,
    weight            integer,
    height            integer,
    length            integer,
    width             integer,
    pre_order_limit   integer,
    pre_order_count   integer,
    max_stock         integer
);

comment on column product_variants.weight is 'in grams';

comment on column product_variants.height is 'in centimeters';

comment on column product_variants.length is 'in centimeters';

comment on column product_variants.width is 'in centimeters';

create index if not exists idx_product_variants_product_id
    on product_variants (product_id);

create table if not exists product_stories
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    variant_id uuid                                               not null
        references product_variants
            on delete cascade,
    content    jsonb                                              not null,
    created_at timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone
);

create index if not exists idx_product_stories_variant_id
    on product_stories (variant_id);

create index if not exists idx_product_stories_content_gin
    on product_stories using gin (content);

create table if not exists variant_images
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    variant_id uuid                                               not null
        references product_variants
            on delete cascade,
    image_url  text                                               not null,
    alt_text   varchar(255),
    is_primary boolean                  default false,
    created_at timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at timestamp with time zone
);

create index if not exists idx_variant_images_variant_id
    on variant_images (variant_id);

create table if not exists variant_attribute_values
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    variant_id   uuid                                               not null
        references product_variants
            on delete cascade,
    attribute_id uuid                                               not null
        references variant_attributes
            on delete cascade,
    value        numeric(10, 2)                                     not null,
    unit         varchar(50),
    created_at   timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at   timestamp with time zone,
    unique (variant_id, attribute_id)
);

create index if not exists idx_variant_attr_vals_variant_id
    on variant_attribute_values (variant_id);

create index if not exists idx_variant_attr_vals_attribute_id
    on variant_attribute_values (attribute_id);

create table if not exists content_products
(
    id            uuid default gen_random_uuid() not null
        primary key,
    content_id    uuid                           not null
        references contents
            on delete cascade,
    product_id    uuid                           not null
        references products
            on delete cascade,
    affiliate_url varchar(255)
);

create index if not exists idx_content_products_content_id
    on content_products (content_id);

create index if not exists idx_content_products_product_id
    on content_products (product_id);

create table if not exists cart_items
(
    id         uuid                     default gen_random_uuid() not null
        primary key,
    cart_id    uuid                                               not null
        references carts
            on delete cascade,
    variant_id uuid                                               not null
        references product_variants
            on delete cascade,
    quantity   integer                                            not null,
    subtotal   numeric(15, 2)                                     not null,
    updated_at timestamp with time zone default CURRENT_TIMESTAMP
);

create index if not exists idx_cart_items_cart_id
    on cart_items (cart_id);

create index if not exists idx_cart_items_variant_id
    on cart_items (variant_id);

create table if not exists orders
(
    id                       uuid                     default gen_random_uuid()             not null
        primary key,
    user_id                  uuid                                                           not null
        references users
            on delete set null,
    status                   order_status                                                   not null,
    total_amount             numeric(15, 2)                                                 not null,
    created_at               timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at               timestamp with time zone default CURRENT_TIMESTAMP,
    full_name                varchar(255),
    phone_number             varchar(20),
    email                    varchar(255),
    street                   varchar(255),
    address_line2            varchar(255),
    city                     varchar(255),
    ghn_province_id          integer,
    ghn_district_id          integer,
    ghn_ward_code            varchar(16),
    province_name            varchar(255),
    district_name            varchar(255),
    ward_name                varchar(255),
    shipping_fee             integer                  default 0,
    action_notes             jsonb,
    user_note                text,
    ghn_order_code           text,
    is_self_picked_up        boolean                  default false                         not null,
    user_resource            text,
    order_type               varchar(50)              default 'STANDARD'::character varying not null,
    confirmation_image       text,
    staff_resource           text,
    deleted_at               timestamp with time zone,
    user_bank_account        text,
    user_bank_name           text,
    user_bank_account_holder text
);

create index if not exists idx_orders_user_id
    on orders (user_id);

create index if not exists idx_orders_status
    on orders (status);

create index if not exists idx_orders_status_created_at
    on orders (status, created_at);

comment on index idx_orders_status_created_at is 'Optimizes standard product revenue queries filtering by PAID status and order date';

create table if not exists order_items
(
    id                     uuid    default gen_random_uuid() not null
        primary key,
    order_id               uuid                              not null
        references orders
            on delete cascade,
    variant_id             uuid                              not null
        references product_variants
            on delete cascade,
    quantity               integer                           not null,
    subtotal               numeric(15, 2)                    not null,
    unit_price             numeric(15, 2)                    not null,
    capacity               numeric(10, 2),
    capacity_unit          varchar(20),
    container_type         varchar(50),
    dispenser_type         varchar(50),
    uses                   varchar(255),
    manufacturing_date     date,
    expiry_date            date,
    instructions           text,
    attributes_description jsonb,
    weight                 integer,
    height                 integer,
    length                 integer,
    width                  integer,
    product_name           text,
    description            text,
    product_type           text,
    brand_id               uuid
                                                             references brands
                                                                 on delete set null,
    category_id            uuid
                                                             references product_categories
                                                                 on delete set null,
    is_review              boolean default false
);

create index if not exists idx_order_items_order_id
    on order_items (order_id);

create index if not exists idx_order_items_variant_id
    on order_items (variant_id);

create table if not exists pre_orders
(
    id                       uuid                     default gen_random_uuid() not null
        primary key,
    user_id                  uuid                                               not null
        references users
            on delete set null,
    variant_id               uuid                                               not null
        references product_variants
            on delete cascade,
    quantity                 integer                                            not null,
    unit_price               numeric(15, 2)                                     not null,
    total_amount             numeric(15, 2)                                     not null,
    status                   pre_order_status                                   not null,
    created_at               timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at               timestamp with time zone default CURRENT_TIMESTAMP,
    capacity                 numeric(10, 2),
    capacity_unit            varchar(20),
    container_type           varchar(50),
    dispenser_type           varchar(50),
    uses                     text,
    manufacturing_date       date,
    expiry_date              date,
    instructions             text,
    attributes_description   jsonb,
    weight                   integer,
    height                   integer,
    length                   integer,
    width                    integer,
    full_name                varchar(255),
    phone_number             varchar(20),
    email                    varchar(255),
    street                   text,
    address_line2            text,
    city                     varchar(100),
    ghn_province_id          integer,
    ghn_district_id          integer,
    ghn_ward_code            varchar(50),
    province_name            varchar(100),
    district_name            varchar(100),
    ward_name                varchar(100),
    is_self_picked_up        boolean                  default false             not null,
    confirmation_image       text,
    user_note                text,
    action_notes             jsonb,
    user_resource            text,
    staff_resource           text,
    deleted_at               timestamp with time zone,
    user_bank_account        text,
    user_bank_name           text,
    user_bank_account_holder text,
    product_name             text,
    description              text,
    product_type             varchar(255),
    brand_id                 uuid
                                                                                references brands
                                                                                    on delete set null,
    category_id              uuid
                                                                                references product_categories
                                                                                    on delete set null,
    is_review                boolean                  default false             not null
);

create index if not exists idx_pre_orders_user_id
    on pre_orders (user_id);

create index if not exists idx_pre_orders_variant_id
    on pre_orders (variant_id);

create index if not exists idx_pre_orders_status
    on pre_orders (status);

create table if not exists refund_requests
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    order_id     uuid
        references orders
            on delete cascade,
    pre_order_id uuid
        references pre_orders
            on delete cascade,
    reason       text                                               not null,
    amount       numeric(15, 2)                                     not null,
    status       varchar(50)                                        not null
        constraint refund_requests_status_check
            check ((status)::text = ANY
                   (ARRAY [('PENDING'::character varying)::text, ('APPROVED'::character varying)::text, ('REJECTED'::character varying)::text, ('COMPLETED'::character varying)::text])),
    requested_at timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP,
    unique (order_id, pre_order_id),
    constraint refund_one_of_order_or_preorder
        check (((order_id IS NOT NULL) AND (pre_order_id IS NULL)) OR
               ((order_id IS NULL) AND (pre_order_id IS NOT NULL)))
);

create index if not exists idx_refund_requests_order_id
    on refund_requests (order_id);

create index if not exists idx_refund_requests_pre_order_id
    on refund_requests (pre_order_id);

create index if not exists idx_refund_requests_status
    on refund_requests (status);

create table if not exists payment_transactions
(
    id               uuid                     default gen_random_uuid() not null
        primary key,
    reference_id     uuid                                               not null,
    reference_type   varchar(50)                                        not null,
    amount           numeric(15, 2)                                     not null,
    method           varchar(50)                                        not null,
    status           payment_transactions_status                        not null,
    transaction_date timestamp with time zone default CURRENT_TIMESTAMP,
    gateway_ref      varchar(255),
    updated_at       timestamp with time zone default CURRENT_TIMESTAMP,
    gateway_id       text,
    payos_metadata   jsonb,
    created_at       timestamp with time zone default CURRENT_TIMESTAMP,
    payer_id         uuid
);

comment on column payment_transactions.payos_metadata is 'Stores PayOS-specific payment data including payment_link_id, order_code, checkout_url, qr_code, expiry, and transaction details';

create index if not exists idx_payment_transactions_ref
    on payment_transactions (reference_id, reference_type);

create index if not exists idx_payment_transactions_payos_metadata
    on payment_transactions using gin (payos_metadata);

create index if not exists idx_payment_transactions_status_method_updated
    on payment_transactions (status, method, updated_at)
    where ((status = 'PENDING'::payment_transactions_status) AND ((method)::text = 'PAYOS'::text));

create index if not exists idx_payment_transactions_payos_order_code
    on payment_transactions ((payos_metadata ->> 'order_code'::text))
    where (payos_metadata IS NOT NULL);

create table if not exists kpi_metrics
(
    id             uuid default gen_random_uuid() not null,
    reference_id   uuid                           not null,
    reference_type reference_type                 not null,
    type           varchar(50)                    not null,
    value          numeric(15, 2)                 not null,
    recorded_date  timestamp with time zone       not null,
    unit           varchar(10),
    primary key (id, recorded_date)
);

create index if not exists idx_kpi_metrics_ref
    on kpi_metrics (reference_id, reference_type);

create index if not exists kpi_metrics_recorded_date_idx
    on kpi_metrics (recorded_date desc);

create index if not exists idx_kpi_metrics_reference_type_id
    on kpi_metrics (reference_type, reference_id);

create table if not exists limited_products
(
    id                      uuid not null
        primary key
        references products
            on delete cascade,
    premiere_date           timestamp with time zone,
    availability_start_date timestamp with time zone,
    availability_end_date   timestamp with time zone,
    concept_id              uuid
        constraint limited_products_concept_id_unique
            unique
                                 references concepts
                                     on delete set null,
    achievable_quantity     integer
);

create table if not exists affiliate_links
(
    id            uuid                     default gen_random_uuid()           not null
        primary key,
    hash          varchar(16)                                                  not null
        unique,
    contract_id   uuid
        references contracts
            on delete cascade,
    content_id    uuid
        references contents
            on delete cascade,
    channel_id    uuid
        references channels
            on delete restrict,
    tracking_url  text                                                         not null,
    status        varchar(20)              default 'active'::character varying not null
        constraint affiliate_links_status_check
            check ((status)::text = ANY
                   (ARRAY [('active'::character varying)::text, ('inactive'::character varying)::text, ('expired'::character varying)::text])),
    created_at    timestamp with time zone default now(),
    updated_at    timestamp with time zone default now(),
    deleted_at    timestamp with time zone,
    affiliate_url text                     default ''::text                    not null,
    metadata      jsonb                    default '{}'::jsonb
);

comment on table affiliate_links is 'Stores unique trackable affiliate links for content+channel combinations';

comment on column affiliate_links.hash is 'Base62 SHA-256 hash (16 chars) for public URL shortening';

comment on column affiliate_links.tracking_url is 'Original affiliate product URL from contract ScopeOfWork';

comment on column affiliate_links.status is 'active: clickable, inactive: paused, expired: contract/content ended';

comment on column affiliate_links.metadata is 'Flexible storage for additional context (e.g. campaign_id, user_id) when standard relations are not used';

create index if not exists idx_affiliate_links_contract_id
    on affiliate_links (contract_id)
    where (deleted_at IS NULL);

create index if not exists idx_affiliate_links_content_id
    on affiliate_links (content_id)
    where (deleted_at IS NULL);

create index if not exists idx_affiliate_links_channel_id
    on affiliate_links (channel_id)
    where (deleted_at IS NULL);

create index if not exists idx_affiliate_links_status
    on affiliate_links (status)
    where (deleted_at IS NULL);

create index if not exists idx_affiliate_links_deleted_at
    on affiliate_links (deleted_at);

create table if not exists content_channels
(
    id                 uuid                     default gen_random_uuid() not null
        primary key,
    content_id         uuid                                               not null
        references contents
            on delete cascade,
    channel_id         uuid                                               not null
        references channels
            on delete cascade,
    post_date          timestamp with time zone,
    auto_post_status   auto_post_status,
    created_at         timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at         timestamp with time zone default CURRENT_TIMESTAMP,
    external_post_id   varchar(255),
    metrics            jsonb,
    last_error         text,
    published_at       timestamp with time zone,
    external_post_url  text,
    affiliate_link_id  uuid
                                                                          references affiliate_links
                                                                              on delete set null,
    external_post_type external_post_type       default 'TEXT'::external_post_type,
    metadata           jsonb
);

create index if not exists idx_content_channels_content_id
    on content_channels (content_id);

create index if not exists idx_content_channels_channel_id
    on content_channels (channel_id);

create table if not exists click_events
(
    id                uuid                     default gen_random_uuid() not null,
    affiliate_link_id uuid                                               not null
        references affiliate_links
            on delete cascade,
    user_id           uuid
                                                                         references users
                                                                             on delete set null,
    clicked_at        timestamp with time zone default now()             not null,
    ip_address        inet,
    user_agent        text,
    referrer_url      text,
    session_id        varchar(255),
    primary key (id, clicked_at)
);

comment on table click_events is 'TimescaleDB hypertable storing individual click events with 90-day retention';

comment on column click_events.clicked_at is 'Partition key for TimescaleDB - DO NOT UPDATE after insert';

comment on column click_events.ip_address is 'Anonymized for privacy - store hashed or truncated version';

comment on column click_events.user_agent is 'Browser user agent for bot detection';

create index if not exists click_events_clicked_at_idx
    on click_events (clicked_at desc);

create index if not exists idx_click_events_affiliate_link_id
    on click_events (affiliate_link_id asc, clicked_at desc);

create index if not exists idx_click_events_user_id
    on click_events (user_id asc, clicked_at desc)
    where (user_id IS NOT NULL);

create index if not exists idx_click_events_session_id
    on click_events (session_id asc, clicked_at desc)
    where (session_id IS NOT NULL);

create table if not exists product_reviews
(
    id            uuid default gen_random_uuid() not null
        constraint product_reviews_pkey1
            primary key,
    product_id    uuid                           not null
        constraint fk_product_reviews_product
            references products
            on delete cascade,
    user_id       uuid
        constraint fk_product_reviews_user
            references users
            on delete set null,
    order_item_id uuid
        constraint fk_product_reviews_order_item
            references order_items
            on delete set null,
    pre_order_id  uuid
        constraint fk_product_reviews_preorder
            references pre_orders
            on delete set null,
    rating_stars  integer                        not null
        constraint product_reviews_rating_stars_check1
            check ((rating_stars >= 1) AND (rating_stars <= 5)),
    comment       text,
    assets_url    text,
    created_at    timestamp with time zone       not null,
    updated_at    timestamp with time zone       not null,
    deleted_at    timestamp with time zone
);

create trigger trg_update_product_average_rating
    after insert or update or delete
    on product_reviews
    for each row
execute procedure update_product_average_rating();

create table if not exists webhook_data
(
    id            uuid                     default gen_random_uuid() not null
        primary key,
    source        varchar(50)                                        not null
        constraint chk_webhook_source
            check ((source)::text = ANY
                   ((ARRAY ['payos'::character varying, 'facebook'::character varying, 'tiktok'::character varying, 'ghn'::character varying, 'other'::character varying])::text[])),
    event_type    varchar(100),
    external_id   varchar(255),
    raw_query     jsonb,
    raw_payload   jsonb                                              not null,
    processed     boolean                  default false,
    processed_at  timestamp with time zone,
    error_message text,
    created_at    timestamp with time zone default now()
);

comment on table webhook_data is 'Stores raw webhook payloads for audit and debugging purposes';

comment on column webhook_data.source is 'Source of the webhook: payos, facebook, tiktok, ghn, other';

comment on column webhook_data.event_type is 'Type of event from the webhook source';

comment on column webhook_data.external_id is 'External reference ID from the webhook source';

comment on column webhook_data.raw_query is 'Raw query string received with the webhook, if applicable';

comment on column webhook_data.raw_payload is 'Complete raw JSON payload received from webhook';

comment on column webhook_data.processed is 'Whether this webhook has been successfully processed';

comment on column webhook_data.processed_at is 'Timestamp when the webhook was processed';

comment on column webhook_data.error_message is 'Error message if webhook processing failed';

create index if not exists idx_webhook_data_source
    on webhook_data (source);

create index if not exists idx_webhook_data_external_id
    on webhook_data (external_id);

create index if not exists idx_webhook_data_created_at
    on webhook_data (created_at desc);

create index if not exists idx_webhook_data_processed
    on webhook_data (processed)
    where (processed = false);

create index if not exists idx_webhook_data_raw_payload
    on webhook_data using gin (raw_payload);

create table if not exists content_comments
(
    id                 uuid                     default gen_random_uuid() not null
        primary key,
    content_channel_id uuid                                               not null
        references content_channels
            on delete cascade,
    comment            text                                               not null,
    reactions          jsonb                                              not null,
    created_at         timestamp with time zone default now(),
    created_by         uuid                                               not null
        references users
            on delete set null,
    updated_at         timestamp with time zone default now(),
    updated_by         uuid
                                                                          references users
                                                                              on delete set null,
    is_censored        boolean                  default false,
    censor_reason      text
);

comment on table content_comments is 'Stores comments for content';

comment on column content_comments.content_channel_id is 'Reference to the content channel being commented on';

comment on column content_comments.comment is 'The text of the comment';

comment on column content_comments.reactions is 'JSONB field storing reactions to the comment';

comment on column content_comments.created_by is 'User who created the comment';

comment on column content_comments.updated_by is 'User who last updated the comment';

comment on column content_comments.is_censored is 'Indicates if the comment has been censored';

comment on column content_comments.censor_reason is 'Reason for censoring the comment, if applicable';

create index if not exists idx_content_comments_content_channel_id
    on content_comments (content_channel_id);

create index if not exists idx_content_comments_created_by
    on content_comments (created_by);

create index if not exists idx_content_comments_reactions_type
    on content_comments using gin ((reactions -> 'type'::text));

create table if not exists schedules
(
    id             uuid                     default gen_random_uuid()            not null
        constraint content_schedules_pkey
            primary key,
    scheduled_at   timestamp with time zone                                      not null,
    status         varchar(30)              default 'PENDING'::character varying not null,
    retry_count    integer                  default 0                            not null,
    last_error     text,
    executed_at    timestamp with time zone,
    created_at     timestamp with time zone default now(),
    updated_at     timestamp with time zone default now(),
    deleted_at     timestamp with time zone,
    created_by     uuid                                                          not null
        constraint content_schedules_created_by_fkey
            references users
            on delete set null,
    reference_id   uuid,
    reference_type reference_type,
    type           varchar(100),
    metadata       jsonb
);

comment on table schedules is 'Stores scheduled content publishing jobs processed via RabbitMQ delayed messages';

comment on column schedules.scheduled_at is 'The time when content should be published';

comment on column schedules.status is 'Current status: PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED';

comment on column schedules.retry_count is 'Number of retry attempts after failures';

comment on column schedules.last_error is 'Error message from the last failed attempt';

comment on column schedules.executed_at is 'Timestamp when the schedule was actually executed';

create index if not exists idx_content_schedules_status
    on schedules (status)
    where (deleted_at IS NULL);

create index if not exists idx_content_schedules_scheduled_at
    on schedules (scheduled_at)
    where (deleted_at IS NULL);

create index if not exists idx_content_schedules_created_by
    on schedules (created_by);

create index if not exists idx_content_schedules_pending
    on schedules (scheduled_at)
    where (((status)::text = 'PENDING'::text) AND (deleted_at IS NULL));

create table if not exists system_alerts
(
    id              uuid                     default gen_random_uuid()           not null
        primary key,
    type            varchar(30)                                                  not null,
    category        varchar(50)                                                  not null,
    severity        varchar(20)              default 'MEDIUM'::character varying not null,
    title           varchar(255)                                                 not null,
    description     text                                                         not null,
    metadata        jsonb                    default '{}'::jsonb,
    target_roles    jsonb                    default '[]'::jsonb                 not null,
    reference_id    uuid,
    reference_type  varchar(50),
    action_url      text,
    status          varchar(20)              default 'ACTIVE'::character varying not null,
    acknowledgement jsonb                    default '{}'::jsonb,
    resolved_by     uuid,
    resolved_at     timestamp with time zone,
    expires_at      timestamp with time zone,
    created_at      timestamp with time zone default now(),
    updated_at      timestamp with time zone default now()
);

comment on table system_alerts is 'Centralized alert system for all staff roles';

comment on column system_alerts.type is 'Alert type: WARNING, ERROR, INFO';

comment on column system_alerts.category is 'Alert category for filtering and grouping';

comment on column system_alerts.severity is 'Alert severity: LOW, MEDIUM, HIGH, CRITICAL';

comment on column system_alerts.target_roles is 'JSON array of user roles that should see this alert';

comment on column system_alerts.reference_id is 'Optional reference to related entity (content, task, campaign, etc.)';

comment on column system_alerts.reference_type is 'Type of the referenced entity';

comment on column system_alerts.action_url is 'URL to navigate when alert is clicked';

comment on column system_alerts.expires_at is 'Optional expiration time for auto-expiring alerts';

create index if not exists idx_system_alerts_status
    on system_alerts (status);

create index if not exists idx_system_alerts_type
    on system_alerts (type);

create index if not exists idx_system_alerts_category
    on system_alerts (category);

create index if not exists idx_system_alerts_severity
    on system_alerts (severity);

create index if not exists idx_system_alerts_expires_at
    on system_alerts (expires_at);

create index if not exists idx_system_alerts_created_at
    on system_alerts (created_at desc);

create index if not exists idx_system_alerts_reference
    on system_alerts (reference_id, reference_type);

create index if not exists idx_system_alerts_target_roles
    on system_alerts using gin (target_roles);

create table if not exists product_options
(
    id          uuid                     default gen_random_uuid() not null
        primary key,
    type        varchar(50)                                        not null
        constraint product_options_type_check
            check ((type)::text = ANY
                   ((ARRAY ['CAPACITY_UNIT'::character varying, 'CONTAINER_TYPE'::character varying, 'DISPENSER_TYPE'::character varying, 'ATTRIBUTE_UNIT'::character varying])::text[])),
    code        varchar(50)                                        not null,
    name        varchar(100)                                       not null,
    description text,
    sort_order  integer                  default 0,
    is_active   boolean                  default true,
    created_at  timestamp with time zone default now(),
    updated_at  timestamp with time zone default now(),
    deleted_at  timestamp with time zone,
    unique (type, code)
);

create index if not exists idx_product_options_type_active
    on product_options (type, is_active)
    where (deleted_at IS NULL);

create index if not exists idx_product_options_code
    on product_options (code);


