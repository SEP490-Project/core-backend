CREATE EXTENSION IF NOT EXISTS pg_trgm;

create table if not exists users
(
    id            uuid                     default gen_random_uuid() not null
        primary key,
    username      varchar(255)                                       not null
        unique,
    email         varchar(255)                                       not null
        unique,
    password_hash text                                               not null,
    full_name     varchar(255)                                       not null,
    phone         varchar(20),
    role          varchar(50)                                        not null
        constraint users_role_check
            check ((role)::text = ANY
                   ((ARRAY ['ADMIN'::character varying, 'MARKETING_STAFF'::character varying, 'CONTENT_STAFF'::character varying, 'SALES_STAFF'::character varying, 'CUSTOMER'::character varying, 'BRAND_PARTNER'::character varying])::text[]))
        constraint chk_roles_users
            check ((role)::text = ANY
                   ((ARRAY ['ADMIN'::character varying, 'MARKETING_STAFF'::character varying, 'CONTENT_STAFF'::character varying, 'SALES_STAFF'::character varying, 'CUSTOMER'::character varying, 'BRAND_PARTNER'::character varying])::text[])),
    created_at    timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at    timestamp with time zone default CURRENT_TIMESTAMP,
    last_login    timestamp with time zone,
    is_active     boolean                  default true,
    deleted_at    timestamp with time zone,
    profile_data  jsonb,
    date_of_birth date,
    avatar_url    text,
    email_enabled boolean                  default true              not null,
    push_enabled  boolean                  default true              not null
);

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
    value_type  value_type               default 'STRING'::value_type not null
        constraint configs_value_type_check
            check ((value_type)::text = ANY
                   (ARRAY [('STRING'::character varying)::text, ('NUMBER'::character varying)::text, ('BOOLEAN'::character varying)::text, ('JSON'::character varying)::text])),
    updated_by  uuid
                                                                      references users
                                                                          on delete set null
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

create table if not exists shipping_addresses
(
    id              uuid                     default gen_random_uuid() not null
        primary key,
    user_id         uuid                                               not null
        references users
            on delete cascade,
    type            varchar(50)                                        not null
        constraint shipping_addresses_type_check
            check ((type)::text = ANY ((ARRAY ['BILLING'::character varying, 'SHIPPING'::character varying])::text[])),
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
            check ((status)::text = ANY ((ARRAY ['ACTIVE'::character varying, 'INACTIVE'::character varying])::text[])),
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
    is_deposit_paid                    boolean                  default false
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
    status                 varchar(50)                                        not null
        constraint contract_payments_status_check
            check ((status)::text = ANY
                   ((ARRAY ['PENDING'::character varying, 'PAID'::character varying, 'OVERDUE'::character varying])::text[])),
    due_date               date                                               not null,
    paid_date              date,
    payment_method         varchar(50)
        constraint contract_payments_payment_method_check
            check ((payment_method)::text = ANY
                   ((ARRAY ['BANK_TRANSFER'::character varying, 'CASH'::character varying, 'CHECK'::character varying])::text[])),
    note                   text,
    created_at             timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at             timestamp with time zone default CURRENT_TIMESTAMP,
    created_by             uuid
                                                                              references users
                                                                                  on delete set null,
    updated_by             uuid
                                                                              references users
                                                                                  on delete set null,
    deleted_at             timestamp with time zone
);

create index if not exists idx_contract_payments_contract_id
    on contract_payments (contract_id);

create table if not exists campaigns
(
    id          uuid                     default gen_random_uuid() not null
        primary key,
    contract_id uuid                                               not null
        references contracts
            on delete cascade,
    name        varchar(255)                                       not null,
    description text,
    start_date  date                                               not null,
    end_date    date                                               not null,
    status      varchar(50)                                        not null
        constraint campaigns_status_check
            check ((status)::text = ANY
                   ((ARRAY ['RUNNING'::character varying, 'COMPLETED'::character varying, 'CANCELED'::character varying])::text[])),
    type        varchar(50)                                        not null,
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

create index if not exists idx_campaigns_contract_id
    on campaigns (contract_id);

create index if not exists idx_campaigns_status
    on campaigns (status);

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
                   ((ARRAY ['NOT_STARTED'::character varying, 'ON_GOING'::character varying, 'CANCELLED'::character varying, 'COMPLETED'::character varying])::text[])),
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
    id           uuid                     default gen_random_uuid() not null
        primary key,
    milestone_id uuid                                               not null
        references milestones
            on delete cascade,
    name         varchar(255)                                       not null,
    description  jsonb,
    deadline     date                                               not null,
    type         varchar(50)                                        not null,
    status       varchar(50)                                        not null
        constraint tasks_status_check
            check ((status)::text = ANY
                   ((ARRAY ['TODO'::character varying, 'IN_PROGRESS'::character varying, 'CANCELLED'::character varying, 'SUBMITTED'::character varying, 'REVISION_REQUESTED'::character varying, 'APPROVED'::character varying, 'ON_RELEASE'::character varying, 'RECAP'::character varying, 'DONE'::character varying])::text[])),
    assigned_to  uuid
                                                                    references users
                                                                        on delete set null,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at   timestamp with time zone,
    created_by   uuid
                                                                    references users
                                                                        on delete set null,
    updated_by   uuid
                                                                    references users
                                                                        on delete set null
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
    affiliate_link     text,
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
    rejection_feedback text
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

create trigger blog_tags_update_usage
    after insert or delete
    on blog_tags
    for each row
execute procedure update_tag_usage_count();

create table if not exists channels
(
    id            uuid                     default gen_random_uuid() not null
        primary key,
    name          varchar(50)                                        not null,
    description   text,
    created_at    timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at    timestamp with time zone default CURRENT_TIMESTAMP,
    is_active     boolean                  default true,
    deleted_at    timestamp with time zone,
    home_page_url text
);

create index if not exists idx_channels_name
    on channels (name);

create table if not exists content_channels
(
    id               uuid                     default gen_random_uuid() not null
        primary key,
    content_id       uuid                                               not null
        references contents
            on delete cascade,
    channel_id       uuid                                               not null
        references channels
            on delete cascade,
    post_date        timestamp with time zone,
    auto_post_status varchar(50)
        constraint content_channels_auto_post_status_check
            check ((auto_post_status)::text = ANY
                   ((ARRAY ['SUCCESS'::character varying, 'FAILED'::character varying, 'PENDING'::character varying])::text[])),
    created_at       timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at       timestamp with time zone default CURRENT_TIMESTAMP
);

create index if not exists idx_content_channels_content_id
    on content_channels (content_id);

create index if not exists idx_content_channels_channel_id
    on content_channels (channel_id);

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

create table if not exists products
(
    id          uuid                     default gen_random_uuid()          not null
        primary key,
    brand_id    uuid                                                        not null
        references brands
            on delete cascade,
    category_id uuid
                                                                            references product_categories
                                                                                on delete set null,
    name        varchar(255)                                                not null,
    description text                                                        not null,
    type        varchar(50)                                                 not null
        constraint products_type_check
            check ((type)::text = ANY ((ARRAY ['STANDARD'::character varying, 'LIMITED'::character varying])::text[])),
    created_at  timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at  timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at  timestamp with time zone,
    task_id     uuid
                                                                            references tasks
                                                                                on delete set null,
    status      varchar(50)              default 'DRAFT'::character varying not null
        constraint products_status_check
            check ((status)::text = ANY
                   ((ARRAY ['DRAFT'::character varying, 'SUBMITTED'::character varying, 'REVISION'::character varying, 'APPROVED'::character varying, 'ACTIVED'::character varying, 'INACTIVED'::character varying])::text[])),
    is_active   boolean                  default false                      not null,
    created_by  uuid
                                                                            references users
                                                                                on delete set null,
    updated_by  uuid
                                                                            references users
                                                                                on delete set null
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
    capacity_unit     varchar(20)
        constraint product_variants_capacity_unit_check
            check ((capacity_unit)::text = ANY
                   ((ARRAY ['ML'::character varying, 'L'::character varying, 'G'::character varying, 'KG'::character varying, 'OZ'::character varying])::text[])),
    container_type    varchar(50)
        constraint product_variants_container_type_check
            check ((container_type)::text = ANY
                   ((ARRAY ['BOTTLE'::character varying, 'TUBE'::character varying, 'JAR'::character varying, 'STICK'::character varying, 'PENCIL'::character varying, 'COMPACT'::character varying, 'PALLETE'::character varying, 'SACHET'::character varying, 'VIAL'::character varying, 'ROLLER_BOTTLE'::character varying])::text[])),
    dispenser_type    varchar(50)
        constraint product_variants_dispenser_type_check
            check ((dispenser_type)::text = ANY
                   ((ARRAY ['PUMP'::character varying, 'SPRAY'::character varying, 'DROPPER'::character varying, 'ROLL_ON'::character varying, 'TWIST_UP'::character varying, 'SQUEEZE'::character varying, 'NONE'::character varying])::text[])),
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
                                                                             on delete set null
);

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
    unit         varchar(50)
        constraint variant_attribute_values_unit_check
            check ((unit)::text = ANY
                   ((ARRAY ['%'::character varying, 'MG'::character varying, 'G'::character varying, 'ML'::character varying, 'L'::character varying, 'IU'::character varying, 'PPM'::character varying, 'NONE'::character varying])::text[])),
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
    id           uuid                     default gen_random_uuid() not null
        primary key,
    user_id      uuid                                               not null
        references users
            on delete set null,
    status       varchar(50)                                        not null
        constraint orders_status_check
            check ((status)::text = ANY
                   ((ARRAY ['PENDING'::character varying, 'PAID'::character varying, 'REFUNDED'::character varying, 'CONFIRMED'::character varying, 'CANCELED'::character varying, 'SHIPPED'::character varying, 'IN_TRANSIT'::character varying, 'DELIVERED'::character varying, 'RECEIVED'::character varying])::text[])),
    total_amount numeric(15, 2)                                     not null,
    address_id   uuid                                               not null
        references shipping_addresses
            on delete restrict,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP
);

create index if not exists idx_orders_user_id
    on orders (user_id);

create index if not exists idx_orders_address_id
    on orders (address_id);

create index if not exists idx_orders_status
    on orders (status);

create table if not exists order_items
(
    id                     uuid                     default gen_random_uuid() not null
        primary key,
    order_id               uuid                                               not null
        references orders
            on delete cascade,
    variant_id             uuid                                               not null
        references product_variants
            on delete cascade,
    quantity               integer                                            not null,
    subtotal               numeric(15, 2)                                     not null,
    unit_price             numeric(15, 2)                                     not null,
    capacity               numeric(10, 2),
    capacity_unit          varchar(20)
        constraint order_items_capacity_unit_check
            check ((capacity_unit)::text = ANY
                   ((ARRAY ['ML'::character varying, 'L'::character varying, 'G'::character varying, 'KG'::character varying, 'OZ'::character varying])::text[])),
    container_type         varchar(50)
        constraint order_items_container_type_check
            check ((container_type)::text = ANY
                   ((ARRAY ['BOTTLE'::character varying, 'TUBE'::character varying, 'JAR'::character varying, 'STICK'::character varying, 'PENCIL'::character varying, 'COMPACT'::character varying, 'PALLETE'::character varying, 'SACHET'::character varying, 'VIAL'::character varying, 'ROLLER_BOTTLE'::character varying])::text[])),
    dispenser_type         varchar(50)
        constraint order_items_dispenser_type_check
            check ((dispenser_type)::text = ANY
                   ((ARRAY ['PUMP'::character varying, 'SPRAY'::character varying, 'DROPPER'::character varying, 'ROLL_ON'::character varying, 'TWIST_UP'::character varying, 'SQUEEZE'::character varying, 'NONE'::character varying])::text[])),
    uses                   varchar(255),
    manufacturing_date     date,
    expiry_date            date,
    instructions           text,
    attributes_description jsonb,
    item_status            varchar(50)                                        not null
        constraint order_items_item_status_check
            check ((item_status)::text = ANY
                   ((ARRAY ['PENDING'::character varying, 'PAID'::character varying, 'REFUNDED'::character varying, 'CONFIRMED'::character varying, 'CANCELED'::character varying, 'SHIPPED'::character varying, 'IN_TRANSIT'::character varying, 'DELIVERED'::character varying, 'RECEIVED'::character varying])::text[])),
    updated_at             timestamp with time zone default CURRENT_TIMESTAMP
);

create index if not exists idx_order_items_order_id
    on order_items (order_id);

create index if not exists idx_order_items_variant_id
    on order_items (variant_id);

create table if not exists pre_orders
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    user_id      uuid                                               not null
        references users
            on delete set null,
    variant_id   uuid                                               not null
        references product_variants
            on delete cascade,
    quantity     integer                                            not null,
    unit_price   numeric(15, 2)                                     not null,
    total_amount numeric(15, 2)                                     not null,
    status       varchar(50)                                        not null
        constraint pre_orders_status_check
            check ((status)::text = ANY
                   ((ARRAY ['PENDING'::character varying, 'PRE_ORDERED'::character varying, 'AWAITING_RELEASE'::character varying, 'AWAITING_PICKUP'::character varying, 'CONFIRMED'::character varying, 'CANCELLED'::character varying, 'IN_TRANSIT'::character varying, 'DELIVERED'::character varying, 'RECEIVED'::character varying])::text[])),
    created_at   timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP
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
                   ((ARRAY ['PENDING'::character varying, 'APPROVED'::character varying, 'REJECTED'::character varying, 'COMPLETED'::character varying])::text[])),
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

create table if not exists reviews
(
    id           uuid                     default gen_random_uuid() not null
        primary key,
    variant_id   uuid                                               not null
        references product_variants
            on delete cascade,
    user_id      uuid
                                                                    references users
                                                                        on delete set null,
    order_id     uuid
                                                                    references orders
                                                                        on delete set null,
    pre_order_id uuid
                                                                    references pre_orders
                                                                        on delete set null,
    rating       integer                                            not null
        constraint reviews_rating_check
            check ((rating >= 1) AND (rating <= 5)),
    comment      text,
    image_url    text,
    review_date  timestamp with time zone default CURRENT_TIMESTAMP,
    created_at   timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at   timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at   timestamp with time zone,
    unique (variant_id, user_id, order_id, pre_order_id),
    constraint review_one_of_order_or_preorder
        check (((order_id IS NOT NULL) AND (pre_order_id IS NULL)) OR
               ((order_id IS NULL) AND (pre_order_id IS NOT NULL)))
);

create index if not exists idx_reviews_variant_id
    on reviews (variant_id);

create index if not exists idx_reviews_user_id
    on reviews (user_id);

create index if not exists idx_reviews_order_id
    on reviews (order_id);

create index if not exists idx_reviews_pre_order_id
    on reviews (pre_order_id);

create table if not exists files
(
    id          uuid                     default gen_random_uuid() not null
        primary key,
    file_name   varchar(255)                                       not null,
    alt_text    varchar(255),
    url         text                                               not null,
    mime_type   varchar(100)                                       not null,
    size        bigint                                             not null,
    uploaded_at timestamp with time zone default CURRENT_TIMESTAMP,
    uploaded_by uuid
                                                                   references users
                                                                       on delete set null
);

create index if not exists idx_files_uploaded_by
    on files (uploaded_by);

create table if not exists payment_transactions
(
    id               uuid                     default gen_random_uuid() not null
        primary key,
    reference_id     uuid                                               not null,
    reference_type   varchar(50)                                        not null,
    amount           numeric(15, 2)                                     not null,
    method           varchar(50)                                        not null
        constraint payment_transactions_method_check
            check ((method)::text = ANY ((ARRAY ['COD'::character varying, 'ONLINE'::character varying])::text[])),
    status           varchar(50)                                        not null
        constraint payment_transactions_status_check
            check ((status)::text = ANY
                   ((ARRAY ['PAID'::character varying, 'PENDING'::character varying, 'REFUNDED'::character varying])::text[])),
    transaction_date timestamp with time zone default CURRENT_TIMESTAMP,
    gateway_ref      varchar(255),
    updated_at       timestamp with time zone default CURRENT_TIMESTAMP,
    gateway_id       text
);

create index if not exists idx_payment_transactions_ref
    on payment_transactions (reference_id, reference_type);

create table if not exists modified_histories
(
    id             uuid                     default gen_random_uuid()                not null
        primary key,
    reference_id   uuid,
    reference_type varchar(50)                                                       not null
        constraint modified_histories_reference_type_chk
            check ((reference_type)::text = ANY
                   ((ARRAY ['CONTRACT'::character varying, 'CAMPAIGN'::character varying, 'MILESTONE'::character varying, 'TASK'::character varying, 'CONTENT'::character varying, 'PRODUCT'::character varying, 'BLOG'::character varying])::text[])),
    operation      varchar(50)                                                       not null
        constraint modified_histories_operation_chk
            check ((operation)::text = ANY
                   ((ARRAY ['CREATE'::character varying, 'UPDATE'::character varying, 'DELETE'::character varying])::text[])),
    description    text                                                              not null,
    changed_by     uuid
                                                                                     references users
                                                                                         on delete set null,
    changed_at     timestamp with time zone default CURRENT_TIMESTAMP,
    status         varchar(20)              default 'IN_PROGRESS'::character varying not null
        constraint modified_histories_status_check
            check ((status)::text = ANY
                   ((ARRAY ['IN_PROGRESS'::character varying, 'COMPLETED'::character varying, 'FAILED'::character varying])::text[]))
);

create index if not exists idx_modified_histories_changed_by
    on modified_histories (changed_by);

create index if not exists idx_modified_histories_poly_ref
    on modified_histories (reference_id, reference_type);

create table if not exists kpi_metrics
(
    id             uuid default gen_random_uuid() not null,
    reference_id   uuid                           not null,
    reference_type reference_type                 not null,
    type           varchar(50)                    not null
        constraint kpi_metrics_type_check
            check ((type)::text = ANY
                   ((ARRAY ['REACH'::character varying, 'IMPRESSIONS'::character varying, 'LIKES'::character varying, 'COMMENTS'::character varying, 'SHARES'::character varying, 'CTR'::character varying, 'ENGAGEMENT'::character varying])::text[])),
    value          numeric(15, 2)                 not null,
    recorded_date  timestamp with time zone       not null,
    unit           varchar(10),
    primary key (id, recorded_date)
);

create index if not exists idx_kpi_metrics_ref
    on kpi_metrics (reference_id, reference_type);

create index if not exists kpi_metrics_recorded_date_idx
    on kpi_metrics (recorded_date desc);

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

create table if not exists limited_products
(
    id                      uuid    not null
        primary key
        references products
            on delete cascade,
    max_stock               integer not null,
    is_free_shipping        boolean default false,
    bought_limit            integer default 1,
    premiere_date           date,
    availability_start_date date,
    availability_end_date   date,
    concept_id              uuid
        constraint limited_products_concept_id_unique
            unique
                                    references concepts
                                        on delete set null
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
    deleted_at        timestamp with time zone
);

comment on table notifications is 'Stores all notification attempts (email and push) with flexible JSONB metadata';

comment on column notifications.type is 'Notification type: EMAIL or PUSH';

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

create table if not exists device_tokens
(
    id            uuid                     default gen_random_uuid() not null
        primary key,
    user_id       uuid                                               not null
        constraint fk_device_tokens_user
            references users
            on update cascade on delete cascade,
    token         varchar(255)                                       not null,
    platform      varchar(50)                                        not null,
    registered_at timestamp with time zone default CURRENT_TIMESTAMP not null,
    last_used_at  timestamp with time zone,
    is_valid      boolean                  default true              not null,
    created_at    timestamp with time zone default CURRENT_TIMESTAMP,
    updated_at    timestamp with time zone default CURRENT_TIMESTAMP,
    deleted_at    timestamp with time zone
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

create table if not exists affiliate_links
(
    id           uuid                     default gen_random_uuid()           not null
        primary key,
    hash         varchar(16)                                                  not null
        unique,
    contract_id  uuid                                                         not null
        references contracts
            on delete cascade,
    content_id   uuid                                                         not null
        references contents
            on delete cascade,
    channel_id   uuid                                                         not null
        references channels
            on delete restrict,
    tracking_url text                                                         not null,
    status       varchar(20)              default 'active'::character varying not null
        constraint affiliate_links_status_check
            check ((status)::text = ANY
                   ((ARRAY ['active'::character varying, 'inactive'::character varying, 'expired'::character varying])::text[])),
    created_at   timestamp with time zone default now(),
    updated_at   timestamp with time zone default now(),
    deleted_at   timestamp with time zone,
    constraint unique_affiliate_combination
        unique (contract_id, content_id, channel_id)
);

comment on table affiliate_links is 'Stores unique trackable affiliate links for content+channel combinations';

comment on column affiliate_links.hash is 'Base62 SHA-256 hash (16 chars) for public URL shortening';

comment on column affiliate_links.tracking_url is 'Original affiliate product URL from contract ScopeOfWork';

comment on column affiliate_links.status is 'active: clickable, inactive: paused, expired: contract/content ended';

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

