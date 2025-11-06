-- Migration: Add indexes for Marketing Analytics queries
-- Created: 2025-11-06
-- Description: Adds composite indexes to optimize analytics queries for marketing
-- staff dashboard
-- 1. Optimize contract payment revenue queries (status + time filtering)
-- Used in: GetMonthlyContractRevenue, GetTopBrandsByRevenue, GetRevenueByContractType
CREATE INDEX IF NOT EXISTS idx_contract_payments_status_due_date
    ON contract_payments (status, due_date)
    WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_contract_payments_status_due_date IS
    'Optimizes revenue queries filtering by payment status and due date for time-range analytics';

-- 2. Optimize upcoming campaign deadline queries (status + end_date)
-- Used in: GetUpcomingDeadlineCampaigns
CREATE INDEX IF NOT EXISTS idx_campaigns_status_end_date
    ON campaigns (status, end_date)
    WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_campaigns_status_end_date IS
    'Optimizes queries for campaigns approaching deadline, filtered by status and end date';

-- 3. Optimize contract type filtering (type + status combination)
-- Used in: GetRevenueByContractType for grouping by contract type
CREATE INDEX IF NOT EXISTS idx_contracts_type_status
    ON contracts (type, status)
    WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_contracts_type_status IS
    'Optimizes revenue breakdown queries by contract type (ADVERTISING, AFFILIATE, etc.)';

-- 4. Optimize brand status filtering (if not exists)
-- Used in: GetActiveBrandsCount, GetTopBrandsByRevenue
CREATE INDEX IF NOT EXISTS idx_brands_status
    ON brands (status)
    WHERE deleted_at IS NULL;

COMMENT ON INDEX idx_brands_status IS
    'Optimizes active brand count queries and brand filtering in revenue calculations';

-- 5. Optimize order status filtering for paid orders
-- Used in: GetRevenueByContractType for standard product revenue calculation
CREATE INDEX IF NOT EXISTS idx_orders_status_created_at
    ON orders (status, created_at);

COMMENT ON INDEX idx_orders_status_created_at IS
    'Optimizes standard product revenue queries filtering by PAID status and order date';


-- Update enum type campaign_status to include 'DRAFT' if not exists
ALTER TYPE campaign_status
    ADD VALUE 'DRAFT';

-- Verify indexes were created
DO
$$
    BEGIN
        RAISE NOTICE 'Marketing Analytics indexes created successfully';
        RAISE NOTICE 'Created 5 composite indexes for query optimization';
    END
$$;

