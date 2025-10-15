alter table modified_histories
    rename column change_type to operation

alter table modified_histories
    add column status varchar(20) not null default 'IN_PROGRESS' CHECK (status in ('IN_PROGRESS','COMPLETED','FAILED')),
    add constraint modified_histories_reference_type_chk CHECK (reference_type in ('CONTRACT','CAMPAIGN','MILESTONE','TASK','CONTENT','PRODUCT','BLOG')),
    add constraint modified_histories_operation_chk CHECK (operation in ('CREATE','UPDATE','DELETE')),
    alter column reference_id drop not null;

