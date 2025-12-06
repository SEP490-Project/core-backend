-- Table
CREATE TABLE IF NOT EXISTS product_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL,
    variant_id UUID,
    user_id UUID,
    order_item_id UUID,
    pre_order_id UUID,
    rating_stars INTEGER NOT NULL CHECK (rating_stars >= 1 AND rating_stars <= 5),
    comment TEXT,
    assets_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
    );

-- Foreign keys
-- NOTE: assumptions: target tables are `products`, `product_variants`, `users`, `orders`, `pre_orders`.
-- If your schema uses different table names (for example order_items), adjust accordingly.
ALTER TABLE product_reviews
  ADD CONSTRAINT fk_product_reviews_product FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
  ADD CONSTRAINT fk_product_reviews_variant FOREIGN KEY (variant_id) REFERENCES product_variants(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_product_reviews_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_product_reviews_order FOREIGN KEY (order_item_id) REFERENCES order_items(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_product_reviews_preorder FOREIGN KEY (pre_order_id) REFERENCES pre_orders(id) ON DELETE SET NULL;


ALTER TABLE orders ADD COLUMN is_review boolean NOT NULL DEFAULT FALSE;
