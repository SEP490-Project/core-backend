ALTER TABLE products ADD COLUMN brand_place_holder varchar(255);

CREATE OR REPLACE FUNCTION fill_brand_place_holder()
RETURNS trigger AS
$$
BEGIN
    IF NEW.brand_id IS NOT NULL THEN
SELECT b.name
INTO NEW.brand_place_holder
FROM brands b
WHERE b.id = NEW.brand_id;
END IF;

RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_fill_brand_place_holder
    BEFORE INSERT OR UPDATE OF brand_id
                     ON products
                         FOR EACH ROW
                         EXECUTE FUNCTION fill_brand_place_holder();


-- Data migration script
-- Migration
BEGIN;
-- 1.Update brand_place_holder
UPDATE products p
SET brand_place_holder = b.name
    FROM brands b
WHERE p.brand_id = b.id
  AND p.type = 'STANDARD'
  AND p.brand_id IS NOT NULL
  AND p.brand_place_holder IS NULL;

-- 2.Set BrandID null
UPDATE products
SET brand_id = NULL
WHERE type = 'STANDARD'
  AND brand_place_holder IS NOT NULL;

COMMIT;
