-- 1. Bảng contracts
ALTER TABLE contracts
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 2. Bảng contract_payments
ALTER TABLE contract_payments
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 3. Bảng campaigns
ALTER TABLE campaigns
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 4. Bảng milestones
ALTER TABLE milestones
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 5. Bảng tasks
ALTER TABLE tasks
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- 6. Bảng contents
ALTER TABLE contents
    ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE SET NULL,
ADD COLUMN updated_by UUID REFERENCES users(id) ON DELETE SET NULL;