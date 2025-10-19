CREATE TYPE concept_status AS ENUM ('UNPUBLISHED', 'DRAFT' ,'PUBLISHED');

CREATE TABLE concepts (
      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
      name VARCHAR(255) NOT NULL,
      description TEXT,
      status concept_status NOT NULL DEFAULT 'DRAFT',
      start_date TIMESTAMP WITH TIME ZONE,
      end_date TIMESTAMP WITH TIME ZONE,
      created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
      updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
      banner_url TEXT,
      video_thumbnail TEXT
);

ALTER TABLE limited_products
    ADD COLUMN concept_id UUID;

ALTER TABLE limited_products
    ADD CONSTRAINT limited_products_concept_id_fkey
        FOREIGN KEY (concept_id) REFERENCES concepts (id)
            ON DELETE SET NULL;

--1--1--
ALTER TABLE limited_products
    ADD CONSTRAINT limited_products_concept_id_unique UNIQUE (concept_id);