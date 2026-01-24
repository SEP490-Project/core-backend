-- Add paid_at column to contract_payments table
ALTER TABLE contract_payments ADD COLUMN paid_at TIMESTAMPTZ;

-- Update paid_at from payment_transactions where possible
UPDATE contract_payments cp
SET paid_at = pt.transaction_date
FROM payment_transactions pt
WHERE
    cp.id = pt.reference_id
    AND pt.reference_type = 'CONTRACT_PAYMENT'
    AND pt.status = 'COMPLETED'
    AND cp.status = 'PAID';

-- Fallback to updated_at for PAID payments that didn't match a transaction
UPDATE contract_payments
SET paid_at = updated_at
WHERE
    status = 'PAID' OR status = 'KOL_REFUND_APPROVED'
    AND paid_at IS NULL;
