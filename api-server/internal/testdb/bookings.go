package testdb

// Synthetic fixtures only. Each integration test receives its own schema.
const Bookings = `
CREATE TABLE st_users (id integer PRIMARY KEY, name text, username text, member_status_id integer, user_age_group_id integer);
CREATE TABLE et_booking_status (id integer PRIMARY KEY, status text);
CREATE TABLE at_trips (id integer PRIMARY KEY, trip_name text, trip_cost_group_id integer);
CREATE TABLE at_bookings (id integer PRIMARY KEY, owner_id integer, trip_id integer, notes text, booking_status_id integer, from_date timestamptz, to_date timestamptz, booking_price numeric, stripe_session_id text);
CREATE TABLE at_booking_people (id integer PRIMARY KEY, owner_id integer, booking_id integer, person_id integer, notes text, created timestamptz, modified timestamptz);
CREATE TABLE at_payments (id integer PRIMARY KEY,booking_id integer);
CREATE TABLE at_trip_cost_groups (id integer PRIMARY KEY);
CREATE TABLE at_trip_costs (id integer PRIMARY KEY, trip_cost_group_id integer, member_status_id integer, user_age_group_id integer, amount numeric);
INSERT INTO st_users VALUES (1,'Owner One','one',1,1),(2,'Owner Two','two',1,1);
INSERT INTO et_booking_status VALUES (1,'Not paid');
INSERT INTO at_trips VALUES (1,'Test trip',1);
INSERT INTO at_bookings VALUES (11,1,1,'first',1,NOW(),NOW()+INTERVAL '1 day',10,NULL),(22,2,1,'second',1,NOW(),NOW()+INTERVAL '1 day',10,NULL);
INSERT INTO at_booking_people VALUES (111,1,11,1,'first',NOW(),NOW()),(222,2,22,2,'second',NOW(),NOW());
INSERT INTO at_trip_costs VALUES (1,1,1,1,10);
ALTER TABLE at_bookings ADD COLUMN created timestamptz DEFAULT CURRENT_TIMESTAMP,ADD COLUMN modified timestamptz DEFAULT CURRENT_TIMESTAMP,ADD COLUMN booking_date date,ADD COLUMN group_booking_id integer,ADD COLUMN payment_date timestamptz,ADD COLUMN amount_paid integer,ADD COLUMN currency text;
ALTER TABLE at_trips ADD COLUMN from_date timestamptz DEFAULT '2025-01-01',ADD COLUMN to_date timestamptz DEFAULT '2030-01-01';
ALTER TABLE at_bookings ADD COLUMN voucher_code text NOT NULL DEFAULT '', ADD COLUMN free_confirmed_at timestamptz;
CREATE TABLE at_booking_vouchers (code text PRIMARY KEY, discount_percent numeric, fixed_cost numeric, expiry_date date, is_active boolean, id serial UNIQUE);
`
