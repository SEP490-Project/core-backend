-- Create provinces, districts, wards tables for storing local location data

BEGIN;

CREATE TABLE IF NOT EXISTS provinces (
    id              INTEGER PRIMARY KEY,
    name            VARCHAR(255) NOT NULL,
    country_id      INTEGER,
    code            VARCHAR(64),
    region_id       INTEGER,
    region_cpn      INTEGER,
    is_enable       INTEGER,
    can_update_cod  BOOLEAN,
    status          INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_provinces_name ON provinces (name);
CREATE INDEX IF NOT EXISTS idx_provinces_deleted_at ON provinces (deleted_at);

CREATE TABLE IF NOT EXISTS districts (
    id              INTEGER PRIMARY KEY,
    province_id     INTEGER NOT NULL REFERENCES provinces(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    code            VARCHAR(64),
    type            INTEGER,
    support_type    INTEGER,
    pick_type       INTEGER,
    deliver_type    INTEGER,
    government_code VARCHAR(64),
    is_enable       INTEGER,
    can_update_cod  BOOLEAN,
    status          INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_districts_province_id ON districts (province_id);
CREATE INDEX IF NOT EXISTS idx_districts_name ON districts (name);
CREATE INDEX IF NOT EXISTS idx_districts_deleted_at ON districts (deleted_at);

CREATE TABLE IF NOT EXISTS wards (
    code            VARCHAR(32) PRIMARY KEY,
    district_id     INTEGER NOT NULL REFERENCES districts(id) ON UPDATE CASCADE ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    support_type    INTEGER,
    pick_type       INTEGER,
    deliver_type    INTEGER,
    government_code VARCHAR(64),
    is_enable       INTEGER,
    can_update_cod  BOOLEAN,
    status          INTEGER,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_wards_district_id ON wards (district_id);
CREATE INDEX IF NOT EXISTS idx_wards_name ON wards (name);
CREATE INDEX IF NOT EXISTS idx_wards_deleted_at ON wards (deleted_at);

-- Add FKs from shipping_addresses to new tables (columns already exist from previous migration)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'shipping_addresses' AND column_name = 'ghn_province_id'
    ) THEN
        ALTER TABLE shipping_addresses
            ADD CONSTRAINT IF NOT EXISTS fk_shipping_addresses_province
            FOREIGN KEY (ghn_province_id) REFERENCES provinces(id)
            ON UPDATE CASCADE ON DELETE SET NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'shipping_addresses' AND column_name = 'ghn_district_id'
    ) THEN
        ALTER TABLE shipping_addresses
            ADD CONSTRAINT IF NOT EXISTS fk_shipping_addresses_district
            FOREIGN KEY (ghn_district_id) REFERENCES districts(id)
            ON UPDATE CASCADE ON DELETE SET NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'shipping_addresses' AND column_name = 'ghn_ward_code'
    ) THEN
        ALTER TABLE shipping_addresses
            ADD CONSTRAINT IF NOT EXISTS fk_shipping_addresses_ward
            FOREIGN KEY (ghn_ward_code) REFERENCES wards(code)
            ON UPDATE CASCADE ON DELETE SET NULL;
    END IF;
END $$;

COMMIT;

