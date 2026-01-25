alter table modified_histories
    rename column change_type to operation

alter table modified_histories
    add column status varchar(20) not null default 'IN_PROGRESS' CHECK (status in ('IN_PROGRESS','COMPLETED','FAILED')),
    add constraint modified_histories_reference_type_chk CHECK (reference_type in ('CONTRACT','CAMPAIGN','MILESTONE','TASK','CONTENT','PRODUCT','BLOG')),
    add constraint modified_histories_operation_chk CHECK (operation in ('CREATE','UPDATE','DELETE')),
    alter column reference_id drop not null;


alter table campaigns
    rename budget_projected to budget;
alter table campaigns
    drop column budget_actual;

alter table configs 
    add column value_type varchar(20) not null default 'STRING' CHECK (value_type in ('STRING', 'NUMBER', 'BOOLEAN', 'JSON')),
    add column updated_by uuid references users(id) on delete set null;

