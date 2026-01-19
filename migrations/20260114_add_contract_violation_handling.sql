-- Migration: Add contract violation handling
-- Date: 2026-01-14
-- Description: Adds support for contract violation tracking, penalty/refund calculations,
--              and links contract payments to milestones

-- ============================================================================
-- STEP 1: Add new contract status values for violation flow
-- ============================================================================

-- Update contract_status check constraint to include violation statuses
ALTER TABLE contracts
DROP constraint IF EXISTS contracts_status_check ;

ALTER TABLE contracts
ADD CONSTRAINT contracts_status_check
CHECK (status IN (
'DRAFT', 'APPROVED', 'ACTIVE', 'COMPLETED', 'INACTIVE', 'TERMINATED',
'BRAND_VIOLATED', 'BRAND_PENALTY_PENDING', 'BRAND_PENALTY_PAID',
'KOL_VIOLATED',
'KOL_REFUND_PENDING',
'KOL_PROOF_SUBMITTED',
'KOL_PROOF_REJECTED',
'KOL_REFUND_APPROVED'
)) ;

-- ============================================================================
-- STEP 2: Add milestone_id column to contract_payments
-- ============================================================================

-- Add milestone_id column to link payments to milestones
ALTER TABLE contract_payments
ADD COLUMN IF NOT EXISTS milestone_id UUID REFERENCES milestones (id) ON DELETE SET NULL ;

-- Create index for milestone_id lookups
CREATE INDEX IF NOT EXISTS idx_contract_payments_milestone_id
ON contract_payments (milestone_id)
WHERE milestone_id IS NOT NULL ;

-- ============================================================================
-- STEP 3: Create contract_violations table
-- ============================================================================

CREATE TABLE IF NOT EXISTS contract_violations (
id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
contract_id UUID NOT NULL REFERENCES contracts (id) ON DELETE CASCADE,
campaign_id UUID REFERENCES campaigns (id) ON DELETE SET NULL,
type VARCHAR (20) NOT NULL CHECK (type IN ('BRAND', 'KOL')),
reason TEXT NOT NULL,

-- Financial details
penalty_amount DECIMAL (15, 2) NOT NULL DEFAULT 0,
refund_amount DECIMAL (15, 2) NOT NULL DEFAULT 0,
total_paid_by_brand DECIMAL (15, 2) NOT NULL DEFAULT 0,
completed_milestones INTEGER NOT NULL DEFAULT 0,
total_milestones INTEGER NOT NULL DEFAULT 0,

-- Calculation breakdown stored as JSONB for auditing
calculation_breakdown JSONB,

-- Proof handling (for KOL refund proof)
proof_status VARCHAR (20) CHECK (proof_status IN ('PENDING',
'APPROVED',
'REJECTED')),
proof_url TEXT,
proof_submitted_at TIMESTAMP WITH TIME ZONE,
proof_submitted_by UUID REFERENCES users (id) ON DELETE SET NULL,
proof_reviewed_at TIMESTAMP WITH TIME ZONE,
proof_reviewed_by UUID REFERENCES users (id) ON DELETE SET NULL,
proof_review_note TEXT,
proof_attempts INTEGER NOT NULL DEFAULT 0,

-- Payment transaction reference (for brand penalty payments)
payment_transaction_id UUID REFERENCES payment_transactions (id) ON DELETE SET NULL,

-- Resolution tracking
resolved_at TIMESTAMP WITH TIME ZONE,
resolved_by UUID REFERENCES users (id) ON DELETE SET NULL,

-- Audit fields
created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW (),
created_by UUID REFERENCES users (id) ON DELETE SET NULL,
updated_by UUID REFERENCES users (id) ON DELETE SET NULL,
deleted_at TIMESTAMP WITH TIME ZONE
) ;

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_contract_violations_contract_id
ON contract_violations (contract_id)
WHERE deleted_at IS NULL ;

CREATE INDEX IF NOT EXISTS idx_contract_violations_campaign_id
ON contract_violations (campaign_id)
WHERE campaign_id IS NOT NULL AND deleted_at IS NULL ;

CREATE INDEX IF NOT EXISTS idx_contract_violations_type
ON contract_violations (type)
WHERE deleted_at IS NULL ;

CREATE INDEX IF NOT EXISTS idx_contract_violations_proof_status
ON contract_violations (proof_status)
WHERE proof_status IS NOT NULL AND deleted_at IS NULL ;

CREATE INDEX IF NOT EXISTS idx_contract_violations_resolved
ON contract_violations (resolved_at)
WHERE deleted_at IS NULL ;

-- ============================================================================
-- STEP 4: Update payment_transactions reference_type constraint
-- ============================================================================

ALTER TYPE payment_transactions_status
ADD VALUE 'CONTRACT_VIOLATION' ;

-- ============================================================================
-- STEP 5: Add comments for documentation
-- ============================================================================

COMMENT ON TABLE contract_violations IS 'Stores contract violation records with financial calculations and proof handling' ;

COMMENT ON COLUMN contract_violations.type IS 'BRAND = brand violated contract, KOL = KOL violated contract' ;
COMMENT ON COLUMN contract_violations.penalty_amount IS 'Penalty amount brand must pay (for brand violations)' ;
COMMENT ON COLUMN contract_violations.refund_amount IS 'Amount KOL must refund (for KOL violations)' ;
COMMENT ON COLUMN contract_violations.total_paid_by_brand IS 'Total amount brand has already paid' ;
COMMENT ON COLUMN contract_violations.calculation_breakdown IS 'JSONB with detailed calculation for auditing' ;
COMMENT ON COLUMN contract_violations.proof_status IS 'Status of KOL refund proof: PENDING, APPROVED, REJECTED' ;
COMMENT ON COLUMN contract_violations.proof_url IS 'URL to proof document uploaded by KOL' ;

COMMENT ON COLUMN contract_payments.milestone_id IS 'Links payment to corresponding campaign milestone for violation tracking' ;

-- ============================================================================
-- STEP 6: Create function for updated_at trigger
-- ============================================================================

-- Create or replace function for auto-updating updated_at
CREATE OR REPLACE FUNCTION update_contract_violations_updated_at ()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql ;

-- Create trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS trigger_contract_violations_updated_at ON contract_violations ;
CREATE TRIGGER trigger_contract_violations_updated_at
BEFORE UPDATE ON contract_violations
FOR EACH ROW
EXECUTE FUNCTION update_contract_violations_updated_at () ;
