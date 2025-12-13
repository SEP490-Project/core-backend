ALTER TABLE products
    ADD COLUMN average_rating NUMERIC(3,2) DEFAULT 0;

UPDATE products p
SET average_rating = sub.avg_rating
    FROM (
    SELECT
        product_id,
        ROUND(AVG(rating_stars)::numeric, 2) AS avg_rating
    FROM product_reviews
    WHERE deleted_at IS NULL  -- nếu bạn dùng soft-delete
    GROUP BY product_id
) sub
WHERE p.id = sub.product_id;

-- Trigger
CREATE OR REPLACE FUNCTION update_product_average_rating()
    RETURNS TRIGGER AS $$
DECLARE
v_product_id UUID;
BEGIN
    v_product_id := COALESCE(NEW.product_id, OLD.product_id);

UPDATE products
SET average_rating = COALESCE((
                                  SELECT ROUND(AVG(rating_stars)::numeric, 2)
                                  FROM product_reviews
                                  WHERE product_id = v_product_id
                                    AND deleted_at IS NULL
                              ), 0)
WHERE id = v_product_id;

RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- attach trigger

DROP TRIGGER IF EXISTS trg_update_product_average_rating ON product_reviews;

CREATE TRIGGER trg_update_product_average_rating
    AFTER INSERT OR UPDATE OR DELETE
                    ON product_reviews
                        FOR EACH ROW
                        EXECUTE FUNCTION update_product_average_rating();

-- Check trigger script
SELECT tgname
FROM pg_trigger
WHERE tgrelid = 'product_reviews'::regclass
  AND NOT tgisinternal;


-- remove is_review column
ALTER TABLE orders
DROP COLUMN is_review;

ALTER TABLE order_items
    ADD COLUMN is_review BOOLEAN DEFAULT FALSE;

ALTER TABLE pre_orders
    ADD COLUMN is_review BOOLEAN DEFAULT FALSE;
