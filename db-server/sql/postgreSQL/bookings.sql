-- Table: public.at_bookings

-- DROP TABLE IF EXISTS public.at_bookings;

CREATE TABLE IF NOT EXISTS public.at_bookings
(
    id integer NOT NULL DEFAULT nextval('at_bookings_id_seq'::regclass),
    owner_id integer NOT NULL DEFAULT 0,
    notes text COLLATE pg_catalog."default",
    from_date timestamp without time zone,
    to_date timestamp without time zone,
    booking_status_id integer NOT NULL DEFAULT 0,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT at_bookings_pkey PRIMARY KEY (id),
    CONSTRAINT bookings_status_id_fkey FOREIGN KEY (booking_status_id)
        REFERENCES public.et_booking_status (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT owner_id_fkey FOREIGN KEY (owner_id)
        REFERENCES public.st_users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.at_bookings
    OWNER to myuser;

GRANT ALL ON TABLE public.at_bookings TO api_user;

GRANT ALL ON TABLE public.at_bookings TO myuser;

-- Trigger: update_at_bookings_modified

-- DROP TRIGGER IF EXISTS update_at_bookings_modified ON public.at_bookings;

-- Is this needed?????????
--CREATE OR REPLACE FUNCTION update_modified_column()
--RETURNS TRIGGER AS $$
--BEGIN
--   NEW.Modified = NOW();
--   RETURN NEW;
--END;
--$$ LANGUAGE plpgsql;

CREATE TRIGGER update_at_bookings_modified
    BEFORE UPDATE 
    ON public.at_bookings
    FOR EACH ROW
    EXECUTE FUNCTION public.update_modified_column();



-- Table: public.et_booking_status

-- DROP TABLE IF EXISTS public.et_booking_status;

CREATE TABLE IF NOT EXISTS public.et_booking_status
(
    id integer NOT NULL DEFAULT nextval('et_booking_status_id_seq'::regclass),
    status character varying(50) COLLATE pg_catalog."default" NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT et_booking_status_pkey PRIMARY KEY (id)
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.et_booking_status
    OWNER to myuser;


-- Table: public.at_booking_people

-- DROP TABLE IF EXISTS public.at_booking_people;

CREATE TABLE IF NOT EXISTS public.at_booking_people
(
    id integer NOT NULL DEFAULT nextval('at_booking_people_id_seq'::regclass),
    owner_id integer NOT NULL DEFAULT 0,
    booking_id integer NOT NULL DEFAULT 0,
    user_id integer NOT NULL DEFAULT 0,
    notes text COLLATE pg_catalog."default",
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT at_booking_people_pkey PRIMARY KEY (id),
    CONSTRAINT booking_id_fkey FOREIGN KEY (booking_id)
        REFERENCES public.at_bookings (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT owner_id_fkey FOREIGN KEY (owner_id)
        REFERENCES public.st_users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID,
    CONSTRAINT user_id_fkey FOREIGN KEY (user_id)
        REFERENCES public.st_users (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
        NOT VALID
)

TABLESPACE pg_default;

ALTER TABLE IF EXISTS public.at_booking_people
    OWNER to myuser;