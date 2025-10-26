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


ALTER TABLE variant_attributes
ADD COLUMN created_by UUID,
ADD COLUMN updated_by UUID,
ADD CONSTRAINT variant_attributes_created_by_fkey
    FOREIGN KEY (created_by) REFERENCES "users" ("id")
    ON DELETE SET NULL,
ADD CONSTRAINT variant_attributes_updated_by_fkey
    FOREIGN KEY (updated_by) REFERENCES "users" ("id")
    ON DELETE SET NULL;

-- Remove NOT NULL constraint from current_stock
ALTER TABLE product_variants
    ALTER COLUMN current_stock DROP NOT NULL;