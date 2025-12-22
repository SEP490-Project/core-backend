DO $$ BEGIN
    CREATE TYPE contract_payments_status AS ENUM ('PENDING', 'PAID', 'OVERDUE', 'TERMINATED');
    ALTER TABLE contract_payments
        DROP CONSTRAINT contract_payments_status_check,
        alter column status TYPE contract_payments_status using status::contract_payments_status;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;
