-- Begin transaction to ensure ACID principles
DO
$$
    BEGIN
        -- 1. Create Representative columns in the Brand table
        ALTER TABLE brands
            ADD COLUMN tax_number                VARCHAR(100),
            ADD COLUMN representative_name       VARCHAR(255),
            ADD COLUMN representative_role       VARCHAR(100),
            ADD COLUMN representative_email      VARCHAR(255),
            ADD COLUMN representative_phone      VARCHAR(25),
            ADD COLUMN representative_citizen_id VARCHAR(100);

        -- 2. Migrate data from Contracts to Brands
        UPDATE brands b
        SET tax_number           = c.representative_tax_number,
            representative_name  = c.representative_name,
            representative_role  = c.representative_role,
            representative_email = c.representative_email,
            representative_phone = c.representative_phone
        FROM contracts c
        WHERE c.brand_id = b.id;

        -- 3. Drop Representative columns from the Contracts table
        ALTER TABLE contracts
            DROP COLUMN representative_tax_number,
            DROP COLUMN representative_name,
            DROP COLUMN representative_role,
            DROP COLUMN representative_email,
            DROP COLUMN representative_phone;

    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'Migration failed, rolling back changes.';
            RAISE;
    END;
$$ LANGUAGE PLPGSQL ;

