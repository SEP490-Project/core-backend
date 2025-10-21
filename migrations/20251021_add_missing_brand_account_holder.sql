alter table contracts
    add column if not exists brand_account_holder varchar(100);

alter table users
    add column if not exists avatar_url text;

