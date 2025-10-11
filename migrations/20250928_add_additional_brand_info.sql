alter table brands
    add column if not exists address varchar (255);

alter table products
    add column if not exists task_id uuid references tasks (id) on delete set null;

