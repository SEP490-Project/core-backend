ALTER TYPE public.order_status ADD VALUE 'AWAITING_PICKUP';

ALTER TABLE public.orders
ADD COLUMN is_self_picked_up BOOLEAN NOT NULL DEFAULT FALSE,
ADD COLUMN self_picked_up_image TEXT;

