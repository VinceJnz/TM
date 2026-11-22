-- Synthetic fixtures for a disposable release-verification database only.
CREATE ROLE tm_smoke_api LOGIN PASSWORD 'synthetic-api';
INSERT INTO st_users(id,name,username,email,member_status_id,user_age_group_id,user_account_status_id,user_birth_date)
VALUES(1,'Smoke Member <img src=x onerror="window.__tmXss=true">','smoke-member','member@example.test',1,1,2,'1990-01-01'),
      (2,'Other Member','other-member','other@example.test',1,1,2,'1990-01-01');
SELECT setval(pg_get_serial_sequence('st_users','id'),2,true);
INSERT INTO st_user_group(user_id,group_id) VALUES(1,1),(2,1);
-- Exercise the Trip List report without granting broader administrative access.
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT 1,r.id,a.id,s.id FROM et_resource r CROSS JOIN et_access_level a CROSS JOIN et_access_scope s
WHERE r.name='tripsReport' AND a.name='GET' AND s.name='any';
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT 1,r.id,a.id,s.id FROM et_resource r CROSS JOIN et_access_level a CROSS JOIN et_access_scope s
WHERE r.name='trips' AND a.name='PUT' AND s.name='own';
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT 1,r.id,a.id,s.id FROM et_resource r CROSS JOIN et_access_level a CROSS JOIN et_access_scope s
WHERE r.name='tripType' AND a.name='GET' AND s.name='any' ON CONFLICT DO NOTHING;
INSERT INTO at_trip_cost_groups(id,description) VALUES(1,'Synthetic prices');
INSERT INTO at_trip_costs(trip_cost_group_id,member_status_id,user_age_group_id,season_id,amount) VALUES(1,1,1,1,10);
INSERT INTO at_trips(id,owner_id,trip_name,from_date,to_date,trip_status_id,trip_type_id,trip_cost_group_id,max_participants)
VALUES(1,1,'Synthetic trip','2026-01-01','2026-01-03',2,1,1,10);
INSERT INTO at_bookings(id,owner_id,trip_id,notes,from_date,to_date,booking_status_id)
VALUES(1,1,1,'Member booking','2026-01-01','2026-01-02',1),(2,2,1,'Other booking','2026-01-01','2026-01-02',1);
INSERT INTO at_booking_people(owner_id,booking_id,person_id) VALUES(1,1,1),(2,2,2);
INSERT INTO at_bookings(id,owner_id,trip_id,notes,from_date,to_date,booking_status_id)
SELECT n,1,1,'Synthetic page item','2026-01-01','2026-01-02',1 FROM generate_series(3,107)n;
INSERT INTO st_token(user_id,name,host,token,token_valid,valid_from,valid_to)
VALUES(1,'login-otp','tm-smoke','sha256:'||encode(sha256(convert_to('synthetic-release-otp','UTF8')),'hex'),true,NOW()-INTERVAL '1 minute',NOW()+INTERVAL '15 minutes');
INSERT INTO st_users(id,name,username,email,member_status_id,user_age_group_id,user_account_status_id,user_birth_date)
VALUES(3,'Smoke Administrator','smoke-admin','admin@example.test',1,1,2,'1990-01-01');
INSERT INTO st_user_group(user_id,group_id) VALUES(3,3);
SELECT setval(pg_get_serial_sequence('st_users','id'),3,true);
INSERT INTO st_token(user_id,name,host,token,token_valid,valid_from,valid_to)
VALUES(3,'login-otp','tm-smoke','sha256:'||encode(sha256(convert_to('synthetic-admin-otp','UTF8')),'hex'),true,NOW()-INTERVAL '1 minute',NOW()+INTERVAL '15 minutes');
