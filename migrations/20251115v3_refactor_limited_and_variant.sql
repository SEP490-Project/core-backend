ALTER TABLE limited_products
DROP COLUMN IF EXISTS max_stock,
    DROP COLUMN IF EXISTS pre_order_limit,
    DROP COLUMN IF EXISTS pre_order_count;

ALTER TABLE product_variants
    ADD COLUMN IF NOT EXISTS max_stock INTEGER,
    ADD COLUMN IF NOT EXISTS pre_order_limit INTEGER,
    ADD COLUMN IF NOT EXISTS pre_order_count INTEGER;

ALTER TABLE pre_orders
DROP CONSTRAINT IF EXISTS pre_orders_status_check;

ALTER TABLE pre_orders
    ADD CONSTRAINT pre_orders_status_check
        CHECK (
            status::TEXT = ANY (ARRAY[
    'PENDING',
    'PAID',
    'PRE_ORDERED',
    'CANCELLED',
    'STOCK_READY',
    'STOCK_PREPARING',
    'AWAITING_PICKUP',
    'IN_TRANSIT',
    'DELIVERED',
    'RECEIVED'
    ]::TEXT[])
    );

ALTER TABLE limited_products
    ALTER COLUMN premiere_date SET DATA TYPE TIMESTAMPTZ USING premiere_date::timestamptz;

ALTER TABLE limited_products
    ALTER COLUMN availability_start_date SET DATA TYPE TIMESTAMPTZ USING availability_start_date::timestamptz;

ALTER TABLE limited_products
    ALTER COLUMN availability_end_date SET DATA TYPE TIMESTAMPTZ USING availability_end_date::timestamptz;

ALTER TABLE pre_orders
    ADD COLUMN is_self_picked_up  BOOLEAN DEFAULT false NOT NULL,
    ADD COLUMN confirmation_image TEXT
    ADD COLUMN user_note TEXT;
