alter table milestones
    add column if not exists budget_percent integer default 0,
    add column if not exists budget_amount numeric(15,2) default 0;

alter table campaigns
    drop column if exists budget;

