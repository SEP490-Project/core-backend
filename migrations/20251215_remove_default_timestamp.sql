ALTER TABLE product_reviews
    ALTER COLUMN created_at DROP DEFAULT;

ALTER TABLE product_reviews
    ALTER COLUMN updated_at DROP DEFAULT;
