alter table campaigns
    drop column if exists budget;

alter table milestones
    drop column if exists budget_percent;
    drop column if exists budget_amount;

