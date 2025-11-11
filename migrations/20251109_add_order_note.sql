ALTER TABLE orders
    ADD COLUMN action_notes JSONB;
    ADD COLUMN user_note TEXT;

    ADD COLUMN ghn_order_code TEXT;

ALTER TABLE order_items
    ADD COLUMN item_recap_description TEXT;
    ADD COLUMN item_recap_image TEXT;
