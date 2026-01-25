-- Begin transaction to ensure ACID principles
DO
$$
    BEGIN
        -- 1. Create Representative columns in the Brand table
        ALTER TABLE brands
            ADD COLUMN if not exists tax_number                VARCHAR(100),
            ADD COLUMN if not exists representative_name       VARCHAR(255),
            ADD COLUMN if not exists representative_role       VARCHAR(100),
            ADD COLUMN if not exists representative_email      VARCHAR(255),
            ADD COLUMN if not exists representative_phone      VARCHAR(25),
            ADD COLUMN if not exists representative_citizen_id VARCHAR(100);

        -- 2. Migrate data from Contracts to Brands
        UPDATE brands b
        SET tax_number           = c.brand_tax_number,
            representative_name  = c.brand_representative_name,
            representative_role  = c.brand_representative_role,
            representative_email = c.brand_representative_email,
            representative_phone = c.brand_representative_phone
        FROM contracts c
        WHERE c.brand_id = b.id;

        -- 3. Drop Representative columns from the Contracts table
        ALTER TABLE contracts
            DROP COLUMN brand_tax_number,
            DROP COLUMN brand_representative_name,
            DROP COLUMN brand_representative_role,
            DROP COLUMN brand_representative_email,
            DROP COLUMN brand_representative_phone;

        -- 4. Add thumbnail_url column to the Contents table
        ALTER TABLE contents
            ADD COLUMN thumbnail_url TEXT;

    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Migration failed, rolling back changes.';
            RAISE;
    END;
$$ LANGUAGE PLPGSQL ;

