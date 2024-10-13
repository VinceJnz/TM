

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
    ID SERIAL PRIMARY KEY,
    Owner_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    Trip_name TEXT NOT NULL,
    Location TEXT,
    Difficulty_level TEXT,  -- Can be changed to ENUM: Easy, Moderate, Difficult
    From_date DATE,
    To_date DATE,
    Max_participants INTEGER NOT NULL DEFAULT 0,
    Trip_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    created TIMESTAMP DEFAULT NOW(),
    modified TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS et_trip_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS et_trip_difficulty (
    id SERIAL PRIMARY KEY,
    level VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS at_bookings (
    ID SERIAL PRIMARY KEY,
    Owner_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    Trip_id INT NOT NULL DEFAULT 0,  -- Default value set to 0
    Notes TEXT,
    From_date TIMESTAMP DEFAULT NULL,
    To_date TIMESTAMP DEFAULT NULL,
    Booking_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE IF NOT EXISTS public.at_booking_people
(
    ID SERIAL PRIMARY KEY,
    owner_id integer NOT NULL DEFAULT 0,
    booking_id integer NOT NULL DEFAULT 0,
    person_id integer NOT NULL DEFAULT 0,
    notes text COLLATE pg_catalog."default",
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.Modified = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_at_bookings_modified
BEFORE UPDATE ON at_bookings
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

CREATE TABLE IF NOT EXISTS et_booking_status (
    id SERIAL PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS st_users (
    ID SERIAL PRIMARY KEY,
    Name VARCHAR(255) NOT NULL,
    Username VARCHAR(255) NOT NULL UNIQUE,
    Email VARCHAR(255) NOT NULL UNIQUE,
    User_status_ID INT NOT NULL DEFAULT 0,  -- Default value set to 0
    Created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    Modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.Modified = NOW();
   RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_st_users_modified
BEFORE UPDATE ON st_users
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

