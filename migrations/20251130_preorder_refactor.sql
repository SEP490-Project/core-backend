ALTER TABLE public.pre_orders
    -- Bank Info
    ADD COLUMN user_bank_account TEXT,
    ADD COLUMN user_bank_name TEXT,
    ADD COLUMN user_bank_account_holder TEXT,

    -- Product fields
    ADD COLUMN product_name TEXT,
    ADD COLUMN description TEXT,
    ADD COLUMN product_type CHARACTER VARYING(255),
    ADD COLUMN brand_id UUID,
    ADD COLUMN category_id UUID,

    -- Add foreign keys
    ADD CONSTRAINT pre_orders_brand_id_fkey
        FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE SET NULL,

    ADD CONSTRAINT pre_orders_category_id_fkey
        FOREIGN KEY (category_id) REFERENCES product_categories(id) ON DELETE SET NULL;

    ALTER TYPE public.pre_order_status ADD VALUE 'REFUND_REQUEST';
    ALTER TYPE public.pre_order_status ADD VALUE 'REFUNDED';