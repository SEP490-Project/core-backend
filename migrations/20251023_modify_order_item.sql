ALTER TABLE order_items
    RENAME COLUMN manufactring_date TO manufacturing_date;

ALTER TABLE payment_transactions
    ADD COLUMN gateway_id TEXT;