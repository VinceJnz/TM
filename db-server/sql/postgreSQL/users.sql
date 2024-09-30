-- Table: public.st_users

-- DROP TABLE IF EXISTS public.st_users;

CREATE TABLE IF NOT EXISTS public.st_users
(
    id integer NOT NULL DEFAULT nextval('st_users_id_seq'::regclass),
    name character varying(255) COLLATE pg_catalog."default" NOT NULL,
    username character varying(255) COLLATE pg_catalog."default" NOT NULL,
    email character varying(255) COLLATE pg_catalog."default" NOT NULL,
    user_status_id integer NOT NULL DEFAULT 0,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT st_users_pkey PRIMARY KEY (id),
    CONSTRAINT st_users_email_key UNIQUE (email),
    CONSTRAINT st_users_username_key UNIQUE (username)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.st_users
    OWNER to myuser;

GRANT ALL ON TABLE public.st_users TO api_user;

GRANT ALL ON TABLE public.st_users TO myuser;

-- Trigger: update_st_users_modified

-- DROP TRIGGER IF EXISTS update_st_users_modified ON public.st_users;

-- Is this needed ????????????
-- CREATE OR REPLACE FUNCTION update_modified_column()
-- RETURNS TRIGGER AS $$
-- BEGIN
--    NEW.Modified = NOW();
--    RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql;

CREATE TRIGGER update_st_users_modified
    BEFORE UPDATE 
    ON public.st_users
    FOR EACH ROW
    EXECUTE FUNCTION public.update_modified_column();
