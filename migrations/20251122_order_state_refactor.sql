ALTER TYPE public.order_status ADD VALUE 'REFUND_REQUEST';
ALTER TYPE public.order_status ADD VALUE 'COMPENSATE_REQUEST';
ALTER TYPE public.order_status ADD VALUE 'COMPENSATED';

ALTER TABLE orders
    ADD COLUMN user_resource TEXT;
    ADD COLUMN staff_resource TEXT;
