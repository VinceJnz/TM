package testdb

const Auth = `
CREATE TABLE st_token (id serial PRIMARY KEY,user_id integer,name text,host text,token text,session_data text,token_valid boolean,valid_from timestamptz,valid_to timestamptz);
CREATE TABLE st_users (id serial PRIMARY KEY,name text,username text,email text,user_address text,member_code text,user_birth_date timestamptz,user_age_group_id integer,user_account_status_id integer,user_password text,user_account_hidden boolean,provider text,provider_id text);
`
