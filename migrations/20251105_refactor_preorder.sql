ALTER TABLE pre_orders
    -- Capacity
    ADD COLUMN IF NOT EXISTS capacity NUMERIC(10,2),

    -- Capacity Unit
    ADD COLUMN IF NOT EXISTS capacity_unit VARCHAR(20)
    CHECK (capacity_unit IN ('ML', 'L', 'G', 'KG', 'OZ')),

    -- Container Type
    ADD COLUMN IF NOT EXISTS container_type VARCHAR(50)
    CHECK (container_type IN (
    'BOTTLE', 'TUBE', 'JAR', 'STICK', 'PENCIL', 'COMPACT',
    'PALLETE', 'SACHET', 'VIAL', 'ROLLER_BOTTLE'
    )),

    -- Dispenser Type
    ADD COLUMN IF NOT EXISTS dispenser_type VARCHAR(50)
    CHECK (dispenser_type IN (
    'PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON',
    'TWIST_UP', 'SQUEEZE', 'NONE'
    )),

    -- Uses
    ADD COLUMN IF NOT EXISTS uses TEXT,

    -- Manufacturing Date
    ADD COLUMN IF NOT EXISTS manufacturing_date DATE,

    -- Expiry Date
    ADD COLUMN IF NOT EXISTS expiry_date DATE,

    -- Instructions
    ADD COLUMN IF NOT EXISTS instructions TEXT,

    -- Attributes Description (JSONB)
    ADD COLUMN IF NOT EXISTS attributes_description JSONB,

    -- Physical Dimensions
    ADD COLUMN IF NOT EXISTS weight INTEGER,  -- grams
    ADD COLUMN IF NOT EXISTS height INTEGER,  -- cm
    ADD COLUMN IF NOT EXISTS length INTEGER,  -- cm
    ADD COLUMN IF NOT EXISTS width INTEGER;   -- cm


-- COPY SHIPPING ADDRESS
ALTER TABLE pre_orders
    ADD COLUMN IF NOT EXISTS full_name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS phone_number VARCHAR(20),
    ADD COLUMN IF NOT EXISTS email VARCHAR(255),
    ADD COLUMN IF NOT EXISTS street TEXT,
    ADD COLUMN IF NOT EXISTS address_line2 TEXT,
    ADD COLUMN IF NOT EXISTS city VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ghn_province_id INTEGER,
    ADD COLUMN IF NOT EXISTS ghn_district_id INTEGER,
    ADD COLUMN IF NOT EXISTS ghn_ward_code VARCHAR(50),
    ADD COLUMN IF NOT EXISTS province_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS district_name VARCHAR(100),
    ADD COLUMN IF NOT EXISTS ward_name VARCHAR(100);


-- Refactor Product
-- 1. Cho phép task_id có thể NULL (hiện tại đã đúng)
ALTER TABLE products
    ALTER COLUMN task_id DROP NOT NULL;

-- 2. Đảm bảo ràng buộc khóa ngoại là SET NULL khi task bị xóa
ALTER TABLE products
DROP CONSTRAINT IF EXISTS products_task_id_fkey,
    ADD CONSTRAINT products_task_id_fkey
    FOREIGN KEY (task_id) REFERENCES tasks(id)
    ON DELETE SET NULL;
