-- Copy user id from user field json to target_user_id
update notification_data nd
set target_user_id = ((nd.user::json->'id')::text)::int
where nd.user is not null;

-- Remove user field
alter table notification_data
drop column "user";

-- Copy member id from member field json to target_user_id if phone is not null
-- After clear member field
-- At the beginning instead of user field, member field was used to store user data
update notification_data nd
set target_user_id = ((nd.member::json->'id')::text)::int, member = null
where nd.member::json->'phone' is not null;