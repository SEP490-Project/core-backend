alter table contracts
    add column if not exists deposit_percent int default 0 check (deposit_percent >= 0 and deposit_percent <= 100),
    add column if not exists deposit_amount numeric default 0 check (deposit_amount >= 0);

