ALTER TABLE product_variants
ADD COLUMN weight INTERGER PRECISION,    -- in grams
ADD COLUMN height INTERGER PRECISION,    -- in centimeters
ADD COLUMN length INTERGER PRECISION,    -- in centimeters
ADD COLUMN width INTERGER PRECISION;     -- in centimeters

COMMENT ON COLUMN product_variants.weight IS 'in grams';
COMMENT ON COLUMN product_variants.height IS 'in centimeters';
COMMENT ON COLUMN product_variants.length IS 'in centimeters';
COMMENT ON COLUMN product_variants.width IS 'in centimeters';


ALTER TABLE order_items
ADD COLUMN weight INTERGER PRECISION,    -- in grams
ADD COLUMN height INTERGER PRECISION,    -- in centimeters
ADD COLUMN length INTERGER PRECISION,    -- in centimeters
ADD COLUMN width INTERGER PRECISION;     -- in centimeters

COMMENT ON COLUMN product_variants.weight IS 'in grams';
COMMENT ON COLUMN product_variants.height IS 'in centimeters';
COMMENT ON COLUMN product_variants.length IS 'in centimeters';
COMMENT ON COLUMN product_variants.width IS 'in centimeters';