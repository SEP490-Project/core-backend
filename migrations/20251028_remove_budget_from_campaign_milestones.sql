alter table campaigns
    drop column if exists budget;

alter table milestones
    drop column if exists budget_percent,
    drop column if exists budget_amount;

alter table contracts
add column if not exists is_deposit_paid boolean default false;

