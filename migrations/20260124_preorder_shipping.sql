ALTER TABLE pre_orders
    ADD COLUMN shipping_fee INT DEFAULT 0,
    ADD COLUMN ghn_order_code TEXT;

ALTER TYPE pre_order_status
    ADD VALUE 'SHIPPED';
