status order_status NOT NULL,

CREATE TYPE pre_order_status AS ENUM (
    'PENDING',
    'PAID',
    'PRE_ORDERED',
    'CANCELLED',
    'AWAITING_PICKUP',
    'IN_TRANSIT',
    'DELIVERED',
    'RECEIVED',
    'COMPENSATE_REQUEST', -- NEW STATUS
    'COMPENSATE'        -- NEW STATUS
);

ALTER TABLE public.pre_orders
ALTER COLUMN status TYPE pre_order_status
    USING status::TEXT::pre_order_status;

ALTER TABLE public.pre_orders
    ADD COLUMN user_resource TEXT,
    ADD COLUMN staff_resource TEXT;

--Rename states
CREATE TYPE public.pre_order_status_new AS ENUM (
    'PENDING',
    'PAID',
    'PRE_ORDERED',
    'CANCELLED',
    'AWAITING_PICKUP',
    'IN_TRANSIT',
    'DELIVERED',
    'RECEIVED',
    'COMPENSATE_REQUEST',
    'COMPENSATED'
);


ALTER TABLE public.pre_orders
ALTER COLUMN status TYPE pre_order_status_new
    USING status::text::pre_order_status_new;


DROP TYPE public.pre_order_status;

ALTER TYPE public.pre_order_status_new RENAME TO pre_order_status;