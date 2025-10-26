ALTER TABLE product_categories
    ADD COLUMN icon_url TEXT;

ALTER TABLE products
DROP COLUMN price

ALTER TABLE products ALTER COLUMN is_active SET DEFAULT false;