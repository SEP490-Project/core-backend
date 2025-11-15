ALTER TABLE limited_products
    DROP COLUMN is_free_shipping;

ALTER TABLE limited_products
    RENAME COLUMN bought_limit TO pre_order_limit;

-- 2b) Update default value for new column name
ALTER TABLE limited_products
    ALTER COLUMN pre_order_limit SET DEFAULT 0;

-- 3) Add new column pre_order_count
ALTER TABLE limited_products
    ADD COLUMN pre_order_count INTEGER DEFAULT 0;