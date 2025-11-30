-- Set product brand relationship to not null
ALTER TABLE products
    ALTER COLUMN brand_id DROP NOT NULL,
DROP CONSTRAINT IF EXISTS products_brand_id_fkey,
    ADD CONSTRAINT products_brand_id_fkey FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE SET NULL;


ALTER TABLE order_items
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS item_status;

-- order_items stuff
ALTER TABLE order_items
    ADD COLUMN product_name TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN product_type TEXT,
    ADD COLUMN brand_id UUID,
    ADD COLUMN category_id UUID,
    ADD CONSTRAINT order_items_brand_id_fkey
        FOREIGN KEY (brand_id)
        REFERENCES brands(id)
        ON DELETE SET NULL,
    ADD CONSTRAINT order_items_category_id_fkey
        FOREIGN KEY (category_id)
        REFERENCES product_categories(id)
        ON DELETE SET NULL;


ALTER TABLE users
    ADD COLUMN bank_account TEXT,
    ADD COLUMN bank_name TEXT,
    ADD COLUMN bank_account_holder TEXT;


ALTER TABLE orders
    ADD COLUMN user_bank_account TEXT,
    ADD COLUMN user_bank_name TEXT,
    ADD COLUMN user_bank_account_holder TEXT;
