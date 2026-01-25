ALTER TABLE product_variants
    ADD COLUMN weight INTEGER,    -- in grams
    ADD COLUMN height INTEGER,    -- in centimeters
    ADD COLUMN length INTEGER,    -- in centimeters
    ADD COLUMN width INTEGER;  -- in centimeters

COMMENT ON COLUMN product_variants.weight IS 'in grams';
COMMENT ON COLUMN product_variants.height IS 'in centimeters';
COMMENT ON COLUMN product_variants.length IS 'in centimeters';
COMMENT ON COLUMN product_variants.width IS 'in centimeters';


ALTER TABLE order_items
    ADD COLUMN weight INTEGER,    -- in grams
    ADD COLUMN height INTEGER,    -- in centimeters
    ADD COLUMN length INTEGER,    -- in centimeters
    ADD COLUMN width INTEGER;  -- in centimeters

COMMENT ON COLUMN order_items.weight IS 'in grams';
COMMENT ON COLUMN order_items.height IS 'in centimeters';
COMMENT ON COLUMN order_items.length IS 'in centimeters';
COMMENT ON COLUMN order_items.width IS 'in centimeters';


