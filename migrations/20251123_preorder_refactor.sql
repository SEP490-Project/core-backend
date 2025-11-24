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