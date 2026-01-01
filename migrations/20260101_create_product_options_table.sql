-- Migration: Create product_options table for runtime-configurable enum values
-- Date: 2026-01-01
-- Description: Replaces hardcoded enums (CapacityUnit, ContainerType, DispenserType, AttributeUnit)
--              with database-backed lookup table

-- Step 1: Create the product_options table
CREATE TABLE IF NOT EXISTS product_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR(50) NOT NULL CHECK (type IN ('CAPACITY_UNIT', 'CONTAINER_TYPE', 'DISPENSER_TYPE', 'ATTRIBUTE_UNIT')),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    sort_order INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(type, code)
);

-- Step 2: Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_product_options_type_active ON product_options(type, is_active) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_product_options_code ON product_options(code);

-- Step 3: Seed initial data from existing enum values
INSERT INTO product_options (type, code, name, sort_order) VALUES
-- Capacity Units
('CAPACITY_UNIT', 'ML', 'Milliliter', 1),
('CAPACITY_UNIT', 'L', 'Liter', 2),
('CAPACITY_UNIT', 'G', 'Gram', 3),
('CAPACITY_UNIT', 'KG', 'Kilogram', 4),
('CAPACITY_UNIT', 'OZ', 'Ounce', 5),

-- Container Types
('CONTAINER_TYPE', 'BOTTLE', 'Bottle', 1),
('CONTAINER_TYPE', 'TUBE', 'Tube', 2),
('CONTAINER_TYPE', 'JAR', 'Jar', 3),
('CONTAINER_TYPE', 'STICK', 'Stick', 4),
('CONTAINER_TYPE', 'PENCIL', 'Pencil', 5),
('CONTAINER_TYPE', 'COMPACT', 'Compact', 6),
('CONTAINER_TYPE', 'PALLETE', 'Palette', 7),
('CONTAINER_TYPE', 'SACHET', 'Sachet', 8),
('CONTAINER_TYPE', 'VIAL', 'Vial', 9),
('CONTAINER_TYPE', 'ROLLER_BOTTLE', 'Roller Bottle', 10),

-- Dispenser Types
('DISPENSER_TYPE', 'PUMP', 'Pump', 1),
('DISPENSER_TYPE', 'SPRAY', 'Spray', 2),
('DISPENSER_TYPE', 'DROPPER', 'Dropper', 3),
('DISPENSER_TYPE', 'ROLL_ON', 'Roll On', 4),
('DISPENSER_TYPE', 'TWIST_UP', 'Twist Up', 5),
('DISPENSER_TYPE', 'SQUEEZE', 'Squeeze', 6),
('DISPENSER_TYPE', 'NONE', 'None', 7),

-- Attribute Units
('ATTRIBUTE_UNIT', '%', 'Percent', 1),
('ATTRIBUTE_UNIT', 'MG', 'Milligram', 2),
('ATTRIBUTE_UNIT', 'G', 'Gram', 3),
('ATTRIBUTE_UNIT', 'ML', 'Milliliter', 4),
('ATTRIBUTE_UNIT', 'L', 'Liter', 5),
('ATTRIBUTE_UNIT', 'IU', 'International Unit', 6),
('ATTRIBUTE_UNIT', 'PPM', 'Parts Per Million', 7),
('ATTRIBUTE_UNIT', 'NONE', 'None', 8)
ON CONFLICT (type, code) DO NOTHING;

-- Step 4: Remove CHECK constraints from existing tables (allows any string value)
-- Note: These constraints reference the old enum values

ALTER TABLE product_variants DROP CONSTRAINT IF EXISTS product_variants_capacity_unit_check;
ALTER TABLE product_variants DROP CONSTRAINT IF EXISTS product_variants_container_type_check;
ALTER TABLE product_variants DROP CONSTRAINT IF EXISTS product_variants_dispenser_type_check;

ALTER TABLE variant_attribute_values DROP CONSTRAINT IF EXISTS variant_attribute_values_unit_check;

ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_capacity_unit_check;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_container_type_check;
ALTER TABLE order_items DROP CONSTRAINT IF EXISTS order_items_dispenser_type_check;

ALTER TABLE pre_orders DROP CONSTRAINT IF EXISTS pre_orders_capacity_unit_check;
ALTER TABLE pre_orders DROP CONSTRAINT IF EXISTS pre_orders_container_type_check;
ALTER TABLE pre_orders DROP CONSTRAINT IF EXISTS pre_orders_dispenser_type_check;
