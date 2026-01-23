alter type payment_transactions_status add value 'REFUNDED' ;
alter type content_status add value 'SCHEDULED' ;

alter table payment_transactions
add column received_by_id uuid ;

-- Back-fill old payment_transactions records with orders and preorders that has been refunded or conpensated
INSERT INTO payment_transactions (
id,
reference_id,
reference_type,
payer_id,
received_by_id,
amount,
method,
status,
transaction_date,
created_at,
updated_at
)
SELECT
gen_random_uuid (),           -- id
po.id,                       -- reference_id
'PREORDER',                 -- reference_type (Assumed enum value)
-- payer_id (NULL implies System/Platform)
'65939f55-23e2-46c8-8884-6158dedb5a5c',
po.user_id,                  -- received_by_id
- (po.total_amount),          -- amount (Negative value for refund)
'BANK_TRANSFER',             -- method
'REFUNDED',                  -- status
po.updated_at,                       -- transaction_date
po.updated_at,                       -- created_at
po.updated_at                        -- updated_at
FROM
pre_orders po
WHERE
po.status IN ('REFUNDED', 'COMPENSATED')
AND NOT EXISTS (
-- Idempotency check: Ensure we haven't already created a transaction for this pre-order
SELECT 1
FROM payment_transactions pt
WHERE pt.reference_id = po.id
AND pt.reference_type = 'PREORDER'
AND pt.status = 'REFUNDED'
) ;

INSERT INTO payment_transactions (
id,
reference_id,
reference_type,
payer_id,
received_by_id,
amount,
method,
status,
transaction_date,
created_at,
updated_at
)
SELECT
gen_random_uuid (),           -- id
o.id,                       -- reference_id
'ORDER',                 -- reference_type (Assumed enum value)
-- payer_id (NULL implies System/Platform)
'65939f55-23e2-46c8-8884-6158dedb5a5c',
o.user_id,                  -- received_by_id
- (o.total_amount),          -- amount (Negative value for refund)
'BANK_TRANSFER',             -- method
'REFUNDED',                  -- status
o.updated_at,                       -- transaction_date
o.updated_at,                       -- created_at
o.updated_at                        -- updated_at
FROM
orders o
WHERE
o.status IN ('REFUNDED', 'COMPENSATED')
AND NOT EXISTS (
-- Idempotency check: Ensure we haven't already created a transaction for this pre-order
SELECT 1
FROM payment_transactions pt
WHERE pt.reference_id = o.id
AND pt.reference_type = 'ORDER'
AND pt.status = 'REFUNDED'
) ;
