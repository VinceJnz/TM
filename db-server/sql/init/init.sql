
-- Connect to the PostgreSQL database as a superuser
-- Replace `postgres` with your superuser if different
-- \c postgres;

-- Step 1: Create a new user
CREATE USER api_user WITH PASSWORD 'api_password' SUPERUSER;

--ALTER USER api_user WITH SUPERUSER;

-- Create the database (if not already created)
--CREATE DATABASE mydatabase;

-- Grant ownership of the database to the user
--ALTER DATABASE mydatabase OWNER TO api_user;

-- Step 2: Grant necessary privileges
-- Connect to your database
--\c mydatabase;

-- Grant all privileges on the database to the new user
--GRANT ALL PRIVILEGES ON DATABASE mydatabase TO api_user;

-- Grant all privileges on all tables in the public schema
--GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO api_user;

-- Grant all privileges on all sequences in the public schema
--GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO api_user;

-- Grant usage on the schema
--GRANT USAGE ON SCHEMA public TO api_user;

-- Grant execute privilege on all functions in the public schema
--GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO api_user;

-- Step 3: Create all the tables
-- ToDo:
--  1. Need to add code/settings to update modified columns

----------------------------------------------
-- Application tables
----------------------------------------------

-- Table for adding people to a booking
-- The assumption is that a booking can contain one or more people. group_bookings are not needed as each booking is a group of people.
-- This will not be needed if the group_booking table is activated.
CREATE TABLE IF NOT EXISTS at_booking_people
(
    id SERIAL PRIMARY KEY,
    owner_id integer NOT NULL DEFAULT 0,
    booking_id integer NOT NULL DEFAULT 0,
    person_id integer NOT NULL DEFAULT 0,
    notes text COLLATE pg_catalog."default",
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (owner_id) REFERENCES at_users(id)
    --FOREIGN KEY (booking_id) REFERENCES at_bookings(id)
    --FOREIGN KEY (person_id) REFERENCES at_users(id)
);

-- Table for bookings info
CREATE TABLE IF NOT EXISTS at_bookings (
    id SERIAL PRIMARY KEY,
    owner_id INT NOT NULL DEFAULT 0,  -- Default value set to 0
    trip_id INT NOT NULL DEFAULT 0,  -- Default value set to 0
    --person_id INTEGER NOT NULL DEFAULT 0, -- Removed, not used. This field is only used for the group_bookings functionality. It is not needed for the booking_people functionality.
    notes TEXT,
    from_date TIMESTAMP DEFAULT NULL,
    to_date TIMESTAMP DEFAULT NULL,
    group_booking_id INTEGER, -- Is this booking for a group??
    booking_status_id INT NOT NULL DEFAULT 0,  -- Default value set to 0
    booking_date DATE NOT NULL, -- The date the booking was made
    payment_date DATE NOT NULL, -- The date the booking was paid for
    booking_price NUMERIC(8,2), -- Use NUMERIC for string MONEY values.
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (trip_id) REFERENCES at_trips(id),
    --FOREIGN KEY (owner_id) REFERENCES at_users(id),
    --FOREIGN KEY (person_id) REFERENCES at_users(id),
    --FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id)
);

-- Table for group bookings
-- The assumption to make this work is that a booking only contains one person and bookings can be grouped by being assiociated with a group
CREATE TABLE at_group_bookings (
    id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) NOT NULL,
    Owner_id INT NOT NULL DEFAULT 0,  -- Default value set to 0. This is the user_id for he user that created the group
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing trip cost groups
CREATE TABLE at_trip_cost_groups (
    id SERIAL PRIMARY KEY,
    description VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (at_trip_costs_id) REFERENCES at_trip_costs(id),
);

-- Table for storing trip costs against user_age_groups, seasons
CREATE TABLE at_trip_costs (
    id SERIAL PRIMARY KEY,
    at_trip_cost_group_id INTEGER NOT NULL,
    description VARCHAR(50) NOT NULL, --This could be derived from member_status, user_age_group, season
    member_status_id INTEGER NOT NULL,
    user_age_group_id INTEGER NOT NULL,
    season_id INTEGER NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (trip_cost_group_id) REFERENCES et_trip_cost_group(id),
    --FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_group(id),
    --FOREIGN KEY (season_id) REFERENCES et_season(id),
);

-- Table for trip info
CREATE TABLE IF NOT EXISTS at_trips (
    id SERIAL PRIMARY KEY,
    owner_id INT NOT NULL DEFAULT 0,
    trip_name TEXT NOT NULL,
    location TEXT,
    difficulty_level_id INT NOT NULL DEFAULT 0,
    from_date DATE, -- season can be derived from the date
    to_date DATE, -- season can be derived from the date
    max_participants INTEGER NOT NULL DEFAULT 0,
    trip_status_id INT NOT NULL DEFAULT 0,
    trip_type_id INTEGER NOT NULL DEFAULT 0,
    trip_cost_group_id INTEGER NOT NULL DEFAULT 0,
    created TIMESTAMP DEFAULT NOW(),
    modified TIMESTAMP DEFAULT NOW()
    --FOREIGN KEY (trip_type_id) REFERENCES trip_types(trip_type_id)
);

-- Table for user payments
CREATE TABLE at_user_payments (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    booking_id INTEGER NOT NULL,
    payment_date DATE NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    payment_method VARCHAR(255), -- Example: 'credit_card', 'paypal', etc.
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (user_id) REFERENCES at_users(id),
    --FOREIGN KEY (booking_id) REFERENCES at_bookings(id)
);

----------------------------------------------
-- Enumeration tables
----------------------------------------------

-- Table for booking status
CREATE TABLE IF NOT EXISTS et_booking_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL, -- Example: 'new', 'cancelled', 'success'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- To be changed to just a flag ?????????????????
-- Table for user membership status
CREATE TABLE et_member_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(255) NOT NULL, -- Example: 'yes', 'no' ??????????
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for seasons
CREATE TABLE et_seasons (
    id SERIAL PRIMARY KEY,
    season VARCHAR(255) NOT NULL, -- Example: 'summer', 'winter', 'off-peak'
    start_day INTEGER NOT NULL, -- Specify the day of the year this season starts: count from 1st-Jan
    length INTEGER NOT NULL, -- Specify the length of the season in days
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for trip difficulty
CREATE TABLE IF NOT EXISTS et_trip_difficulty (
    id SERIAL PRIMARY KEY,
    level VARCHAR(50) NOT NULL,  -- Example: 'Medium Fit', 'Slow Fit', 'Family', 'All'
    level_short VARCHAR(3) NOT NULL,  -- Example: 'MF', 'SF', 'F', 'A'
    description VARCHAR(255) NOT NULL,  -- Example: Explaination of what medium fit trip means
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for trip status
CREATE TABLE IF NOT EXISTS et_trip_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,  -- Example: 'Open', 'Closed', 'Cancelled', 'Postponed', 'Full'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for trip type
CREATE TABLE et_trip_type (
    id SERIAL PRIMARY KEY,
    type VARCHAR(255) NOT NULL, -- Example: 'hiking', 'camping', 'rafting', 'cycling', 'skiing'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for user age group
CREATE TABLE et_user_age_groups (
    id SERIAL PRIMARY KEY,
    age_group VARCHAR(255) NOT NULL, -- Example: 'infant', 'child', 'youth', 'adult', 'senior', 'life'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- To be removed. This is superceeded by et_user_account_status
-- Table for user status
--CREATE TABLE et_user_status (
--  id SERIAL PRIMARY KEY,
--    status VARCHAR(255) NOT NULL, -- Example: 'current', 'expired', 'cancelled', 'non-member' ??????????
--    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
--    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
--);


----------------------------------------------
-- Security tables
----------------------------------------------

-- Table for token info
CREATE TABLE st_token (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    name VARCHAR(45) NOT NULL,
    host VARCHAR(45) DEFAULT NULL,
    token VARCHAR(45) DEFAULT NULL,
    token_valid BOOLEAN NOT NULL DEFAULT FALSE,
    valid_from TIMESTAMP with time zone,
    valid_to TIMESTAMP with time zone,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (token_valid_id) REFERENCES et_token_valid(id)
);

-- Table for user info
CREATE TABLE IF NOT EXISTS st_users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL UNIQUE, -- This includes an index to ensure uniqueness
    email VARCHAR(255) NOT NULL UNIQUE, -- This includes an index to ensure uniqueness
    user_address VARCHAR(255),
    member_code VARCHAR(20),
    user_birth_date DATE NOT NULL, --This can be used to calculate what age group to apply
    user_age_group_id INTEGER NOT NULL DEFAULT 0,
    member_status_id INTEGER NOT NULL DEFAULT 0, -- Example: 'yes', 'no'
    user_password VARCHAR(45) DEFAULT NULL, -- This will probably not be used (see: salt, verifier)
    salt BYTEA DEFAULT NULL, -- varbinary(30)
    verifier BYTEA DEFAULT NULL, -- varbinary(500)
    user_account_status_id INT NOT NULL DEFAULT 1, -- Example: 'current', 'disabled', 'new', 'verified', 'password reset required'
    user_account_hidden BOOLEAN, -- Hide the user account from the public, Admins can still see it
    webauthn_user_id BYTEA DEFAULT NULL, -- varbinary(50). Stores the random ID generated by the server during registration
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_group(id)
    --FOREIGN KEY (user_account_status_id) REFERENCES et_user_account_status(id)
    --FOREIGN KEY (member_status_id) REFERENCES et_member_status(id)
);

CREATE TABLE st_user_group (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    group_id INTEGER NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (user_id) REFERENCES st_users(id)
    --FOREIGN KEY (group_id) REFERENCES st_group(id)
);

CREATE TABLE st_group (
    id SERIAL PRIMARY KEY,
    name VARCHAR(45) DEFAULT NULL, -- Example: 'SysAdmin', 'Admin', 'User'
    description VARCHAR(45) DEFAULT NULL,
    admin_flag BOOLEAN DEFAULT FALSE,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE st_group_resource (
    id SERIAL PRIMARY KEY,
    group_id INTEGER NOT NULL,
    resource_id INTEGER NOT NULL, -- ID for the url to to access the resource
    access_level_id INTEGER NOT NULL, -- Example: 'none', 'get', 'post', 'put', 'delete' (OR: 'none', 'select', 'insert', 'update', 'delete')
    access_type_id INTEGER NOT NULL, -- Example: 'admin', 'owner', 'user' ????? don't know if this is useful
    admin_flag BOOLEAN DEFAULT FALSE,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (group_id) REFERENCES st_group(id)
    --FOREIGN KEY (resource_id) REFERENCES st_resource(id)
    --FOREIGN KEY (access_level_id) REFERENCES et_access_level(id)
    --FOREIGN KEY (access_type_id) REFERENCES et_access_type(id)
);

-- Looking at how to deal with column restrictions
-- One option is to store sql code in the db for each group resource. This could get messy.
CREATE TABLE st_resource_column (
    id SERIAL PRIMARY KEY,
    resource_id INTEGER NOT NULL, -- ID for the url to to access the resource
    access_level_id INTEGER NOT NULL,
    access_type_id INTEGER NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (resource_id) REFERENCES st_resource(id)
);

-- Table stores resource names, used in security access selection in groups etc.
CREATE TABLE et_resource (
    id SERIAL PRIMARY KEY,
    name VARCHAR(45) DEFAULT NULL, -- Example: 'trips', 'users', 'bookings', 'member_status' (the url to to access the resource)
    description VARCHAR(45) DEFAULT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for user account status
CREATE TABLE et_user_account_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(255) NOT NULL, -- Example: 'current', 'disabled', 'new', 'verified', 'password reset required'
    description VARCHAR(255), 
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO et_user_account_status (status, description)
VALUES 
('Account New', 'A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin.'),
('Account Verified', 'The email address has been verified. An Admin now needs to activate the account.'),
('Account Active', 'An account that has been activated, and is currently active.'),
('Account Disabled', 'An account that has been disabled.'),
('Account Reset Required', 'The account is flagged for a password reset. The user will be informed at the next login.');


CREATE TABLE et_access_level (
    id SERIAL PRIMARY KEY,
    name VARCHAR(45) DEFAULT NULL, -- Example: 'none', 'get', 'post', 'put', 'delete' (OR: 'none', 'select', 'insert', 'update', 'delete')
    description VARCHAR(45) DEFAULT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- I think this can be removed and replaced with an admin flag, or just remove altogether?????
CREATE TABLE et_access_type (
    id SERIAL PRIMARY KEY,
    name VARCHAR(45) DEFAULT NULL, -- Example: 'admin', 'owner', 'user' ????? don't know if this is useful
    description VARCHAR(45) DEFAULT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE et_token_valid (
    id SERIAL PRIMARY KEY,
    name VARCHAR(45) DEFAULT NULL, -- Example: 'No', 'Yes'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing WebAuthn credentials
CREATE TABLE st_webauthn_credentials (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    credential_data JSONB NOT NULL,
    device_metadata JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for fast lookup by user_id and credential_id
-- Uncomment the following lines to create indexes for faster lookups
--CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);
--CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_credential_id ON webauthn_credentials(credential_id);

-- Create triggers to update the modified column automatically

-- 1. Create the trigger function
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified := NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. Generate the CREATE TRIGGER statements for all tables with a "modified" column
-- This will generate the SQL statements to create triggers for all tables that have a "modified" column
-- You can run this query to see the generated SQL statements
-- and then copy them to execute them to create the triggers.
SELECT
    'CREATE TRIGGER set_modified_timestamp_' || table_name ||
    ' BEFORE UPDATE ON ' || quote_ident(table_name) ||
    ' FOR EACH ROW EXECUTE FUNCTION update_modified_column();'
FROM information_schema.columns
WHERE column_name = 'modified'
  AND table_schema = 'public';

-- 2. Create the trigger for each table that needs it
