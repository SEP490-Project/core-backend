alter table products
    add column is_active boolean not null default true;

alter table product_variants
    add column is_active boolean not null default true;

