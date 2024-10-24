
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

CREATE TABLE IF NOT EXISTS at_trips (
    id SERIAL PRIMARY KEY,
    owner_ID INT NOT NULL DEFAULT 0,
    trip_name TEXT NOT NULL,
    location TEXT,
    difficulty_level_id INT NOT NULL DEFAULT 0,
    from_date DATE, -- season can be derived from the date
    to_date DATE, -- season can be derived from the date
    max_participants INTEGER NOT NULL DEFAULT 0,
    trip_status_ID INT NOT NULL DEFAULT 0,
    trip_type_id INTEGER NOT NULL DEFAULT 0,
    at_trip_cost_group_id INTEGER NOT NULL DEFAULT 0,
    created TIMESTAMP DEFAULT NOW(),
    modified TIMESTAMP DEFAULT NOW()
    --FOREIGN KEY (trip_type_id) REFERENCES trip_types(trip_type_id)
);

-- Table for trip type
CREATE TABLE et_trip_type (
    id SERIAL PRIMARY KEY,
    type VARCHAR(255) NOT NULL, -- Example: 'hiking', 'camping', 'rafting', 'cycling', 'skiing'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS et_trip_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,  -- Example: 'Open', 'Closed', 'Cancelled', 'Postponed', 'Full'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS et_trip_difficulty (
    id SERIAL PRIMARY KEY,
    level VARCHAR(50) NOT NULL,  -- Example: 'Medium Fit', 'Slow Fit', 'Family', 'All'
    level_short VARCHAR(3) NOT NULL,  -- Example: 'MF', 'SF', 'F', 'A'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Table for storing trip cost groups
CREATE TABLE at_trip_cost_groups (
    id SERIAL PRIMARY KEY,
    description VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (at_trip_costs_id) REFERENCES at_trip_costs(id),
);

-- Table for storing trip costs against user_categorys, seasons
CREATE TABLE at_trip_costs (
    id SERIAL PRIMARY KEY,
    at_trip_cost_group_id INTEGER NOT NULL,
    description VARCHAR(50) NOT NULL, --This could be derived from user_status, user_category, season
    user_status_id INTEGER NOT NULL,
    user_category_id INTEGER NOT NULL,
    season_id INTEGER NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (trip_cost_group_id) REFERENCES et_trip_cost_group(id),
    --FOREIGN KEY (user_category_id) REFERENCES et_user_category(id),
    --FOREIGN KEY (season_id) REFERENCES et_season(id),
);

CREATE TABLE IF NOT EXISTS at_bookings (
    id SERIAL PRIMARY KEY,
    owner_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    trip_id INT NOT NULL DEFAULT 0,  -- Default value set to 0
    person_id INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    from_date TIMESTAMP DEFAULT NULL,
    to_date TIMESTAMP DEFAULT NULL,
    group_booking_id INTEGER, -- Is this booking for a group??
    booking_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    booking_date DATE NOT NULL,
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (trip_id) REFERENCES at_trips(id),
    --FOREIGN KEY (owner_id) REFERENCES at_users(id),
    --FOREIGN KEY (person_id) REFERENCES at_users(id),
    --FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id)
);

-- Table for group bookings
CREATE TABLE at_group_bookings (
    id SERIAL PRIMARY KEY,
    group_name VARCHAR(255) NOT NULL,
    Owner_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0. This is the user_id for he user that created the group
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- This needs to be removed ************** it will no longer be needed once the booking table is updated.
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

CREATE TABLE IF NOT EXISTS et_booking_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS st_users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    user_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    user_birth_date DATE NOT NULL, --This can be used to calculate what age group to apply
    user_category_id INT NOT NULL DEFAULT 0,
    user_status_id INT NOT NULL DEFAULT 0,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    --FOREIGN KEY (user_category_id) REFERENCES et_user_category(id)
    --FOREIGN KEY (user_status_id) REFERENCES et_user_status(id)
);

-- Table for user status group
CREATE TABLE et_user_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(255) NOT NULL, -- Example: 'current', 'expired', 'cancelled', 'non-member'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


-- Table for user age group
CREATE TABLE et_user_categorys (
    id SERIAL PRIMARY KEY,
    category VARCHAR(255) NOT NULL, -- Example: 'infant', 'child', 'youth', 'adult'
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

-- Table for seasons
CREATE TABLE et_seasons (
    id SERIAL PRIMARY KEY,
    season VARCHAR(255) NOT NULL, -- Example: 'summer', 'winter', 'off-peak'
    start_day INTEGER NOT NULL, -- Specify the day of the year this season starts: count from 1st-Jan
    length INTEGER NOT NULL, -- Specify the length of the season in days
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
