ALTER TABLE content_channels
ADD COLUMN metadata jsonb;

ALTER TABLE channels
ADD COLUMN vault_path varchar(255);

UPDATE channels
SET vault_path = CASE
    WHEN hashed_access_token ILIKE 'secrets/%' THEN hashed_access_token
    ELSE null
END
WHERE 1 = 1;

