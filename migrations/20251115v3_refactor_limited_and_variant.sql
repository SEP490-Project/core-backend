ALTER TABLE limited_products
DROP COLUMN IF EXISTS max_stock,
    DROP COLUMN IF EXISTS pre_order_limit,
    DROP COLUMN IF EXISTS pre_order_count;

ALTER TABLE product_variants
    ADD COLUMN IF NOT EXISTS max_stock INTEGER,
    ADD COLUMN IF NOT EXISTS pre_order_limit INTEGER,
    ADD COLUMN IF NOT EXISTS pre_order_count INTEGER;
