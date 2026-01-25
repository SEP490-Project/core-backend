alter table payment_transactions
    add column created_at timestamptz default current_timestamp;

