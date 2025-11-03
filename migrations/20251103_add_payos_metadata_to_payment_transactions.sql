-- Migration: Add PayOS metadata JSONB field to payment_transactions table
-- Date: 2025-11-03
-- Description: Adds payos_metadata JSONB column to store PayOS-specific data
-- Add Payment transaction status ENUM type if not exists
DO
$$
    BEGIN
        -- STEP 1: Create new ENUM type if not exists
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'payment_transactions_status') THEN
            CREATE TYPE payment_transactions_status AS ENUM (('PENDING', 'COMPLETED', 'FAILED', 'CANCELLED', 'EXPIRED'));
            RAISE NOTICE '✅ Created new ENUM type payment_transactions_status';
        ELSE
            RAISE NOTICE 'payment_transactions_status enum already exists';
        END IF;

        -- STEP 2: Find and drop the old check constraint if it exists
        ALTER TABLE payment_transactions
            DROP CONSTRAINT payment_transactions_status_check,
            drop constraint payment_transactions_method_check;

        -- STEP 3: Alter column to ENUM type using safe cast
        ALTER TABLE payment_transactions
            ALTER COLUMN status TYPE payment_transactions_status
                USING status::payment_transactions_status;

        RAISE NOTICE '✅ Converted orders.order_status to enum type successfully';
    END
$$
;

-- Add payos_metadata column
ALTER TABLE payment_transactions 
ADD COLUMN IF NOT EXISTS payos_metadata JSONB;

-- Create GIN index for efficient JSONB queries
CREATE INDEX IF NOT EXISTS idx_payment_transactions_payos_metadata 
ON payment_transactions USING GIN (payos_metadata);

-- Create composite index for expired link queries
-- This optimizes the cron job that searches for expired pending PayOS payments
CREATE INDEX IF NOT EXISTS idx_payment_transactions_status_method_updated 
ON payment_transactions (status, method, updated_at) 
WHERE status = 'PENDING' AND method = 'PAYOS';

-- Add index for orderCode lookup from webhook
CREATE INDEX IF NOT EXISTS idx_payment_transactions_payos_order_code
ON payment_transactions ((payos_metadata->>'order_code'))
WHERE payos_metadata IS NOT NULL;

-- Comments
COMMENT ON COLUMN payment_transactions.payos_metadata IS 'Stores PayOS-specific payment data including payment_link_id, order_code, checkout_url, qr_code, expiry, and transaction details';

