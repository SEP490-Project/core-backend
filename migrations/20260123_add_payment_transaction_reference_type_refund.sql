alter type payment_transactions_status add value 'REFUNDED' ;

alter table payment_transactions
add column received_by_id uuid ;
