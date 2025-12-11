ALTER TABLE contracts
ADD COLUMN deposit_proof_url TEXT;

ALTER TABLE configs
DROP constraint configs_value_type_check ;

ALTER TYPE value_type ADD VALUE 'TIPTAP_JSON'
