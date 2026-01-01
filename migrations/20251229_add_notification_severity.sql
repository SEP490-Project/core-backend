create type notification_severity as enum ('INFO', 'WARN', 'ERROR', 'SUCCESS') ;

alter table notifications
add column severity notification_severity default 'INFO' ;
