--
-- PostgreSQL database dump
--

-- Dumped from database version 13.22 (Debian 13.22-1.pgdg13+1)
-- Dumped by pg_dump version 17.1

-- Started on 2026-01-27 11:20:53

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 3329 (class 1262 OID 16384)
-- Name: mydatabase; Type: DATABASE; Schema: -; Owner: myuser
--

CREATE DATABASE mydatabase WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE mydatabase OWNER TO myuser;

\connect mydatabase

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- TOC entry 4 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: myuser
--

CREATE SCHEMA public;


ALTER SCHEMA public OWNER TO myuser;

--
-- TOC entry 3330 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: myuser
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- TOC entry 251 (class 1255 OID 16726)
-- Name: update_modified_column(); Type: FUNCTION; Schema: public; Owner: myuser
--

CREATE FUNCTION public.update_modified_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.modified := NOW();
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_modified_column() OWNER TO myuser;

--
-- TOC entry 249 (class 1255 OID 16709)
-- Name: vj_execute_dynamic_query(text); Type: FUNCTION; Schema: public; Owner: myuser
--

CREATE FUNCTION public.vj_execute_dynamic_query(query text) RETURNS TABLE(result json)
    LANGUAGE plpgsql
    AS $$
BEGIN
    RETURN QUERY EXECUTE format('SELECT json_agg(t) FROM (%s) t', query);
END;
$$;


ALTER FUNCTION public.vj_execute_dynamic_query(query text) OWNER TO myuser;

--
-- TOC entry 250 (class 1255 OID 16710)
-- Name: vj_execute_multiple_queries(text[]); Type: FUNCTION; Schema: public; Owner: myuser
--

CREATE FUNCTION public.vj_execute_multiple_queries(queries text[]) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
    query text;
BEGIN
    FOREACH query IN ARRAY queries
    LOOP
        EXECUTE query;
    END LOOP;
END;
$$;


ALTER FUNCTION public.vj_execute_multiple_queries(queries text[]) OWNER TO myuser;

--
-- TOC entry 200 (class 1259 OID 16417)
-- Name: at_booking_users_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_booking_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_booking_users_id_seq OWNER TO myuser;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 201 (class 1259 OID 16419)
-- Name: at_booking_people; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_booking_people (
    id integer DEFAULT nextval('public.at_booking_users_id_seq'::regclass) NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    booking_id integer DEFAULT 0 NOT NULL,
    person_id integer DEFAULT 0 NOT NULL,
    notes text,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.at_booking_people OWNER TO myuser;

--
-- TOC entry 202 (class 1259 OID 16433)
-- Name: at_bookings_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_bookings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_bookings_id_seq OWNER TO myuser;

--
-- TOC entry 203 (class 1259 OID 16435)
-- Name: at_bookings; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_bookings (
    id integer DEFAULT nextval('public.at_bookings_id_seq'::regclass) NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    notes text,
    from_date timestamp without time zone,
    to_date timestamp without time zone,
    booking_status_id integer DEFAULT 0 NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    trip_id integer DEFAULT 0 NOT NULL,
    booking_date date,
    group_booking_id integer,
    booking_price numeric(8,2),
    payment_date date,
    amount_paid bigint,
    currency character(3),
    stripe_session_id character(100)
);


ALTER TABLE public.at_bookings OWNER TO myuser;

--
-- TOC entry 204 (class 1259 OID 16449)
-- Name: at_group_bookings_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_group_bookings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_group_bookings_id_seq OWNER TO myuser;

--
-- TOC entry 205 (class 1259 OID 16451)
-- Name: at_group_bookings; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_group_bookings (
    id integer DEFAULT nextval('public.at_group_bookings_id_seq'::regclass) NOT NULL,
    group_name character varying(255) NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.at_group_bookings OWNER TO myuser;

--
-- TOC entry 206 (class 1259 OID 16460)
-- Name: at_trip_cost_groups_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_trip_cost_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_trip_cost_groups_id_seq OWNER TO myuser;

--
-- TOC entry 207 (class 1259 OID 16462)
-- Name: at_trip_cost_groups; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_trip_cost_groups (
    id integer DEFAULT nextval('public.at_trip_cost_groups_id_seq'::regclass) NOT NULL,
    description character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.at_trip_cost_groups OWNER TO myuser;

--
-- TOC entry 208 (class 1259 OID 16470)
-- Name: at_trip_costs_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_trip_costs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_trip_costs_id_seq OWNER TO myuser;

--
-- TOC entry 209 (class 1259 OID 16472)
-- Name: at_trip_costs; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_trip_costs (
    id integer DEFAULT nextval('public.at_trip_costs_id_seq'::regclass) NOT NULL,
    trip_cost_group_id integer NOT NULL,
    description character varying(50),
    member_status_id integer NOT NULL,
    user_age_group_id integer NOT NULL,
    season_id integer NOT NULL,
    amount numeric(10,2) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.at_trip_costs OWNER TO myuser;

--
-- TOC entry 210 (class 1259 OID 16480)
-- Name: at_trips_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_trips_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_trips_id_seq OWNER TO myuser;

--
-- TOC entry 211 (class 1259 OID 16482)
-- Name: at_trips; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_trips (
    id integer DEFAULT nextval('public.at_trips_id_seq'::regclass) NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    trip_name text NOT NULL,
    location text,
    from_date date,
    to_date date,
    max_participants integer DEFAULT 0 NOT NULL,
    created timestamp without time zone DEFAULT now(),
    modified timestamp without time zone DEFAULT now(),
    trip_status_id integer DEFAULT 0 NOT NULL,
    trip_type_id integer DEFAULT 0 NOT NULL,
    difficulty_level_id integer,
    trip_cost_group_id integer,
    description text
);


ALTER TABLE public.at_trips OWNER TO myuser;

--
-- TOC entry 212 (class 1259 OID 16497)
-- Name: at_user_payments_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.at_user_payments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.at_user_payments_id_seq OWNER TO myuser;

--
-- TOC entry 213 (class 1259 OID 16499)
-- Name: at_user_payments; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.at_user_payments (
    id integer DEFAULT nextval('public.at_user_payments_id_seq'::regclass) NOT NULL,
    user_id integer NOT NULL,
    booking_id integer NOT NULL,
    payment_date date NOT NULL,
    amount numeric(10,2) NOT NULL,
    payment_method character varying(255),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.at_user_payments OWNER TO myuser;

--
-- TOC entry 214 (class 1259 OID 16507)
-- Name: et_access_level_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_access_level_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_access_level_id_seq OWNER TO myuser;

--
-- TOC entry 215 (class 1259 OID 16509)
-- Name: et_access_level; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_access_level (
    id integer DEFAULT nextval('public.et_access_level_id_seq'::regclass) NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_access_level OWNER TO myuser;

--
-- TOC entry 216 (class 1259 OID 16517)
-- Name: et_access_type_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_access_type_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_access_type_id_seq OWNER TO myuser;

--
-- TOC entry 217 (class 1259 OID 16519)
-- Name: et_access_type; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_access_type (
    id integer DEFAULT nextval('public.et_access_type_id_seq'::regclass) NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_access_type OWNER TO myuser;

--
-- TOC entry 218 (class 1259 OID 16527)
-- Name: et_booking_status_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_booking_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_booking_status_id_seq OWNER TO myuser;

--
-- TOC entry 219 (class 1259 OID 16529)
-- Name: et_booking_status; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_booking_status (
    id integer DEFAULT nextval('public.et_booking_status_id_seq'::regclass) NOT NULL,
    status character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_booking_status OWNER TO myuser;

--
-- TOC entry 220 (class 1259 OID 16537)
-- Name: et_member_status_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_member_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_member_status_id_seq OWNER TO myuser;

--
-- TOC entry 221 (class 1259 OID 16539)
-- Name: et_member_status; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_member_status (
    id integer DEFAULT nextval('public.et_member_status_id_seq'::regclass) NOT NULL,
    status character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_member_status OWNER TO myuser;

--
-- TOC entry 222 (class 1259 OID 16547)
-- Name: et_resource_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_resource_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_resource_id_seq OWNER TO myuser;

--
-- TOC entry 223 (class 1259 OID 16549)
-- Name: et_resource; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_resource (
    id integer DEFAULT nextval('public.et_resource_id_seq'::regclass) NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_resource OWNER TO myuser;

--
-- TOC entry 224 (class 1259 OID 16557)
-- Name: et_seasons_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_seasons_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_seasons_id_seq OWNER TO myuser;

--
-- TOC entry 225 (class 1259 OID 16559)
-- Name: et_seasons; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_seasons (
    id integer DEFAULT nextval('public.et_seasons_id_seq'::regclass) NOT NULL,
    season character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    start_day integer,
    length integer
);


ALTER TABLE public.et_seasons OWNER TO myuser;

--
-- TOC entry 226 (class 1259 OID 16567)
-- Name: et_trip_difficulty_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_trip_difficulty_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_trip_difficulty_id_seq OWNER TO myuser;

--
-- TOC entry 227 (class 1259 OID 16569)
-- Name: et_trip_difficulty; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_trip_difficulty (
    id integer DEFAULT nextval('public.et_trip_difficulty_id_seq'::regclass) NOT NULL,
    level character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    level_short character varying(3),
    description character varying(255)
);


ALTER TABLE public.et_trip_difficulty OWNER TO myuser;

--
-- TOC entry 228 (class 1259 OID 16577)
-- Name: et_trip_status_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_trip_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_trip_status_id_seq OWNER TO myuser;

--
-- TOC entry 229 (class 1259 OID 16579)
-- Name: et_trip_status; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_trip_status (
    id integer DEFAULT nextval('public.et_trip_status_id_seq'::regclass) NOT NULL,
    status character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_trip_status OWNER TO myuser;

--
-- TOC entry 230 (class 1259 OID 16587)
-- Name: et_trip_types_trip_type_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_trip_types_trip_type_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_trip_types_trip_type_id_seq OWNER TO myuser;

--
-- TOC entry 231 (class 1259 OID 16589)
-- Name: et_trip_type; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_trip_type (
    id integer DEFAULT nextval('public.et_trip_types_trip_type_id_seq'::regclass) NOT NULL,
    type character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_trip_type OWNER TO myuser;

--
-- TOC entry 232 (class 1259 OID 16597)
-- Name: et_user_account_status_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_user_account_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_user_account_status_id_seq OWNER TO myuser;

--
-- TOC entry 233 (class 1259 OID 16599)
-- Name: et_user_account_status; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_user_account_status (
    id integer DEFAULT nextval('public.et_user_account_status_id_seq'::regclass) NOT NULL,
    status character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    description character varying(255)
);


ALTER TABLE public.et_user_account_status OWNER TO myuser;

--
-- TOC entry 234 (class 1259 OID 16610)
-- Name: et_user_age_group_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.et_user_age_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.et_user_age_group_id_seq OWNER TO myuser;

--
-- TOC entry 235 (class 1259 OID 16612)
-- Name: et_user_age_groups; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.et_user_age_groups (
    id integer DEFAULT nextval('public.et_user_age_group_id_seq'::regclass) NOT NULL,
    age_group character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.et_user_age_groups OWNER TO myuser;

--
-- TOC entry 236 (class 1259 OID 16620)
-- Name: st_group_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.st_group_id_seq OWNER TO myuser;

--
-- TOC entry 237 (class 1259 OID 16622)
-- Name: st_group; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_group (
    id integer DEFAULT nextval('public.st_group_id_seq'::regclass) NOT NULL,
    name character varying(45),
    description character varying(45),
    admin_flag boolean DEFAULT false,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.st_group OWNER TO myuser;

--
-- TOC entry 238 (class 1259 OID 16631)
-- Name: st_group_resource_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_group_resource_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.st_group_resource_id_seq OWNER TO myuser;

--
-- TOC entry 239 (class 1259 OID 16633)
-- Name: st_group_resource; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_group_resource (
    id integer DEFAULT nextval('public.st_group_resource_id_seq'::regclass) NOT NULL,
    group_id integer NOT NULL,
    resource_id integer NOT NULL,
    access_level_id integer NOT NULL,
    access_type_id integer NOT NULL,
    admin_flag boolean DEFAULT false,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.st_group_resource OWNER TO myuser;

--
-- TOC entry 240 (class 1259 OID 16642)
-- Name: st_token_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.st_token_id_seq OWNER TO myuser;

--
-- TOC entry 241 (class 1259 OID 16644)
-- Name: st_token; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_token (
    id integer DEFAULT nextval('public.st_token_id_seq'::regclass) NOT NULL,
    user_id integer NOT NULL,
    name character varying(45) NOT NULL,
    host character varying(45),
    token character varying(45),
    valid_from timestamp with time zone,
    valid_to timestamp with time zone,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    token_valid boolean DEFAULT false NOT NULL
);


ALTER TABLE public.st_token OWNER TO myuser;

--
-- TOC entry 242 (class 1259 OID 16653)
-- Name: st_user_group_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_user_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.st_user_group_id_seq OWNER TO myuser;

--
-- TOC entry 243 (class 1259 OID 16655)
-- Name: st_user_group; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_user_group (
    id integer DEFAULT nextval('public.st_user_group_id_seq'::regclass) NOT NULL,
    user_id integer NOT NULL,
    group_id integer NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.st_user_group OWNER TO myuser;

--
-- TOC entry 244 (class 1259 OID 16663)
-- Name: st_users_id_seq; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    MAXVALUE 2147483647
    CACHE 1;


ALTER SEQUENCE public.st_users_id_seq OWNER TO myuser;

--
-- TOC entry 245 (class 1259 OID 16665)
-- Name: st_users; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_users (
    id integer DEFAULT nextval('public.st_users_id_seq'::regclass) NOT NULL,
    name character varying(255) NOT NULL,
    username character varying(255) NOT NULL,
    email character varying(255) NOT NULL,
    member_status_id integer DEFAULT 0 NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    user_birth_date date,
    user_age_group_id integer,
    salt bytea,
    verifier bytea,
    user_address character varying(255),
    member_code character varying(20),
    user_password character varying(45),
    user_account_status_id integer DEFAULT 0 NOT NULL,
    user_account_hidden boolean,
    webauthn_user_id bytea,
    provider character varying(30),
    provider_id character varying(255)
);


ALTER TABLE public.st_users OWNER TO myuser;

--
-- TOC entry 248 (class 1259 OID 16787)
-- Name: st_webauthn_credentials; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_webauthn_credentials (
    id integer NOT NULL,
    user_id integer NOT NULL,
    credential_data jsonb NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    credential_id bytea,
    device_name character varying(45),
    device_metadata jsonb,
    last_used timestamp without time zone
);


ALTER TABLE public.st_webauthn_credentials OWNER TO myuser;

--
-- TOC entry 247 (class 1259 OID 16785)
-- Name: st_webauthn_credentials_id_seq1; Type: SEQUENCE; Schema: public; Owner: myuser
--

CREATE SEQUENCE public.st_webauthn_credentials_id_seq1
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.st_webauthn_credentials_id_seq1 OWNER TO myuser;

--
-- TOC entry 3332 (class 0 OID 0)
-- Dependencies: 247
-- Name: st_webauthn_credentials_id_seq1; Type: SEQUENCE OWNED BY; Schema: public; Owner: myuser
--

ALTER SEQUENCE public.st_webauthn_credentials_id_seq1 OWNED BY public.st_webauthn_credentials.id;


--
-- TOC entry 246 (class 1259 OID 16703)
-- Name: vt_trips; Type: VIEW; Schema: public; Owner: myuser
--

CREATE VIEW public.vt_trips AS
 SELECT at_trips.id,
    at_trips.owner_id,
    at_trips.trip_name,
    at_trips.location,
    at_trips.from_date,
    at_trips.to_date,
    at_trips.max_participants,
    at_trips.created,
    at_trips.modified,
    at_trips.trip_status_id,
    at_trips.trip_type_id,
    at_trips.difficulty_level_id
   FROM public.at_trips;


ALTER VIEW public.vt_trips OWNER TO myuser;

--
-- TOC entry 3112 (class 2604 OID 16790)
-- Name: st_webauthn_credentials id; Type: DEFAULT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_webauthn_credentials ALTER COLUMN id SET DEFAULT nextval('public.st_webauthn_credentials_id_seq1'::regclass);


--
-- TOC entry 3116 (class 2606 OID 16432)
-- Name: at_booking_people at_booking_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT at_booking_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3118 (class 2606 OID 16448)
-- Name: at_bookings at_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT at_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3120 (class 2606 OID 16459)
-- Name: at_group_bookings at_group_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_group_bookings
    ADD CONSTRAINT at_group_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3122 (class 2606 OID 16469)
-- Name: at_trip_cost_groups at_trip_cost_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_cost_groups
    ADD CONSTRAINT at_trip_cost_groups_pkey PRIMARY KEY (id);


--
-- TOC entry 3124 (class 2606 OID 16479)
-- Name: at_trip_costs at_trip_costs_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_costs
    ADD CONSTRAINT at_trip_costs_pkey PRIMARY KEY (id);


--
-- TOC entry 3126 (class 2606 OID 16496)
-- Name: at_trips at_trips_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trips
    ADD CONSTRAINT at_trips_pkey PRIMARY KEY (id);


--
-- TOC entry 3128 (class 2606 OID 16506)
-- Name: at_user_payments at_user_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_user_payments
    ADD CONSTRAINT at_user_payments_pkey PRIMARY KEY (id);


--
-- TOC entry 3130 (class 2606 OID 16516)
-- Name: et_access_level et_access_level_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_level
    ADD CONSTRAINT et_access_level_pkey PRIMARY KEY (id);


--
-- TOC entry 3132 (class 2606 OID 16526)
-- Name: et_access_type et_access_type_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_type
    ADD CONSTRAINT et_access_type_pkey PRIMARY KEY (id);


--
-- TOC entry 3134 (class 2606 OID 16536)
-- Name: et_booking_status et_booking_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_booking_status
    ADD CONSTRAINT et_booking_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3136 (class 2606 OID 16546)
-- Name: et_member_status et_member_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_member_status
    ADD CONSTRAINT et_member_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3138 (class 2606 OID 16556)
-- Name: et_resource et_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_resource
    ADD CONSTRAINT et_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3140 (class 2606 OID 16566)
-- Name: et_seasons et_seasons_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_seasons
    ADD CONSTRAINT et_seasons_pkey PRIMARY KEY (id);


--
-- TOC entry 3142 (class 2606 OID 16576)
-- Name: et_trip_difficulty et_trip_difficulty_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_difficulty
    ADD CONSTRAINT et_trip_difficulty_pkey PRIMARY KEY (id);


--
-- TOC entry 3144 (class 2606 OID 16586)
-- Name: et_trip_status et_trip_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_status
    ADD CONSTRAINT et_trip_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3146 (class 2606 OID 16596)
-- Name: et_trip_type et_trip_types_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_type
    ADD CONSTRAINT et_trip_types_pkey PRIMARY KEY (id);


--
-- TOC entry 3148 (class 2606 OID 16609)
-- Name: et_user_account_status et_user_account_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_account_status
    ADD CONSTRAINT et_user_account_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3150 (class 2606 OID 16619)
-- Name: et_user_age_groups et_user_age_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_age_groups
    ADD CONSTRAINT et_user_age_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3152 (class 2606 OID 16630)
-- Name: st_group st_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group
    ADD CONSTRAINT st_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3154 (class 2606 OID 16641)
-- Name: st_group_resource st_group_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group_resource
    ADD CONSTRAINT st_group_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3156 (class 2606 OID 16652)
-- Name: st_token st_token_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_token
    ADD CONSTRAINT st_token_pkey PRIMARY KEY (id);


--
-- TOC entry 3158 (class 2606 OID 16662)
-- Name: st_user_group st_user_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_user_group
    ADD CONSTRAINT st_user_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3160 (class 2606 OID 16679)
-- Name: st_users st_users_email_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_email_key UNIQUE (email);


--
-- TOC entry 3162 (class 2606 OID 16677)
-- Name: st_users st_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3164 (class 2606 OID 16681)
-- Name: st_users st_users_username_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_username_key UNIQUE (username);


--
-- TOC entry 3166 (class 2606 OID 16797)
-- Name: st_webauthn_credentials st_webauthn_credentials_pkey1; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_webauthn_credentials
    ADD CONSTRAINT st_webauthn_credentials_pkey1 PRIMARY KEY (id);


--
-- TOC entry 3170 (class 2620 OID 16772)
-- Name: at_booking_people set_modified_timestamp_at_booking_people; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_booking_people BEFORE UPDATE ON public.at_booking_people FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3171 (class 2620 OID 16771)
-- Name: at_bookings set_modified_timestamp_at_bookings; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_bookings BEFORE UPDATE ON public.at_bookings FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3172 (class 2620 OID 16752)
-- Name: at_group_bookings set_modified_timestamp_at_group_bookings; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_group_bookings BEFORE UPDATE ON public.at_group_bookings FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3173 (class 2620 OID 16753)
-- Name: at_trip_cost_groups set_modified_timestamp_at_trip_cost_groups; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trip_cost_groups BEFORE UPDATE ON public.at_trip_cost_groups FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3174 (class 2620 OID 16754)
-- Name: at_trip_costs set_modified_timestamp_at_trip_costs; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trip_costs BEFORE UPDATE ON public.at_trip_costs FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3175 (class 2620 OID 16755)
-- Name: at_trips set_modified_timestamp_at_trips; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trips BEFORE UPDATE ON public.at_trips FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3176 (class 2620 OID 16756)
-- Name: at_user_payments set_modified_timestamp_at_user_payments; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_user_payments BEFORE UPDATE ON public.at_user_payments FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3177 (class 2620 OID 16757)
-- Name: et_access_level set_modified_timestamp_et_access_level; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_access_level BEFORE UPDATE ON public.et_access_level FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3178 (class 2620 OID 16758)
-- Name: et_access_type set_modified_timestamp_et_access_type; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_access_type BEFORE UPDATE ON public.et_access_type FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3179 (class 2620 OID 16774)
-- Name: et_booking_status set_modified_timestamp_et_booking_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_booking_status BEFORE UPDATE ON public.et_booking_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3180 (class 2620 OID 16759)
-- Name: et_member_status set_modified_timestamp_et_member_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_member_status BEFORE UPDATE ON public.et_member_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3181 (class 2620 OID 16760)
-- Name: et_resource set_modified_timestamp_et_resource; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_resource BEFORE UPDATE ON public.et_resource FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3182 (class 2620 OID 16761)
-- Name: et_seasons set_modified_timestamp_et_seasons; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_seasons BEFORE UPDATE ON public.et_seasons FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3183 (class 2620 OID 16762)
-- Name: et_trip_difficulty set_modified_timestamp_et_trip_difficulty; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_difficulty BEFORE UPDATE ON public.et_trip_difficulty FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3184 (class 2620 OID 16763)
-- Name: et_trip_status set_modified_timestamp_et_trip_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_status BEFORE UPDATE ON public.et_trip_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3185 (class 2620 OID 16764)
-- Name: et_trip_type set_modified_timestamp_et_trip_type; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_type BEFORE UPDATE ON public.et_trip_type FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3186 (class 2620 OID 16765)
-- Name: et_user_account_status set_modified_timestamp_et_user_account_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_user_account_status BEFORE UPDATE ON public.et_user_account_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3187 (class 2620 OID 16766)
-- Name: et_user_age_groups set_modified_timestamp_et_user_age_groups; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_user_age_groups BEFORE UPDATE ON public.et_user_age_groups FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3188 (class 2620 OID 16767)
-- Name: st_group set_modified_timestamp_st_group; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_group BEFORE UPDATE ON public.st_group FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3189 (class 2620 OID 16768)
-- Name: st_group_resource set_modified_timestamp_st_group_resource; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_group_resource BEFORE UPDATE ON public.st_group_resource FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3190 (class 2620 OID 16769)
-- Name: st_token set_modified_timestamp_st_token; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_token BEFORE UPDATE ON public.st_token FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3191 (class 2620 OID 16770)
-- Name: st_user_group set_modified_timestamp_st_user_group; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_user_group BEFORE UPDATE ON public.st_user_group FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3192 (class 2620 OID 16773)
-- Name: st_users set_modified_timestamp_st_users; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_users BEFORE UPDATE ON public.st_users FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3167 (class 2606 OID 16688)
-- Name: at_booking_people booking_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT booking_id_fkey FOREIGN KEY (booking_id) REFERENCES public.at_bookings(id) NOT VALID;


--
-- TOC entry 3169 (class 2606 OID 16698)
-- Name: at_bookings bookings_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT bookings_status_id_fkey FOREIGN KEY (booking_status_id) REFERENCES public.et_booking_status(id) NOT VALID;


--
-- TOC entry 3168 (class 2606 OID 16693)
-- Name: at_booking_people user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (person_id) REFERENCES public.st_users(id) NOT VALID;


--
-- TOC entry 3331 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: myuser
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2026-01-27 11:20:54

--
-- PostgreSQL database dump complete
--

