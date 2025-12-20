
alter table content_schedules rename to schedules;

alter type reference_type add value if not exists 'CONTENT_CHANNEL' ;
alter type reference_type add value if not exists 'CONTENT' ;
alter type reference_type add value if not exists 'SCHEDULE' ;
alter type reference_type add value if not exists 'MILESTONE' ;
alter type reference_type add value if not exists 'CAMPAIGN' ;
alter type reference_type add value if not exists 'CONTRACT' ;
alter type reference_type add value if not exists 'ORDER' ;
alter type reference_type add value if not exists 'USER' ;
alter type reference_type add value if not exists 'BRAND' ;
alter type reference_type add value if not exists 'OTHER' ;
alter type reference_type add value if not exists 'NOTIFICATION' ;

alter table schedules
drop column if exists content_channel_id,
add column if not exists reference_id uuid,
add column if not exists reference_type reference_type,
add column if not exists type varchar (100),
add column if not exists metadata jsonb ;

alter table contents
add column tags varchar (100) [] ;
