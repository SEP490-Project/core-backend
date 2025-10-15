ALTER TABLE users ADD COLUMN date_of_birth DATE;

ALTER TABLE users ADD constraint CHK_ROLES_USERS CHECK (
    role IN (
        'ADMIN',
        'MARKETING_STAFF',
        'CONTENT_STAFF',
        'SALES_STAFF',
        'CUSTOMER',
        'BRAND_PARTNER'
    )
);
