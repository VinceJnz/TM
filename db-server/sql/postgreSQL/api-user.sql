-- Connect to the PostgreSQL database as a superuser
-- Replace `postgres` with your superuser if different
-- \c postgres;

-- Step 1: Create a new user
CREATE USER api_user WITH PASSWORD 'api_password';

-- Step 2: Grant necessary privileges
-- Connect to your database
\c mydatabase;

-- Grant all privileges on the database to the new user
GRANT ALL PRIVILEGES ON DATABASE mydatabase TO api_user;

-- Grant all privileges on all tables in the public schema
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO api_user;

-- Grant all privileges on all sequences in the public schema
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO api_user;

-- Grant usage on the schema
GRANT USAGE ON SCHEMA public TO api_user;

-- Grant execute privilege on all functions in the public schema
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO api_user;




-- Role: api_user
-- DROP ROLE IF EXISTS api_user;

--CREATE ROLE api_user WITH
--  LOGIN
--  SUPERUSER
--  INHERIT
--  CREATEDB
--  CREATEROLE
--  NOREPLICATION
--  ENCRYPTED PASSWORD 'md55b0e6f9a043946643c4cdb816129befd';