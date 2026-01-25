alter table contracts
    add column if not exists brand_account_holder varchar(100);

alter table users
    add column if not exists avatar_url text;

create index idx_channels_name on channels(name);

alter table channels
    add column if not exists home_page_url text;

