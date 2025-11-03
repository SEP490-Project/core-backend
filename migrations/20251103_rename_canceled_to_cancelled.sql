begin
;
DO
$$
    BEGIN
        -- STEP 1: Create new ENUM type if not exists
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'order_status') THEN
            CREATE TYPE order_status AS ENUM ('PENDING', 'PAID', 'REFUNDED', 'CONFIRMED', 'CANCELLED', 'SHIPPED', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED');
            RAISE NOTICE '✅ Created new ENUM type order_status';
        ELSE
            RAISE NOTICE 'order_status enum already exists';
        END IF;

        -- STEP 2: Find and drop the old check constraint if it exists
        ALTER TABLE orders
            DROP CONSTRAINT orders_status_check;

        -- STEP 3: Alter column to ENUM type using safe cast
        ALTER TABLE orders
            ALTER COLUMN status TYPE order_status
                USING status::order_status;
        alter table order_items
            alter column item_status type order_status
                using item_status::order_status;

        RAISE NOTICE '✅ Converted orders.order_status to enum type successfully';
    END
$$
;

DO
$$
    begin
        -- STEP 1: Create new ENUM type if not exists
        IF NOT EXISTS (SELECT 1
                       FROM pg_type
                       WHERE typname = 'campaign_status') THEN
            CREATE TYPE campaign_status AS ENUM ('DRAFT', 'RUNNING', 'COMPLETED', 'CANCELLED');
            RAISE NOTICE '✅ Created new ENUM type campaign_status';
        ELSE
            RAISE NOTICE 'campaign_status enum already exists';
        END IF;

        -- STEP 2: Find and drop the old check constraint if it exists
        ALTER TABLE campaigns
            DROP CONSTRAINT campaigns_status_check;

        -- STEP 3: Alter column to ENUM type using safe cast
        ALTER TABLE campaigns
            ALTER COLUMN status TYPE campaign_status
                USING status::campaign_status;

        RAISE NOTICE '✅ Converted campaigns.campaign_status to enum type successfully';
    end;
$$
;

end
;

