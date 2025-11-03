-- Step 1: Add new columns to contracts table
ALTER TABLE contracts
    ADD COLUMN IF NOT EXISTS parent_contract_id UUID REFERENCES contracts (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS title VARCHAR(255),
    ADD COLUMN IF NOT EXISTS brand_tax_number VARCHAR(100),
    ADD COLUMN IF NOT EXISTS brand_representative_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS brand_representative_role VARCHAR(255),
    ADD COLUMN IF NOT EXISTS brand_representative_phone VARCHAR(20),
    ADD COLUMN IF NOT EXISTS brand_representative_email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS brand_bank_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS brand_account_number VARCHAR(255),
    ADD COLUMN IF NOT EXISTS contract_file_url TEXT,
    ADD COLUMN IF NOT EXISTS proposal_file_url TEXT;

-- Step 2: Migrate existing data from brands to contracts (if any contracts exist)
-- This will copy the brand representative info to existing contracts
UPDATE contracts c
SET brand_tax_number = b.tax_number,
    brand_representative_name = b.representative_name,
    brand_representative_role = b.representative_role,
    brand_representative_phone = b.representative_phone,
    brand_representative_email = b.representative_email
FROM brands b
WHERE c.brand_id = b.id
  AND b.tax_number IS NOT NULL;  -- Only update if brand has these fields populated

ALTER TABLE contracts
  ADD COLUMN IF NOT EXISTS contract_number VARCHAR(255);

-- Populate contract_number for existing rows if empty: use prefix + short id
UPDATE contracts
SET contract_number = ('CN-' || substring(id::text from 1 for 8))
WHERE contract_number IS NULL;

-- Step 3: Drop columns from brands table
ALTER TABLE brands
    DROP COLUMN IF EXISTS tax_number,
    DROP COLUMN IF EXISTS representative_name,
    DROP COLUMN IF EXISTS representative_role,
    DROP COLUMN IF EXISTS representative_phone,
    DROP COLUMN IF EXISTS representative_email;

-- Step 4: Update representative fields in contracts to be nullable (they were NOT
-- NULL before)
-- This allows flexibility as not all contracts require all representative details
ALTER TABLE contracts
    ALTER COLUMN representative_name DROP NOT NULL,
    ALTER COLUMN representative_role DROP NOT NULL,
    ALTER COLUMN representative_phone DROP NOT NULL,
    ALTER COLUMN representative_email DROP NOT NULL;

ALTER TABLE contracts
  ADD COLUMN IF NOT EXISTS contract_number VARCHAR(255),
  ADD COLUMN IF NOT EXISTS signed_date DATE,
  ADD COLUMN IF NOT EXISTS signed_location TEXT,
  ADD COLUMN IF NOT EXISTS currency VARCHAR(3) DEFAULT 'VND';

-- Step 5: Map legacy status values into new enum set
-- Assumption mapping: EXPIRED -> COMPLETED, CANCELLED -> TERMINATED, ACTIVE -> ACTIVE
UPDATE contracts
SET status = CASE
  WHEN status = 'EXPIRED' THEN 'COMPLETED'
  WHEN status = 'CANCELLED' THEN 'TERMINATED'
  WHEN status = 'ACTIVE' THEN 'ACTIVE'
  ELSE 'DRAFT'
END
WHERE status IS NOT NULL;

-- Step 6: Replace old check constraints with new ones matching the domain model
ALTER TABLE contracts
  DROP CONSTRAINT IF EXISTS contracts_type_check,
  DROP CONSTRAINT IF EXISTS contracts_status_check;

ALTER TABLE contracts
  ADD CONSTRAINT contracts_type_check CHECK (type IN ('ADVERTISING', 'AFFILIATE', 'BRAND_AMBASSADOR', 'CO_PRODUCING')),
  ADD CONSTRAINT contracts_status_check CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'TERMINATED'));

-- Step 7: Ensure JSONB columns are NOT NULL (model enforces non-null)
UPDATE contracts SET financial_terms = '{}'::jsonb WHERE financial_terms IS NULL;
UPDATE contracts SET scope_of_work = '{}'::jsonb WHERE scope_of_work IS NULL;
UPDATE contracts SET legal_terms = '{}'::jsonb WHERE legal_terms IS NULL;
ALTER TABLE contracts ALTER COLUMN financial_terms SET NOT NULL;
ALTER TABLE contracts ALTER COLUMN scope_of_work SET NOT NULL;
ALTER TABLE contracts ALTER COLUMN legal_terms SET NOT NULL;


-- Step 5: Add comments for documentation
COMMENT ON COLUMN contracts.brand_tax_number IS 'Brand tax number stored in contract for record-keeping at time of signing';
COMMENT ON COLUMN contracts.brand_representative_name IS 'Brand representative name at time of contract signing';
COMMENT ON COLUMN contracts.brand_representative_role IS 'Brand representative role at time of contract signing';
COMMENT ON COLUMN contracts.brand_representative_phone IS 'Brand representative phone at time of contract signing';
COMMENT ON COLUMN contracts.brand_representative_email IS 'Brand representative email at time of contract signing';
COMMENT ON COLUMN contracts.brand_bank_name IS 'Brand bank name for contract payments';
COMMENT ON COLUMN contracts.brand_account_number IS 'Brand bank account number for contract payments';
COMMENT ON COLUMN contracts.parent_contract_id IS 'Reference to parent contract for amendments or related contracts';
COMMENT ON COLUMN contracts.title IS 'Contract title/name for easy identification';
COMMENT ON COLUMN contracts.representative_name IS 'KOL/Influencer representative name';
COMMENT ON COLUMN contracts.representative_role IS 'KOL/Influencer representative role';
COMMENT ON COLUMN contracts.representative_phone IS 'KOL/Influencer representative phone';
COMMENT ON COLUMN contracts.representative_email IS 'KOL/Influencer representative email';

