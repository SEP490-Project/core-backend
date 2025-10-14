CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Core Entities
CREATE TABLE configs
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key         VARCHAR(255) UNIQUE NOT NULL,
    value       TEXT                NOT NULL,
    type        VARCHAR(30)         NOT NULL CHECK (
        type IN ('EMAIL', 'PASSWORD', 'NUMBER', 'DATE')
        ),
    description TEXT,
    created_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE users
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(255) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT                NOT NULL,
    full_name     VARCHAR(255)        NOT NULL,
    phone         VARCHAR(20),
    role          VARCHAR(50)         NOT NULL CHECK (
        role IN (
                 'ADMIN',
                 'MARKETING_STAFF',
                 'CONTENT_STAFF',
                 'SALES_STAFF',
                 'CUSTOMER',
                 'BRAND_PARTNER'
            )
        ),
    date_of_birth DATE,
    created_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    last_login    TIMESTAMPTZ,
    is_active     BOOLEAN          DEFAULT TRUE,
    deleted_at    TIMESTAMPTZ,
-- Additional profile information used for AI assistance personalization
    profile_data  JSONB
);

CREATE TABLE logged_sessions
(
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    refresh_token_hash TEXT         NOT NULL,
    device_fingerprint VARCHAR(255) NOT NULL,
    expiry_at          TIMESTAMPTZ  NOT NULL,
    is_revoked         BOOLEAN          DEFAULT FALSE,
    last_used_at       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at         TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at         TIMESTAMPTZ
);

CREATE TABLE notifications
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    type         VARCHAR(50) NOT NULL CHECK (type IN ('REMINDER', 'ALERT', 'EMAIL')),
    message      TEXT        NOT NULL,
    message_data JSONB,
    channel      VARCHAR(50) NOT NULL CHECK (channel IN ('EMAIL', 'IN_APP', 'PUSH')),
    send_time    TIMESTAMPTZ NOT NULL,
    status       VARCHAR(50) NOT NULL CHECK (status IN ('PENDING', 'SENT', 'READ')),
    related_id   UUID,
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE shipping_addresses
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
-- For now only 'SHIPPING' is used since There is no credit card payment yet.
    type          VARCHAR(50)  NOT NULL CHECK (type IN ('BILLING', 'SHIPPING')),
    full_name     VARCHAR(255) NOT NULL,          -- The full name of the recipient at the shipping address, which may be different from the user account holder's name (e.g., when sending a gift).
    phone_number  VARCHAR(20),                    -- The recipient's phone number, often required by delivery services for contact during shipment.
    email         VARCHAR(255),                   -- The recipient's email address, used for sending shipping notifications and tracking updates. Can differ from the user's primary account email.
    street        VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),                   -- An optional field for additional address details, such as an apartment number, suite, building name, or P.O. Box.
    city          VARCHAR(255) NOT NULL,
    state         VARCHAR(255),                   -- The state, province, or region. This field is nullable as not all countries use this subdivision in their addresses.
    postal_code   VARCHAR(20)  NOT NULL,
    country       VARCHAR(255) NOT NULL,
    company       VARCHAR(255),                   -- An optional field for the company name if the package is being delivered to a business address.
    is_default    BOOLEAN          DEFAULT FALSE, -- A flag to mark one address as the user's primary or default shipping address, which can be pre-selected during checkout to speed up the process.
    created_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at    TIMESTAMPTZ
);
-- Campaigns Entities
CREATE TABLE brands
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
-- This is nullable since brand info can be created without account
    user_id       UUID REFERENCES users (id) ON DELETE RESTRICT,
    name          VARCHAR(255) NOT NULL,
    description   TEXT,
    contact_email VARCHAR(255),
    contact_phone VARCHAR(20),
    address       VARCHAR(255),
    website       VARCHAR(255),
    logo_url      TEXT,
    status        VARCHAR(50)  NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE contracts
(
    id                                 UUID PRIMARY KEY         DEFAULT gen_random_uuid(),
    parent_contract_id                 UUID         REFERENCES contracts (id) ON DELETE SET NULL,
    title                              VARCHAR(255),
    contract_number                    VARCHAR(100) NOT NULL UNIQUE,           -- (Số Hợp đồng)

-- Categorizing the contract is crucial as it defines which JSONB fields are required.
    type                               VARCHAR(50)  NOT NULL CHECK (
        type IN ('ADVERTISING', 'AFFILIATE', 'BRAND_AMBASSADOR', 'CO_PRODUCING')
        ),
    status                             VARCHAR(50)  NOT NULL    DEFAULT 'DRAFT' CHECK (
        status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'TERMINATED')
        ),

-- Dates and Locations (Common to all templates)
    signed_date                        DATE,                                   -- (Ngày ký)
    signed_location                    TEXT,                                   -- (tại [Địa điểm])
    start_date                         DATE,                                   -- (Ngày bắt đầu hiệu lực)
    end_date                           DATE,                                   -- (Ngày kết thúc / Thời hạn hợp đồng)

-- Brand reference and brand information stored in contract for record-keeping
    brand_id                           UUID         NOT NULL REFERENCES brands (id),
    brand_tax_number                   VARCHAR(100),
    brand_representative_name          VARCHAR(255),
    brand_representative_role          VARCHAR(255),
    brand_representative_phone         VARCHAR(20),
    brand_representative_email         VARCHAR(255),
    brand_bank_name                    VARCHAR(255),
    brand_account_number               VARCHAR(255),

-- KOL/Influencer Representative information (The other party in the contract)
    representative_name                VARCHAR(255) NOT NULL,                  -- (Họ và tên)
    representative_role                VARCHAR(255),                           -- (Chức vụ)
    representative_phone               VARCHAR(20),                            -- (Số điện thoại)
    representative_email               VARCHAR(255),                           -- (Email)
    representative_tax_number          VARCHAR(100),
    representative_bank_name           VARCHAR(255),
    representative_bank_account_number VARCHAR(255),
    representative_bank_account_holder VARCHAR(255),

-- Financials
    currency                           VARCHAR(3)               DEFAULT 'VND', -- (Đồng tiền thanh toán - Template 1)

-- This field stores the payment structure, which varies wildly between the 4 templates.
    financial_terms                    JSONB        NOT NULL    DEFAULT '{}',

-- Scope of Work and Deliverables
-- Combines "Nội dung hợp đồng", "Yêu cầu kỹ thuật", and "Trách nhiệm" regarding the
-- work itself.
    scope_of_work                      JSONB        NOT NULL    DEFAULT '{}',

-- Legal Clauses and Penalties
-- Combines Penalties, Force Majeure, Warranty, Dispute Resolution, etc.
    legal_terms                        JSONB        NOT NULL    DEFAULT '{}',

-- File attachments
    contract_file_url                  TEXT,
    proposal_file_url                  TEXT,

-- Auditing
    created_at                         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at                         TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at                         TIMESTAMP WITH TIME ZONE DEFAULT NULL
);

CREATE TABLE contract_payments
(
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id            UUID           NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    installment_percentage DECIMAL(5, 2) CHECK (
        installment_percentage BETWEEN 0 AND 100
        ),
    amount                 DECIMAL(15, 2) NOT NULL,
    status                 VARCHAR(50)    NOT NULL CHECK (
        status IN ('PENDING', 'PAID', 'OVERDUE')
        ),
    due_date               DATE           NOT NULL,
    paid_date              DATE,
    payment_method         VARCHAR(50) CHECK (
        payment_method IN ('BANK_TRANSFER', 'CASH', 'CHECK')
        ),
    note                   TEXT,
    created_at             TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at             TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE campaigns
(
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id      UUID         NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    description      TEXT,
    start_date       DATE         NOT NULL,
    end_date         DATE         NOT NULL,
    status           VARCHAR(50)  NOT NULL CHECK (
        status IN ('RUNNING', 'COMPLETED', 'CANCELED')
        ),
    budget_projected DECIMAL(15, 2),
    budget_actual    DECIMAL(15, 2),
    type             VARCHAR(50)  NOT NULL, -- Use the same type as the contract
    created_at       TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at       TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at       TIMESTAMPTZ
);

CREATE TABLE milestones
(
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id           UUID        NOT NULL REFERENCES campaigns (id) ON DELETE CASCADE,
    description           TEXT        NOT NULL,
    due_date              DATE        NOT NULL,
    completed_at          TIMESTAMPTZ,
    completion_percentage INTEGER          DEFAULT 0 CHECK (
        completion_percentage BETWEEN 0 AND 100
        ),
    status                VARCHAR(50) NOT NULL CHECK (
        status IN ('NOT_STARTED', 'ON_GOING', 'CANCELLED', 'COMPLETED')
        ),
    behind_schedule       BOOLEAN          DEFAULT FALSE,
    created_at            TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at            TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at            TIMESTAMPTZ
);

CREATE TABLE tasks
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    milestone_id UUID         NOT NULL REFERENCES milestones (id) ON DELETE CASCADE,
    name         VARCHAR(255) NOT NULL,
    description  JSONB,
    deadline     DATE         NOT NULL,
    type         VARCHAR(50)  NOT NULL,
    status       VARCHAR(50)  NOT NULL CHECK (
        status IN (
                   'TODO',
                   'IN_PROGRESS',
                   'CANCELLED',
                   'SUBMITTED',
                   'REVISION_REQUESTED',
                   'APPROVED',
                   'ON_RELEASE',
                   'RECAP',
                   'DONE'
            )
        ),
    assigned_to  UUID         REFERENCES users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at   TIMESTAMPTZ
);

CREATE TABLE contents
(
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id           UUID         REFERENCES tasks (id) ON DELETE SET NULL,
    title             VARCHAR(255) NOT NULL,
    type              VARCHAR(50)  NOT NULL CHECK (
        type IN ('POST', 'VIDEO')
        ),
    body              JSONB        NOT NULL,
    publish_date      TIMESTAMPTZ,
    affiliate_link    VARCHAR(255),
    status            VARCHAR(50)  NOT NULL CHECK (
        status IN (
                   'DRAFT',
                   'AWAIT_STAFF',
                   'AWAIT_BRAND',
                   'REJECTED',
                   'APPROVED',
                   'POSTED'
            )
        ),
    ai_generated_text TEXT,
    created_at        TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at        TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at        TIMESTAMPTZ
);

CREATE TABLE blogs
(
    content_id UUID PRIMARY KEY REFERENCES contents (id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users (id) ON DELETE SET NULL,
    tags       JSONB,
    excerpt    TEXT,
    read_time  INTEGER
);

CREATE TABLE tags
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    usage_count INTEGER          DEFAULT 0,
    created_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE blog_tags
(
    blog_id UUID NOT NULL REFERENCES blogs (content_id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (blog_id, tag_id)
);

CREATE TABLE channels
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(50) NOT NULL CHECK (
        name IN ('WEBSITE', 'FACEBOOK', 'TIKTOK')
        ),
    description TEXT,
    created_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    is_active   BOOLEAN          DEFAULT TRUE,
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE content_channels
(
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id       UUID NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    channel_id       UUID NOT NULL REFERENCES channels (id) ON DELETE CASCADE,
    post_date        TIMESTAMPTZ,
    auto_post_status VARCHAR(50) CHECK (
        auto_post_status IN ('SUCCESS', 'FAILED', 'PENDING')
        ),
    created_at       TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at       TIMESTAMPTZ      DEFAULT current_timestamp
);

-- E-Commerce Entities
CREATE TABLE product_categories
(
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(255) NOT NULL,
    description        TEXT,
    parent_category_id UUID         REFERENCES product_categories (
                                                                   id
        ) ON DELETE SET NULL,
    created_at         TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at         TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at         TIMESTAMPTZ
);

CREATE TABLE products
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id      UUID           NOT NULL REFERENCES brands (id) ON DELETE CASCADE,
    category_id   UUID           REFERENCES product_categories (id) ON DELETE SET NULL,
    task_id       UUID           REFERENCES tasks (id) ON DELETE SET NULL,
    name          VARCHAR(255)   NOT NULL,
    description   TEXT           NOT NULL,
    price         DECIMAL(15, 2) NOT NULL,
    current_stock INTEGER        NOT NULL,
    type          VARCHAR(50)    NOT NULL CHECK (
        type IN ('STANDARD', 'LIMITED')
        ),
    is_active     BOOLEAN          DEFAULT TRUE,
    created_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at    TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at    TIMESTAMPTZ
);

CREATE TABLE limited_products
(
    id                      UUID PRIMARY KEY REFERENCES products (id) ON DELETE CASCADE,
    max_stock               INTEGER NOT NULL,
    is_free_shipping        BOOLEAN DEFAULT FALSE,
    bought_limit            INTEGER DEFAULT 1,
    premiere_date           DATE,
    availability_start_date DATE,
    availability_end_date   DATE
);

CREATE TABLE product_variants
(
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id        UUID    NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    price             DECIMAL(15, 2),
    current_stock     INTEGER NOT NULL,
    capacity          DECIMAL(10, 2),
    capacity_unit     VARCHAR(20) CHECK (
        capacity_unit IN ('ML', 'L', 'G', 'KG', 'OZ')
        ),
    container_type    VARCHAR(50) CHECK (
        container_type IN (
                           'BOTTLE',
                           'TUBE',
                           'JAR',
                           'STICK',
                           'PENCIL',
                           'COMPACT',
                           'PALLETE',
                           'SACHET',
                           'VIAL',
                           'ROLLER_BOTTLE'
            )
        ),
    dispenser_type    VARCHAR(50) CHECK (
        dispenser_type IN (
                           'PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON', 'TWIST_UP', 'SQUEEZE', 'NONE'
            )
        ),
    uses              VARCHAR(255),
    manufactring_date DATE,
    expiry_date       DATE,
    instructions      TEXT,
    is_default        BOOLEAN          DEFAULT FALSE,
    is_active         BOOLEAN          DEFAULT TRUE,
    created_at        TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at        TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at        TIMESTAMPTZ
);

-- Product story to hold rich content about the product, this is reserved for
-- limited_product only
CREATE TABLE product_stories
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID  NOT NULL REFERENCES product_variants (
                                                           id
        ) ON DELETE CASCADE,
    content    JSONB NOT NULL,
    created_at TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE variant_images
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id UUID NOT NULL REFERENCES product_variants (
                                                          id
        ) ON DELETE CASCADE,
    image_url  TEXT NOT NULL,
    alt_text   VARCHAR(255),
    is_primary BOOLEAN          DEFAULT FALSE,
    created_at TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE variant_attributes
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ingredient  VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at  TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE variant_attribute_values
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id   UUID           NOT NULL REFERENCES product_variants (
                                                                      id
        ) ON DELETE CASCADE,
    attribute_id UUID           NOT NULL REFERENCES variant_attributes (
                                                                        id
        ) ON DELETE CASCADE,
    value        DECIMAL(10, 2) NOT NULL,
    unit         VARCHAR(50) CHECK (
        unit IN
        ('%', 'MG', 'G', 'ML', 'L', 'IU', 'PPM', 'NONE')
        ),
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at   TIMESTAMPTZ,
    UNIQUE (variant_id, attribute_id)
);

CREATE TABLE content_products
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_id    UUID NOT NULL REFERENCES contents (id) ON DELETE CASCADE,
    product_id    UUID NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    affiliate_url VARCHAR(255)
);

-- Cart, Order, and Review Entities
CREATE TABLE carts
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE cart_items
(
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cart_id    UUID           NOT NULL REFERENCES carts (id) ON DELETE CASCADE,
    variant_id UUID           NOT NULL REFERENCES product_variants (
                                                                    id
        ) ON DELETE CASCADE,
    quantity   INTEGER        NOT NULL,
    subtotal   DECIMAL(15, 2) NOT NULL,
    updated_at TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE orders
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID           NOT NULL REFERENCES users (id) ON DELETE SET NULL,
    status       VARCHAR(50)    NOT NULL CHECK (
        status IN (
                   'PENDING',
                   'PAID',
                   'REFUNDED',
                   'CONFIRMED',
                   'CANCELED',
                   'SHIPPED',
                   'IN_TRANSIT',
                   'DELIVERED',
                   'RECEIVED'
            )
        ),
    total_amount DECIMAL(15, 2) NOT NULL,
    address_id   UUID           NOT NULL REFERENCES shipping_addresses (
                                                                        id
        ) ON DELETE RESTRICT,
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE order_items
(
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id               UUID           NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    variant_id             UUID           NOT NULL REFERENCES product_variants (
                                                                                id
        ) ON DELETE CASCADE,
    quantity               INTEGER        NOT NULL,
    subtotal               DECIMAL(15, 2) NOT NULL,

-- Snapshot fields to preserve product data at the time of purchase
-- The price of a single unit when the order was placed.
    unit_price             DECIMAL(15, 2) NOT NULL,
    capacity               DECIMAL(10, 2),
    capacity_unit          VARCHAR(20) CHECK (
        capacity_unit IN ('ML', 'L', 'G', 'KG', 'OZ')
        ),
    container_type         VARCHAR(50) CHECK (
        container_type IN (
                           'BOTTLE',
                           'TUBE',
                           'JAR',
                           'STICK',
                           'PENCIL',
                           'COMPACT',
                           'PALLETE',
                           'SACHET',
                           'VIAL',
                           'ROLLER_BOTTLE'
            )
        ),
    dispenser_type         VARCHAR(50) CHECK (
        dispenser_type IN (
                           'PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON', 'TWIST_UP', 'SQUEEZE', 'NONE'
            )
        ),
    uses                   VARCHAR(255),
    manufactring_date      DATE,
    expiry_date            DATE,
    instructions           TEXT,
    attributes_description JSONB, -- A human-readable description of the variant's attributes (e.g., "Color: Blue, Size: Large").
    item_status            VARCHAR(50)    NOT NULL CHECK (
        item_status IN (
                        'PENDING',
                        'PAID',
                        'REFUNED',
                        'CONFIRMED',
                        'CANCELED',
                        'SHIPPED',
                        'IN_TRANSIT',
                        'DELIVERED',
                        'RECEIVED'
            )
        ),
    updated_at             TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE pre_orders
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID           NOT NULL REFERENCES users (id) ON DELETE SET NULL,
    variant_id   UUID           NOT NULL REFERENCES product_variants (
                                                                      id
        ) ON DELETE CASCADE,
    quantity     INTEGER        NOT NULL,
    unit_price   DECIMAL(15, 2) NOT NULL,
    total_amount DECIMAL(15, 2) NOT NULL,
    status       VARCHAR(50)    NOT NULL CHECK (
        status IN (
                   'PENDING',
                   'PRE_ORDERED',
                   'AWAITING_RELEASE',
                   'AWAITING_PICKUP',
                   'CONFIRMED',
                   'CANCELLED',
                   'IN_TRANSIT',
                   'DELIVERED',
                   'RECEIVED'
            )
        ),
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp
);

CREATE TABLE refund_requests
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id     UUID REFERENCES orders (id) ON DELETE CASCADE,
    pre_order_id UUID REFERENCES pre_orders (id) ON DELETE CASCADE,
    reason       TEXT           NOT NULL,
    amount       DECIMAL(15, 2) NOT NULL,
    status       VARCHAR(50)    NOT NULL CHECK (
        status IN ('PENDING', 'APPROVED', 'REJECTED', 'COMPLETED')
        ),
    requested_at TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    constraint REFUND_ONE_OF_ORDER_OR_PREORDER CHECK (
        (order_id IS NOT NULL AND pre_order_id IS NULL)
            OR (order_id IS NULL AND pre_order_id IS NOT NULL)
        ),
    UNIQUE (order_id, pre_order_id)
);

CREATE TABLE reviews
(
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id   UUID    NOT NULL REFERENCES product_variants (
                                                               id
        ) ON DELETE CASCADE,
    user_id      UUID    REFERENCES users (id) ON DELETE SET NULL,
    order_id     UUID    REFERENCES orders (id) ON DELETE SET NULL,
    pre_order_id UUID    REFERENCES pre_orders (id) ON DELETE SET NULL,
    rating       INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment      TEXT,
    image_url    TEXT,
    review_date  TIMESTAMPTZ      DEFAULT current_timestamp,
    created_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    updated_at   TIMESTAMPTZ      DEFAULT current_timestamp,
    deleted_at   TIMESTAMPTZ,
    constraint REVIEW_ONE_OF_ORDER_OR_PREORDER CHECK (
        (order_id IS NOT NULL AND pre_order_id IS NULL)
            OR (order_id IS NULL AND pre_order_id IS NOT NULL)
        ),
    UNIQUE (variant_id, user_id, order_id, pre_order_id)
);

-- A table to log metadata about uploaded files. Other tables link via storing the URL.
CREATE TABLE files
(
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_name   VARCHAR(255) NOT NULL,
    alt_text    VARCHAR(255),
    url         TEXT         NOT NULL,
    mime_type   VARCHAR(100) NOT NULL,
    size        BIGINT       NOT NULL,
    uploaded_at TIMESTAMPTZ      DEFAULT current_timestamp,
    uploaded_by UUID         REFERENCES users (id) ON DELETE SET NULL
);

-- A centralized table for financial transactions.
-- reference_type can be 'ORDER', 'CONTRACT_PAYMENT'.
CREATE TABLE payment_transactions
(
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id     UUID           NOT NULL,
    reference_type   VARCHAR(50)    NOT NULL,
    amount           DECIMAL(15, 2) NOT NULL,
    method           VARCHAR(50)    NOT NULL CHECK (method IN ('COD', 'ONLINE')),
    status           VARCHAR(50)    NOT NULL CHECK (
        status IN ('PAID', 'PENDING', 'REFUNDED')
        ),
    transaction_date TIMESTAMPTZ      DEFAULT current_timestamp,
    gateway_ref      VARCHAR(255),
    updated_at       TIMESTAMPTZ      DEFAULT current_timestamp
);

-- A centralized table for tracking changes to other tables.
-- reference_type could be 'CONTRACT', 'PRODUCT', 'USER_ACCOUNT', etc.
CREATE TABLE modified_histories
(
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id   UUID        NOT NULL,
    reference_type VARCHAR(50) NOT NULL,
    change_type    VARCHAR(50) NOT NULL,
    description    TEXT        NOT NULL,
    changed_by     UUID        REFERENCES users (id) ON DELETE SET NULL,
    changed_at     TIMESTAMPTZ      DEFAULT current_timestamp
);

-- A centralized table for performance metrics.
-- reference_type can be 'CONTENT', 'CAMPAIGN'.
CREATE TABLE kpi_metrics
(
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reference_id   UUID           NOT NULL,
    reference_type VARCHAR(50)    NOT NULL CHECK (
        reference_type IN ('CONTENT', 'CAMPAIGN')
        ),
    type           VARCHAR(50)    NOT NULL CHECK (
        type IN (
                 'REACH',
                 'IMPRESSIONS',
                 'LIKES',
                 'COMMENTS',
                 'SHARES',
                 'CTR',
                 'ENGAGEMENT'
            )
        ),
    value          DECIMAL(15, 2) NOT NULL,
    recorded_date  TIMESTAMPTZ    NOT NULL,
    unit           VARCHAR(10)
);

-- =================================================================
-- I. CORE INDEXES FOR FOREIGN KEYS (Essential for JOIN performance)
-- =================================================================
-- Core Entities
CREATE INDEX idx_logged_sessions_user_id ON logged_sessions (user_id);
CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_modified_histories_changed_by ON modified_histories (
                                                                      changed_by
    );
CREATE INDEX idx_shipping_addresses_user_id ON shipping_addresses (user_id);

-- Campaigns & Contracts
CREATE INDEX idx_contracts_brand_id ON contracts (brand_id);
CREATE INDEX idx_contract_payments_contract_id ON contract_payments (
                                                                     contract_id
    );
CREATE INDEX idx_campaigns_contract_id ON campaigns (contract_id);
CREATE INDEX idx_milestones_campaign_id ON milestones (campaign_id);
CREATE INDEX idx_tasks_milestone_id ON tasks (milestone_id);
CREATE INDEX idx_tasks_assigned_to ON tasks (assigned_to);

-- Content
CREATE INDEX idx_contents_task_id ON contents (task_id);
CREATE INDEX idx_blogs_author_id ON blogs (author_id);
-- For finding blogs by tag
CREATE INDEX idx_blog_tags_tag_id ON blog_tags (tag_id);
CREATE INDEX idx_content_channels_content_id ON content_channels (content_id);
CREATE INDEX idx_content_channels_channel_id ON content_channels (channel_id);

-- E-Commerce
CREATE INDEX idx_products_brand_id ON products (brand_id);
CREATE INDEX idx_product_categories_parent_id ON product_categories (
                                                                     parent_category_id
    );
-- For finding products by category
CREATE INDEX idx_product_variants_product_id ON product_variants (product_id);
CREATE INDEX idx_product_stories_variant_id ON product_stories (variant_id);
CREATE INDEX idx_variant_images_variant_id ON variant_images (variant_id);
CREATE INDEX idx_variant_attr_vals_variant_id ON variant_attribute_values (
                                                                           variant_id
    );
CREATE INDEX idx_variant_attr_vals_attribute_id ON variant_attribute_values (
                                                                             attribute_id
    );
CREATE INDEX idx_content_products_content_id ON content_products (content_id);
CREATE INDEX idx_content_products_product_id ON content_products (product_id);

-- Cart, Order, and Review
CREATE INDEX idx_carts_user_id ON carts (user_id);
CREATE INDEX idx_cart_items_cart_id ON cart_items (cart_id);
CREATE INDEX idx_cart_items_variant_id ON cart_items (variant_id);
CREATE INDEX idx_orders_user_id ON orders (user_id);
CREATE INDEX idx_orders_address_id ON orders (address_id);
CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_variant_id ON order_items (variant_id);
CREATE INDEX idx_pre_orders_user_id ON pre_orders (user_id);
CREATE INDEX idx_pre_orders_variant_id ON pre_orders (variant_id);
CREATE INDEX idx_refund_requests_order_id ON refund_requests (order_id);
CREATE INDEX idx_refund_requests_pre_order_id ON refund_requests (pre_order_id);
CREATE INDEX idx_reviews_variant_id ON reviews (variant_id);
CREATE INDEX idx_reviews_user_id ON reviews (user_id);
CREATE INDEX idx_reviews_order_id ON reviews (order_id);
CREATE INDEX idx_reviews_pre_order_id ON reviews (pre_order_id);

-- Polymorphic & Utility Tables
CREATE INDEX idx_payment_transactions_ref ON payment_transactions (
                                                                   reference_id, reference_type
    );
CREATE INDEX idx_modified_histories_poly_ref ON modified_histories (
                                                                    reference_id, reference_type
    );
CREATE INDEX idx_kpi_metrics_ref ON kpi_metrics (reference_id, reference_type);
CREATE INDEX idx_files_uploaded_by ON files (uploaded_by);


-- =================================================================
-- II. INDEXES FOR STATUS, TYPE, AND OTHER COMMON FILTERS
-- =================================================================
-- Users
CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_is_active ON users (is_active);

-- Notifications
CREATE INDEX idx_notifications_status ON notifications (status);

-- Contracts & Campaigns
CREATE INDEX idx_contracts_status_type ON contracts (status, type);
CREATE INDEX idx_campaigns_status ON campaigns (status);
CREATE INDEX idx_tasks_status ON tasks (status);
CREATE INDEX idx_contents_status ON contents (status);

-- Products & Orders
CREATE INDEX idx_products_type ON products (type);
CREATE INDEX idx_orders_status ON orders (status);
CREATE INDEX idx_pre_orders_status ON pre_orders (status);
CREATE INDEX idx_refund_requests_status ON refund_requests (status);

-- =================================================================
-- III. OPTIONAL BUT HIGHLY RECOMMENDED INDEXES
-- =================================================================
-- For enabling efficient text search (e.g., for search bars)
-- NOTE: You must first enable the extension in your database by running: CREATE
-- EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_products_name_trgm ON products USING GIN (name gin_trgm_ops);
CREATE INDEX idx_contents_title_trgm ON contents USING GIN (title gin_trgm_ops);
CREATE INDEX idx_tags_name_trgm ON tags USING GIN (name gin_trgm_ops);
CREATE INDEX idx_brands_name_trgm ON brands USING GIN (name gin_trgm_ops);

-- For efficiently querying data inside JSONB fields
CREATE INDEX idx_users_profile_data_gin ON users USING GIN (profile_data);
CREATE INDEX idx_contents_body_gin ON contents USING GIN (body);
CREATE INDEX idx_product_stories_content_gin ON product_stories USING GIN (content);

