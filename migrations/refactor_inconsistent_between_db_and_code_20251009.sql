-- Description: Refactor inconsistent naming for completion_percentage between DB and
-- code
alter table milestones
    rename column completion_percent to completion_percentage;


-- Description: Change campaign status check constraint from ON_GOING to RUNNING
alter table campaigns
    drop constraint campaigns_status_check;
alter table campaigns
    add constraint campaigns_status_check CHECK (
        status IN ('RUNNING', 'COMPLETED', 'CANCELED')
        );


-- Description: Drop not null constraint on assigned_to in tasks table
-- Newly created tasks do not have to be assigned to a user yet.
alter table tasks
    alter column assigned_to drop not null;

