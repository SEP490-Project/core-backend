DO $$ BEGIN
    CREATE TYPE contract_payments_status AS ENUM ('PENDING', 'PAID', 'OVERDUE', 'TERMINATED');
    ALTER TABLE contract_payments
        DROP CONSTRAINT contract_payments_status_check,
        alter column status TYPE contract_payments_status using status::contract_payments_status;
EXCEPTION
    WHEN duplicate_object THEN null;
END $$ ;

ALTER TABLE contents
ALTER COLUMN tags SET DATA TYPE jsonb USING to_jsonb (tags),
ALTER COLUMN tags SET DEFAULT '[]'::jsonb ;

alter table contract_payments
add column if not exists base_amount decimal (15, 2) default 0,
add column if not exists performance_amount decimal (15, 2) default 0 ;
