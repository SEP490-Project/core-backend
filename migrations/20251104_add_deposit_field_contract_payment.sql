ALTER TABLE contract_payments
    ADD COLUMN is_deposit BOOL DEFAULT false;

alter table contracts 
    add column reject_reason TEXT;

ALTER TABLE campaigns 
    ADD COLUMN reject_reason TEXT;

