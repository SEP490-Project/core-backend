-- Migration: Add payment calculation fields for AFFILIATE and CO_PRODUCING contracts
-- Created: November 26, 2025
-- Purpose: Support auto-recalculation of payment amounts and payment locking mechanism
ALTER TABLE orders
ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE default NULL;

-- =====================================================
-- 1. Add period tracking fields
-- =====================================================
-- These fields define the payment period boundaries for AFFILIATE/CO_PRODUCING
-- contracts
ALTER TABLE contract_payments
ADD COLUMN IF NOT EXISTS period_start TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS period_end TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN contract_payments.period_start IS 'Start of the payment period (inclusive) for AFFILIATE/CO_PRODUCING contracts';
COMMENT ON COLUMN contract_payments.period_end IS 'End of the payment period (exclusive) for AFFILIATE/CO_PRODUCING contracts';

-- =====================================================
-- 2. Add calculation tracking fields
-- =====================================================
-- Track when calculation was last performed and store detailed breakdown
ALTER TABLE contract_payments
ADD COLUMN IF NOT EXISTS calculated_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS calculation_breakdown JSONB;

COMMENT ON COLUMN contract_payments.calculated_at IS 'Timestamp of last payment amount calculation';
COMMENT ON COLUMN contract_payments.calculation_breakdown IS 'Detailed breakdown of payment calculation (tier breakdown for AFFILIATE, revenue breakdown for CO_PRODUCING)';

-- =====================================================
-- 3. Add payment locking fields
-- =====================================================
-- When a payment link is created, lock the current amount to prevent changes
-- New clicks/revenue during this time will be attributed to the next period
ALTER TABLE contract_payments
ADD COLUMN IF NOT EXISTS locked_amount DECIMAL(15,2),
ADD COLUMN IF NOT EXISTS locked_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS locked_clicks INT8,
ADD COLUMN IF NOT EXISTS locked_revenue DECIMAL(15,2);

COMMENT ON COLUMN contract_payments.locked_amount IS 'Locked payment amount when payment link was created';
COMMENT ON COLUMN contract_payments.locked_at IS 'Timestamp when payment amount was locked for payment processing';
COMMENT ON COLUMN contract_payments.locked_clicks IS 'Locked total clicks count at time of locking (AFFILIATE contracts only)';
COMMENT ON COLUMN contract_payments.locked_revenue IS 'Locked total revenue at time of locking (CO_PRODUCING contracts only)';

-- =====================================================
-- 4. Add index for faster period lookups
-- =====================================================
-- Composite index for querying payments by period (used in auto-recalculation)
CREATE INDEX IF NOT EXISTS idx_contract_payments_period 
ON contract_payments (contract_id, period_start, period_end)
WHERE deleted_at IS NULL;

-- Index for finding locked payments that need processing
CREATE INDEX IF NOT EXISTS idx_contract_payments_locked
ON contract_payments (locked_at)
WHERE locked_at IS NOT NULL AND deleted_at IS NULL;

-- =====================================================
-- 5. Update existing AFFILIATE/CO_PRODUCING payments with period info
-- =====================================================
-- This CTE updates existing contract payments that don't have period info set
-- by calculating the period based on due_date and payment cycle from the contract's
-- financial_terms
-- Note: This is a best-effort migration for existing data.
-- Manual review may be needed for edge cases.
with
    contract_cycles as (
        select
            cp.id as payment_id,
            c.id as contract_id,
            c.type as contract_type,
            cp.due_date,
            case
                when c.type = 'AFFILIATE'
                then (c.financial_terms ->> 'payment_cycle')::text
                when c.type = 'CO_PRODUCING'
                then (c.financial_terms ->> 'profit_distribution_cycle')::text
                else null
            end as payment_cycle
        from contract_payments cp
        join contracts c on c.id = cp.contract_id
        where
            cp.period_start is null
            and c.type in ('AFFILIATE', 'CO_PRODUCING')
            and cp.deleted_at is null
    ),
    calculated_periods as (
        select
            payment_id,
            due_date,
            case
                payment_cycle
                when 'MONTHLY'
                then date_trunc('month', due_date)
                when 'QUARTERLY'
                then date_trunc('quarter', due_date)
                when 'ANNUALLY'
                then date_trunc('year', due_date)
                else due_date
            end as period_start,
            case
                payment_cycle
                when 'MONTHLY'
                then date_trunc('month', due_date) + interval '1 month'
                when 'QUARTERLY'
                then date_trunc('quarter', due_date) + interval '3 months'
                when 'ANNUALLY'
                then date_trunc('year', due_date) + interval '1 year'
                else due_date + interval '1 month'
            end as period_end
        from contract_cycles
        where payment_cycle is not null
    )
    update contract_payments cp
    set
    period_start = calc.period_start,
    period_end = calc.period_end
from calculated_periods calc
where cp.id = calc.payment_id
;

