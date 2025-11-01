-- Thêm các cột để lưu thông tin location GHN
ALTER TABLE shipping_addresses
ADD COLUMN ghn_province_id   INTEGER,
ADD COLUMN ghn_district_id   INTEGER,
ADD COLUMN ghn_ward_code     VARCHAR(16),
ADD COLUMN province_name     VARCHAR(255),
ADD COLUMN district_name     VARCHAR(255),
ADD COLUMN ward_name         VARCHAR(255),
ALTER COLUMN country DROP NOT NULL,
DROP COLUMN state,
DROP COLUMN company;