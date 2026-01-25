-- Migration: Add CO_PRODUCING Refund Workflow Fields
-- Date: 2026-01-20
-- Description: Adds fields to support refund workflow when CO_PRODUCING contracts
--              have negative net amounts (company owes brand money).

-- ============================================================================
-- STEP 1: Add refund workflow columns to contract_payments table
-- ============================================================================

ALTER TABLE contract_payments
ADD column IF NOT EXISTS refund_amount DECIMAL (15, 2) NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS refund_proof_url TEXT,
ADD COLUMN IF NOT EXISTS refund_proof_note TEXT,
ADD COLUMN IF NOT EXISTS refund_submitted_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS refund_submitted_by UUID REFERENCES users (id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS refund_reviewed_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS refund_reviewed_by UUID REFERENCES users (id) ON DELETE SET NULL,
ADD COLUMN IF NOT EXISTS refund_reject_reason TEXT,
ADD COLUMN IF NOT EXISTS refund_attempts INT NOT NULL DEFAULT 0 ;

-- ============================================================================
-- STEP 2: Add column documentation
-- ============================================================================

COMMENT ON COLUMN contract_payments.refund_amount IS 'Amount system owes brand when NetAmount < 0 (CO_PRODUCING contracts)' ;
COMMENT ON COLUMN contract_payments.refund_proof_url IS 'URL to proof image of refund payment (bank transfer screenshot)' ;
COMMENT ON COLUMN contract_payments.refund_proof_note IS 'Optional note from Marketing Staff with proof submission' ;
COMMENT ON COLUMN contract_payments.refund_submitted_at IS 'Timestamp when proof was submitted' ;
COMMENT ON COLUMN contract_payments.refund_submitted_by IS 'Marketing Staff user who submitted proof' ;
COMMENT ON COLUMN contract_payments.refund_reviewed_at IS 'Timestamp when Brand reviewed proof' ;
COMMENT ON COLUMN contract_payments.refund_reviewed_by IS 'Brand user who reviewed proof' ;
COMMENT ON COLUMN contract_payments.refund_reject_reason IS 'Reason for rejection if Brand rejected proof' ;
COMMENT ON COLUMN contract_payments.refund_attempts IS 'Number of proof submission attempts (max configurable in admin_config)' ;

-- ============================================================================
-- STEP 3: Convert status column from CHECK constraint to PostgreSQL ENUM type
-- ============================================================================

-- New: System owes brand, awaiting refund proof
ALTER TYPE contract_payments_status ADD VALUE 'KOL_PENDING' ;
-- New: Proof submitted, awaiting brand review
ALTER TYPE contract_payments_status ADD VALUE 'KOL_PROOF_SUBMITTED' ;
-- New: Brand rejected proof, resubmission needed
ALTER TYPE contract_payments_status ADD VALUE 'KOL_PROOF_REJECTED' ;
-- New: Refund completed (terminal state)
ALTER TYPE contract_payments_status ADD VALUE 'KOL_REFUND_APPROVED' ;

-- ============================================================================
-- STEP 4: Create indexes for queries
-- ============================================================================

-- Index for daily job: auto-approve expired refund proofs
CREATE INDEX IF NOT EXISTS idx_contract_payments_refund_review
ON contract_payments (status, refund_submitted_at)
WHERE status = 'KOL_PROOF_SUBMITTED' ;

-- Index for daily job: mark zero-amount payments as PAID after due date
CREATE INDEX IF NOT EXISTS idx_contract_payments_pending_zero_amount
ON contract_payments (status, amount, due_date)
WHERE status = 'PENDING' AND amount = 0 ;

-- Index for Marketing Staff dashboard: pending refunds list
CREATE INDEX IF NOT EXISTS idx_contract_payments_pending_refunds
ON contract_payments (status, due_date DESC)
WHERE status IN ('KOL_PENDING', 'KOL_PROOF_REJECTED') ;

-- Index for Brand dashboard: awaiting review list
CREATE INDEX IF NOT EXISTS idx_contract_payments_awaiting_review
ON contract_payments (status, refund_submitted_at DESC)
WHERE status = 'KOL_PROOF_SUBMITTED' ;

-- ============================================================================
-- VERIFICATION: List new columns
-- ============================================================================

DO $$
    DECLARE
        column_count INT;
    BEGIN
        SELECT COUNT(*) INTO column_count
        FROM information_schema.columns
        WHERE table_name = 'contract_payments'
          AND column_name IN ('refund_amount', 'refund_proof_url', 'refund_submitted_at');

        IF column_count = 3 THEN
            RAISE NOTICE 'Migration completed successfully. All refund columns added.';
        ELSE
            RAISE WARNING 'Migration may be incomplete. Expected 3 refund columns, found %', column_count;
        END IF;
    END$$ ;
