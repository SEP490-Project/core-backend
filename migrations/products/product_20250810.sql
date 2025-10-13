ALTER TABLE products
DROP COLUMN IF EXISTS current_stock;

-- Variants
ALTER TABLE product_variants
ADD COLUMN created_by UUID,
ADD COLUMN updated_by UUID,
ADD CONSTRAINT product_variants_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES "users" ("id")
    ON DELETE SET NULL,
ADD CONSTRAINT product_variants_updated_by_fkey
    FOREIGN KEY (updated_by) REFERENCES "users" ("id")
    ON DELETE SET NULL;
