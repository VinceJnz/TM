--
-- PostgreSQL database dump
--

\restrict lpwpapLssGMobU8OiciWKYaRc5xaaQXvshTEjZIQN8qg54UBPGUoKqdx4rODbD3

-- Dumped from database version 13.23 (Debian 13.23-1.pgdg13+1)
-- Dumped by pg_dump version 18.1

-- Started on 2026-02-28 22:44:06

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

DROP DATABASE IF EXISTS mydatabase;
--
-- TOC entry 3373 (class 1262 OID 16384)
-- Name: mydatabase; Type: DATABASE; Schema: -; Owner: myuser
--

CREATE DATABASE mydatabase WITH TEMPLATE = template0 ENCODING = 'UTF8' LOCALE_PROVIDER = libc LOCALE = 'en_US.utf8';


ALTER DATABASE mydatabase OWNER TO myuser;

\unrestrict lpwpapLssGMobU8OiciWKYaRc5xaaQXvshTEjZIQN8qg54UBPGUoKqdx4rODbD3
\connect mydatabase
\restrict lpwpapLssGMobU8OiciWKYaRc5xaaQXvshTEjZIQN8qg54UBPGUoKqdx4rODbD3

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
-- TOC entry 3374 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: myuser
--

COMMENT ON SCHEMA public IS 'standard public schema';


--
-- TOC entry 778 (class 1247 OID 16833)
-- Name: app_role; Type: TYPE; Schema: public; Owner: myuser
--

CREATE TYPE public.app_role AS ENUM (
    'user',
    'admin',
    'sysadmin'
);


ALTER TYPE public.app_role OWNER TO myuser;

--
-- TOC entry 250 (class 1255 OID 16726)
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
-- TOC entry 248 (class 1255 OID 16709)
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
-- TOC entry 249 (class 1255 OID 16710)
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
    stripe_session_id character varying(100)
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
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    role public.app_role
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
    token_valid boolean DEFAULT false NOT NULL,
    session_data json
);


ALTER TABLE public.st_token OWNER TO myuser;

--
-- TOC entry 247 (class 1259 OID 16821)
-- Name: st_user_credentials; Type: TABLE; Schema: public; Owner: myuser
--

CREATE TABLE public.st_user_credentials (
    id integer NOT NULL,
    user_id integer NOT NULL,
    credential_id bytea,
    last_used timestamp without time zone,
    credential_data jsonb NOT NULL,
    device_name character varying(45),
    device_metadata jsonb,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.st_user_credentials OWNER TO myuser;

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
    user_address character varying(255),
    member_code character varying(20),
    user_password character varying(100),
    user_account_status_id integer DEFAULT 0 NOT NULL,
    user_account_hidden boolean,
    provider character varying(30),
    provider_id character varying(255)
);


ALTER TABLE public.st_users OWNER TO myuser;

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
-- TOC entry 3322 (class 0 OID 16419)
-- Dependencies: 201
-- Data for Name: at_booking_people; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_booking_people VALUES (24, 17, 34, 17, 'nnn', '2024-12-26 06:13:05.649785', '2024-12-26 06:13:05.649785');
INSERT INTO public.at_booking_people VALUES (10, 16, 10, 2, '123lkjo69696', '2024-10-06 10:10:55.298416', '2024-10-06 10:10:55.298416');
INSERT INTO public.at_booking_people VALUES (12, 16, 10, 3, 'XXXYY', '2024-10-07 01:36:09.828853', '2024-10-07 01:36:09.828853');
INSERT INTO public.at_booking_people VALUES (16, 16, 7, 2, 'yyyyyy', '2024-10-11 06:18:35.343878', '2024-10-11 06:18:35.343878');
INSERT INTO public.at_booking_people VALUES (21, 16, 7, 3, 'hhhhh', '2024-10-11 22:19:35.215926', '2024-10-11 22:19:35.215926');
INSERT INTO public.at_booking_people VALUES (22, 16, 13, 1, 'nnnnn', '2024-10-11 23:52:30.138021', '2024-10-11 23:52:30.138021');
INSERT INTO public.at_booking_people VALUES (4, 16, 7, 1, 'A first booking', '2024-10-04 09:08:20.931727', '2024-10-04 09:08:20.931727');
INSERT INTO public.at_booking_people VALUES (23, 17, 34, 16, 'qqqqdddd', '2024-12-20 09:14:36.347532', '2024-12-20 09:14:36.347532');
INSERT INTO public.at_booking_people VALUES (26, 17, 36, 16, 'vvv', '2025-03-04 07:43:09.762168', '2025-03-04 07:43:09.762168');
INSERT INTO public.at_booking_people VALUES (27, 17, 36, 17, 'vbbb', '2025-03-04 07:43:27.032349', '2025-03-04 07:43:27.032349');


--
-- TOC entry 3324 (class 0 OID 16435)
-- Dependencies: 203
-- Data for Name: at_bookings; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_bookings VALUES (18, 16, 'Vince''s booking', '0001-01-01 00:00:00', '0001-01-01 00:00:00', 1, '2024-10-16 02:51:04.680147', '2024-12-20 09:08:11.157029', 2, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (14, 16, 'a random booking', '2024-11-02 00:00:00', '2024-11-01 00:00:00', 1, '2024-10-13 22:44:38.713194', '2024-12-20 09:08:11.157029', 1, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (27, 16, 'wow more bookings', '2024-12-01 00:00:00', '2024-12-05 00:00:00', 1, '2024-10-18 00:12:21.727307', '2024-12-20 09:08:11.157029', 2, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (13, 16, 'another one', '2024-11-01 00:00:00', '2024-12-01 00:00:00', 3, '2024-10-11 23:46:36.105431', '2024-12-20 09:08:11.157029', 1, NULL, NULL, 200.10, '2024-10-28', NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (7, 16, 'A first booking', '2024-11-03 00:00:00', '2024-11-07 00:00:00', 1, '2024-09-29 10:05:59.254752', '2024-12-20 09:08:11.157029', 1, NULL, NULL, 213.45, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (34, 17, 'kkkeeeYZZZZZ', '2024-11-01 00:00:00', '2024-12-01 00:00:00', 1, '2024-12-20 09:10:07.684579', '2025-01-01 08:23:16.960475', 1, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (36, 17, 'hhhhhhhhhhhhh', '2024-12-01 00:00:00', '2024-12-11 00:00:00', 1, '2025-03-04 07:00:43.604301', '2025-03-04 07:00:43.604301', 2, NULL, NULL, NULL, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (37, 16, 'Vince2 Booking1', '2025-05-19 00:00:00', '2025-05-23 00:00:00', 1, '2025-03-04 08:35:44.46041', '2025-03-04 08:35:44.46041', 7, NULL, NULL, 0.00, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (10, 16, 'Booking 2', '2024-10-06 00:00:00', '2024-10-20 00:00:00', 1, '2024-10-06 09:14:26.265978', '2026-01-28 10:34:55.21241', 2, NULL, NULL, 364.00, '2026-01-28', 36400, NULL, 'cs_test_a1eL63TdEAA6dR650t6d4ap0npXlXOiZCAk6V50IHgIluo39Ab3qneoiCR');


--
-- TOC entry 3326 (class 0 OID 16451)
-- Dependencies: 205
-- Data for Name: at_group_bookings; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_group_bookings VALUES (1, 'Vince''s group', 0, '2024-10-29 23:01:14.192148', '2024-10-29 23:01:14.192148');


--
-- TOC entry 3328 (class 0 OID 16462)
-- Dependencies: 207
-- Data for Name: at_trip_cost_groups; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_trip_cost_groups VALUES (1, 'Lodge winter rate', '2024-10-26 06:56:50.644474', '2024-10-26 06:56:50.644474');


--
-- TOC entry 3330 (class 0 OID 16472)
-- Dependencies: 209
-- Data for Name: at_trip_costs; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_trip_costs VALUES (4, 1, NULL, 1, 2, 2, 42.00, '2024-10-28 06:36:22.313324', '2024-10-28 06:36:22.313324');
INSERT INTO public.at_trip_costs VALUES (5, 1, NULL, 2, 2, 2, 63.00, '2024-10-28 06:46:00.905423', '2024-10-28 06:46:00.905423');
INSERT INTO public.at_trip_costs VALUES (2, 1, NULL, 1, 3, 2, 11.00, '2024-10-26 07:17:22.832664', '2024-10-26 07:17:22.832664');
INSERT INTO public.at_trip_costs VALUES (3, 1, NULL, 1, 4, 2, 21.00, '2024-10-26 07:19:42.892956', '2024-10-26 07:19:42.892956');
INSERT INTO public.at_trip_costs VALUES (6, 1, NULL, 1, 1, 2, 5.00, '2025-01-14 07:07:35.112759', '2025-01-14 07:07:35.112759');
INSERT INTO public.at_trip_costs VALUES (7, 1, NULL, 2, 1, 2, 6.00, '2025-01-14 07:07:53.892965', '2025-01-14 07:07:53.892965');
INSERT INTO public.at_trip_costs VALUES (8, 1, NULL, 2, 3, 2, 13.00, '2025-01-14 07:08:06.32252', '2025-01-14 07:08:06.32252');
INSERT INTO public.at_trip_costs VALUES (9, 1, NULL, 2, 4, 2, 25.00, '2025-01-14 07:08:21.262404', '2025-01-14 07:08:21.262404');


--
-- TOC entry 3332 (class 0 OID 16482)
-- Dependencies: 211
-- Data for Name: at_trips; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_trips VALUES (2, 16, 'A new trip', NULL, '2024-12-01', '2024-12-11', 3, '2024-10-11 21:38:01.098156', '2024-10-11 21:38:01.098156', 1, 1, 2, 1, NULL);
INSERT INTO public.at_trips VALUES (1, 16, 'Trip1a', NULL, '2024-11-01', '2024-12-01', 3, '2024-10-10 10:37:23.829893', '2024-10-10 10:37:23.829893', 3, 4, 3, 1, NULL);
INSERT INTO public.at_trips VALUES (6, 16, 'A fantastic trip a!', NULL, '2024-10-17', '2024-10-17', 11, '2024-10-17 23:57:28.0255', '2024-10-17 23:57:28.0255', 1, 1, 1, 1, NULL);
INSERT INTO public.at_trips VALUES (7, 16, '3rd trip', NULL, '2025-05-19', '2025-05-23', 1, '2024-10-18 00:11:25.195676', '2024-10-18 00:11:25.195676', 1, 2, 1, 1, NULL);


--
-- TOC entry 3334 (class 0 OID 16499)
-- Dependencies: 213
-- Data for Name: at_user_payments; Type: TABLE DATA; Schema: public; Owner: myuser
--



--
-- TOC entry 3336 (class 0 OID 16509)
-- Dependencies: 215
-- Data for Name: et_access_level; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_access_level VALUES (1, 'none', NULL, '2024-11-24 23:23:04.959786', '2024-11-24 23:23:04.959786');
INSERT INTO public.et_access_level VALUES (2, 'get', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (3, 'put', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (4, 'post', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (5, 'delete', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');


--
-- TOC entry 3338 (class 0 OID 16519)
-- Dependencies: 217
-- Data for Name: et_access_type; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_access_type VALUES (2, 'owner', NULL, '2024-11-24 23:15:13.207337', '2024-11-24 23:15:13.207337');
INSERT INTO public.et_access_type VALUES (1, 'admin', NULL, '2024-11-24 23:14:57.462385', '2024-11-24 23:14:57.462385');


--
-- TOC entry 3340 (class 0 OID 16529)
-- Dependencies: 219
-- Data for Name: et_booking_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_booking_status VALUES (1, 'New', '2024-09-29 10:11:30.906822', '2024-09-29 10:11:30.906822');
INSERT INTO public.et_booking_status VALUES (2, 'Cancelled', '2024-09-30 08:29:27.323855', '2024-09-30 08:29:27.323855');
INSERT INTO public.et_booking_status VALUES (3, 'Paid', '2024-09-30 08:29:41.411134', '2024-09-30 08:29:41.411134');


--
-- TOC entry 3342 (class 0 OID 16539)
-- Dependencies: 221
-- Data for Name: et_member_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_member_status VALUES (1, 'Yes', '2025-01-12 08:36:38.416883', '2025-01-12 08:36:38.416883');
INSERT INTO public.et_member_status VALUES (2, 'No', '2025-01-12 08:36:12.929545', '2025-01-12 08:36:12.929545');


--
-- TOC entry 3344 (class 0 OID 16549)
-- Dependencies: 223
-- Data for Name: et_resource; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_resource VALUES (1, 'users', NULL, '2024-11-24 23:20:21.603342', '2024-11-24 23:20:21.603342');
INSERT INTO public.et_resource VALUES (2, 'trips', NULL, '2024-11-24 23:20:40.311542', '2024-11-24 23:20:40.311542');
INSERT INTO public.et_resource VALUES (3, 'bookings', NULL, '2024-11-24 23:20:48.161546', '2024-11-24 23:20:48.161546');
INSERT INTO public.et_resource VALUES (8, 'auth', 'auth url', '2024-12-05 07:13:21.44181', '2024-12-05 07:13:21.44181');
INSERT INTO public.et_resource VALUES (9, 'tripStatus', 'trip Status', '2024-12-05 07:51:55.13568', '2024-12-05 07:51:55.13568');
INSERT INTO public.et_resource VALUES (10, 'tripDifficulty', 'trip Difficulty', '2024-12-05 07:52:04.899949', '2024-12-05 07:52:04.899949');
INSERT INTO public.et_resource VALUES (11, 'bookingStatus', 'Booking Status', '2024-12-13 07:18:33.355895', '2024-12-13 07:18:33.355895');
INSERT INTO public.et_resource VALUES (13, 'userAgeGroups', 'User Age Groups', '2024-12-13 07:30:39.813523', '2024-12-13 07:30:39.813523');
INSERT INTO public.et_resource VALUES (14, 'userAccountStatus', 'User Account Status', '2024-12-13 07:30:58.923303', '2024-12-13 07:30:58.923303');
INSERT INTO public.et_resource VALUES (16, 'tripsReport', 'trips Report (Participant Status)', '2024-12-14 05:37:54.996572', '2024-12-14 05:37:54.996572');
INSERT INTO public.et_resource VALUES (17, 'bookingPeople', 'Booking People', '2024-12-20 09:12:42.840489', '2024-12-20 09:12:42.840489');
INSERT INTO public.et_resource VALUES (15, 'userMemberStatus', 'User Member Status', '2024-12-13 07:31:12.61181', '2024-12-13 07:31:12.61181');
INSERT INTO public.et_resource VALUES (5, 'myBookings', 'My Bookings', '2025-02-27 06:08:15.40331', '2025-02-27 06:08:15.40331');
INSERT INTO public.et_resource VALUES (19, 'webauthn', 'WebAuthn device management', '2025-09-30 08:08:17.550883', '2025-09-30 08:08:17.550883');
INSERT INTO public.et_resource VALUES (20, 'set-username', 'For setting the user name', '2025-12-26 03:35:15.628581', '2025-12-26 03:36:13.263913');


--
-- TOC entry 3346 (class 0 OID 16559)
-- Dependencies: 225
-- Data for Name: et_seasons; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_seasons VALUES (2, 'Winter', '2024-10-26 05:28:39.461983', '2024-10-26 05:28:39.461983', NULL, NULL);
INSERT INTO public.et_seasons VALUES (1, 'Summer', '2024-10-26 05:28:32.741256', '2024-10-26 05:28:32.741256', NULL, NULL);


--
-- TOC entry 3348 (class 0 OID 16569)
-- Dependencies: 227
-- Data for Name: et_trip_difficulty; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_trip_difficulty VALUES (1, 'Medium to Fit', '2024-10-26 06:03:17.094307', '2024-10-26 06:03:17.094307', 'MF', 'Up to 8 hours per day, pace faster than M, off track and above bush line travel to be expected.');
INSERT INTO public.et_trip_difficulty VALUES (2, 'Easy', '2024-10-26 06:03:53.30417', '2024-10-26 06:03:53.30417', 'E', 'Up to 4 hours per day, pace slower than EM.');
INSERT INTO public.et_trip_difficulty VALUES (3, 'Slow medium', '2024-10-28 06:35:23.182334', '2024-10-28 06:35:23.182334', 'SM', 'Medium trip at a slower pace than the standard pace');


--
-- TOC entry 3350 (class 0 OID 16579)
-- Dependencies: 229
-- Data for Name: et_trip_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_trip_status VALUES (1, 'New', '2024-10-10 10:34:38.851101', '2024-10-10 10:34:38.851101');
INSERT INTO public.et_trip_status VALUES (2, 'Cancelled', '2024-10-10 10:34:49.204346', '2024-10-10 10:34:49.204346');
INSERT INTO public.et_trip_status VALUES (3, 'Completed', '2024-10-10 10:34:56.149138', '2024-10-10 10:34:56.149138');


--
-- TOC entry 3352 (class 0 OID 16589)
-- Dependencies: 231
-- Data for Name: et_trip_type; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_trip_type VALUES (1, 'Hiking', '2024-10-26 05:39:39.691348', '2024-10-26 05:39:39.691348');
INSERT INTO public.et_trip_type VALUES (2, 'Skiing', '2024-10-26 05:39:45.72734', '2024-10-26 05:39:45.72734');
INSERT INTO public.et_trip_type VALUES (3, 'Cycling', '2024-10-26 05:39:52.463626', '2024-10-26 05:39:52.463626');
INSERT INTO public.et_trip_type VALUES (4, 'Camping', '2024-10-26 05:40:03.152673', '2024-10-26 05:40:03.152673');
INSERT INTO public.et_trip_type VALUES (5, 'Rafting', '2024-10-26 05:40:08.886972', '2024-10-26 05:40:08.886972');
INSERT INTO public.et_trip_type VALUES (6, 'Climbing', '2024-10-26 05:40:15.566278', '2024-10-26 05:40:15.566278');


--
-- TOC entry 3354 (class 0 OID 16599)
-- Dependencies: 233
-- Data for Name: et_user_account_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_user_account_status VALUES (0, 'New', '2025-01-12 07:47:04.2678', '2025-01-12 07:47:04.2678', 'A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin.');
INSERT INTO public.et_user_account_status VALUES (1, 'Verified', '2024-12-01 09:36:35.177495', '2024-12-01 09:36:35.177495', 'The email address has been verified. An Admin now needs to activate the account.');
INSERT INTO public.et_user_account_status VALUES (2, 'Active', '2024-12-01 09:36:50.00299', '2024-12-01 09:36:50.00299', 'An account that has been activated, and is currently active.');
INSERT INTO public.et_user_account_status VALUES (3, 'Disabled', '2025-01-12 08:28:36.131138', '2025-01-12 08:28:36.131138', 'An account that has been disabled.');
INSERT INTO public.et_user_account_status VALUES (4, 'Reset', '2025-01-12 08:29:16.932961', '2025-01-12 08:29:16.932961', 'The account is flagged for a password reset. The user will be informed at the next login.');
INSERT INTO public.et_user_account_status VALUES (5, 'WebAuthnReset', '2025-08-16 05:08:58.699501', '2025-08-16 05:09:29.400568', 'The account is flagged for a webAuthn reset');


--
-- TOC entry 3356 (class 0 OID 16612)
-- Dependencies: 235
-- Data for Name: et_user_age_groups; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_user_age_groups VALUES (1, 'Infant', '2024-10-26 07:08:21.941336', '2024-10-26 07:08:21.941336');
INSERT INTO public.et_user_age_groups VALUES (2, 'Adult', '2024-10-26 07:08:36.386124', '2024-10-26 07:08:36.386124');
INSERT INTO public.et_user_age_groups VALUES (3, 'Child', '2024-10-26 07:08:42.383391', '2024-10-26 07:08:42.383391');
INSERT INTO public.et_user_age_groups VALUES (4, 'Youth', '2024-10-26 07:09:03.922737', '2024-10-26 07:09:03.922737');


--
-- TOC entry 3358 (class 0 OID 16622)
-- Dependencies: 237
-- Data for Name: st_group; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_group VALUES (1, 'Sys Admin', 'System admin', '2024-11-24 23:31:26.62785', '2026-02-27 09:48:24.645486', 'sysadmin');
INSERT INTO public.st_group VALUES (3, 'App Admin', 'Application admin', '2024-12-05 23:46:11.129967', '2026-02-27 09:48:24.645486', 'admin');
INSERT INTO public.st_group VALUES (2, 'User', 'app users', '2024-12-05 06:57:30.344557', '2026-02-27 09:48:24.645486', 'user');


--
-- TOC entry 3360 (class 0 OID 16633)
-- Dependencies: 239
-- Data for Name: st_group_resource; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_group_resource VALUES (1, 1, 1, 2, 1, '2024-11-24 23:34:35.86487', '2024-11-24 23:34:35.86487');
INSERT INTO public.st_group_resource VALUES (3, 1, 1, 3, 1, '2024-11-24 23:35:01.83461', '2024-11-24 23:35:01.83461');
INSERT INTO public.st_group_resource VALUES (4, 1, 1, 4, 1, '2024-11-24 23:35:17.568932', '2024-11-24 23:35:17.568932');
INSERT INTO public.st_group_resource VALUES (5, 1, 1, 5, 1, '2024-11-24 23:35:39.544196', '2024-11-24 23:35:39.544196');
INSERT INTO public.st_group_resource VALUES (6, 1, 2, 2, 1, '2024-11-24 23:40:19.506411', '2024-11-24 23:40:19.506411');
INSERT INTO public.st_group_resource VALUES (7, 1, 2, 3, 1, '2024-11-24 23:42:30.994064', '2024-11-24 23:42:30.994064');
INSERT INTO public.st_group_resource VALUES (8, 2, 2, 2, 2, '2024-12-05 06:59:42.806108', '2024-12-05 06:59:42.806108');
INSERT INTO public.st_group_resource VALUES (12, 2, 8, 2, 2, '2024-12-05 07:13:57.298371', '2024-12-05 07:13:57.298371');
INSERT INTO public.st_group_resource VALUES (13, 2, 9, 2, 2, '2024-12-05 07:52:39.946898', '2024-12-05 07:52:39.946898');
INSERT INTO public.st_group_resource VALUES (18, 2, 3, 5, 2, '2024-12-05 07:56:52.363959', '2024-12-05 07:56:52.363959');
INSERT INTO public.st_group_resource VALUES (17, 2, 3, 4, 2, '2024-12-05 07:56:43.626614', '2024-12-05 07:56:43.626614');
INSERT INTO public.st_group_resource VALUES (16, 2, 3, 3, 2, '2024-12-05 07:56:32.357164', '2024-12-05 07:56:32.357164');
INSERT INTO public.st_group_resource VALUES (15, 2, 3, 2, 2, '2024-12-05 07:56:06.118582', '2024-12-05 07:56:06.118582');
INSERT INTO public.st_group_resource VALUES (14, 2, 10, 2, 2, '2024-12-05 07:52:49.566355', '2024-12-05 07:52:49.566355');
INSERT INTO public.st_group_resource VALUES (19, 2, 11, 2, 2, '2024-12-13 07:26:15.856341', '2024-12-13 07:26:15.856341');
INSERT INTO public.st_group_resource VALUES (20, 2, 13, 2, 2, '2024-12-13 07:31:44.260408', '2024-12-13 07:31:44.260408');
INSERT INTO public.st_group_resource VALUES (21, 2, 15, 2, 2, '2024-12-13 07:32:38.319088', '2024-12-13 07:32:38.319088');
INSERT INTO public.st_group_resource VALUES (22, 2, 14, 2, 2, '2024-12-13 07:32:54.10676', '2024-12-13 07:32:54.10676');
INSERT INTO public.st_group_resource VALUES (23, 2, 1, 2, 2, '2024-12-13 07:51:52.949741', '2024-12-13 07:51:52.949741');
INSERT INTO public.st_group_resource VALUES (24, 2, 16, 2, 2, '2024-12-14 05:38:15.734325', '2024-12-14 05:38:15.734325');
INSERT INTO public.st_group_resource VALUES (25, 2, 17, 2, 2, '2024-12-20 09:13:11.812551', '2024-12-20 09:13:11.812551');
INSERT INTO public.st_group_resource VALUES (26, 2, 17, 3, 2, '2024-12-20 09:13:24.920667', '2024-12-20 09:13:24.920667');
INSERT INTO public.st_group_resource VALUES (27, 2, 17, 4, 2, '2024-12-20 09:13:32.890183', '2024-12-20 09:13:32.890183');
INSERT INTO public.st_group_resource VALUES (28, 2, 17, 5, 2, '2024-12-20 09:13:42.950509', '2024-12-20 09:13:42.950509');
INSERT INTO public.st_group_resource VALUES (30, 2, 5, 2, 2, '2025-02-27 06:45:48.486863', '2025-02-27 06:45:48.486863');
INSERT INTO public.st_group_resource VALUES (31, 2, 5, 3, 2, '2025-02-27 06:46:06.882877', '2025-02-27 06:46:06.882877');
INSERT INTO public.st_group_resource VALUES (32, 2, 5, 4, 2, '2025-02-27 06:46:16.122971', '2025-02-27 06:46:16.122971');
INSERT INTO public.st_group_resource VALUES (33, 2, 5, 5, 2, '2025-02-27 06:46:32.787876', '2025-02-27 06:46:32.787876');
INSERT INTO public.st_group_resource VALUES (36, 2, 19, 2, 2, '2025-09-30 08:08:51.554737', '2025-09-30 08:08:51.554737');
INSERT INTO public.st_group_resource VALUES (37, 2, 19, 5, 2, '2025-09-30 08:09:09.626432', '2025-09-30 08:09:09.626432');
INSERT INTO public.st_group_resource VALUES (38, 2, 20, 3, 2, '2025-12-26 03:37:59.772919', '2025-12-26 03:37:59.772919');
INSERT INTO public.st_group_resource VALUES (39, 2, 20, 4, 2, '2025-12-26 03:38:23.187051', '2025-12-26 03:38:23.187051');
INSERT INTO public.st_group_resource VALUES (40, 2, 20, 2, 2, '2025-12-26 03:39:04.988133', '2025-12-26 03:39:04.988133');
INSERT INTO public.st_group_resource VALUES (41, 2, 1, 4, 2, '2025-12-26 04:38:24.188317', '2025-12-26 04:38:24.188317');


--
-- TOC entry 3362 (class 0 OID 16644)
-- Dependencies: 241
-- Data for Name: st_token; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_token VALUES (591, 16, 'session', 'localhost', 'H59f27FngFGW4SOOSTo08w-1ko6_v_tGxgu8YAX22ok', '2026-02-28 09:20:08.18886+00', '2026-02-28 10:20:08.18886+00', '2026-02-28 09:20:08.189643', '2026-02-28 09:20:08.189643', true, NULL);


--
-- TOC entry 3367 (class 0 OID 16821)
-- Dependencies: 247
-- Data for Name: st_user_credentials; Type: TABLE DATA; Schema: public; Owner: myuser
--



--
-- TOC entry 3364 (class 0 OID 16655)
-- Dependencies: 243
-- Data for Name: st_user_group; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_user_group VALUES (1, 16, 1, '2024-11-24 23:31:58.500845', '2024-11-24 23:31:58.500845');
INSERT INTO public.st_user_group VALUES (2, 17, 2, '2024-11-28 03:54:26.342778', '2024-11-28 03:54:26.342778');
INSERT INTO public.st_user_group VALUES (3, 3, 2, '2024-12-27 03:14:27.828175', '2024-12-27 03:14:27.828175');
INSERT INTO public.st_user_group VALUES (4, 1, 2, '2024-12-27 03:14:42.095962', '2024-12-27 03:14:42.095962');
INSERT INTO public.st_user_group VALUES (5, 2, 2, '2024-12-27 03:14:50.895662', '2024-12-27 03:14:50.895662');
INSERT INTO public.st_user_group VALUES (6, 15, 2, '2024-12-27 03:15:03.932908', '2024-12-27 03:15:03.932908');
INSERT INTO public.st_user_group VALUES (9, 20, 1, '2025-06-19 10:09:16.850648', '2025-06-19 10:09:16.850648');
INSERT INTO public.st_user_group VALUES (12, 35, 1, '2025-08-10 00:29:44.744291', '2025-08-10 00:29:44.744291');
INSERT INTO public.st_user_group VALUES (11, 36, 1, '2025-08-03 04:47:51.633783', '2025-08-13 22:26:44.387595');
INSERT INTO public.st_user_group VALUES (14, 37, 2, '2025-09-28 23:33:26.293864', '2025-09-28 23:33:26.293864');
INSERT INTO public.st_user_group VALUES (15, 39, 2, '2025-09-29 01:22:01.090636', '2025-09-29 01:22:01.090636');
INSERT INTO public.st_user_group VALUES (20, 76, 2, '2026-02-28 08:54:16.953424', '2026-02-28 08:54:16.953424');


--
-- TOC entry 3366 (class 0 OID 16665)
-- Dependencies: 245
-- Data for Name: st_users; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_users VALUES (2, 'Donna Jennings12', 'donna', 'dj0040@gmail.com', 1, '2024-10-04 09:08:52.345413', '2025-01-14 07:10:06.688522', NULL, 1, '93 Farnham Street', '1234B', NULL, 1, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (3, 'Dylan Jennings1', 'Dylan', 'dylan@dt.net.nz', 1, '2024-10-04 09:09:11.226469', '2025-01-14 07:11:07.063766', NULL, 4, '93 Farnham Street', '12347', NULL, 1, true, NULL, NULL);
INSERT INTO public.st_users VALUES (36, 'test1', 'test1', 'vince.jennings101@gmail.com', 0, '2025-08-13 22:25:08.509142', '2025-09-28 08:12:01.518222', NULL, 2, NULL, 'q54wggf', NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (76, 'Vince J Basic', 'VinceBasic', 'Vince.basic@gmail.com', 1, '2026-02-22 00:10:45.937897', '2026-02-28 08:54:47.569492', '2004-03-02', NULL, '93 F', NULL, '$2a$10$XMZLg.K.FgsaMM2zsuMFk.DxDHqjauPXamInaKDkinINjAbVSru9y', 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (15, 'Vince Jennings', 'vince1', 'vince.jennings@gmail.com', 1, '2024-11-14 00:12:08.297532', '2026-02-13 05:38:27.850516', NULL, 1, '93 Farnham Street', '1', NULL, 2, NULL, 'google', '117679618749034714503');
INSERT INTO public.st_users VALUES (35, 'test2', 'test2', 'vince.jennings102@gmail.com', 1, '2025-08-10 00:26:32.902823', '2025-09-28 08:12:27.706428', NULL, 2, NULL, '5t2tgw', NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (62, 'Vince Jennings', 'VinceTTC', 'vince.jennings@ttc.org.nz', 1, '2026-02-16 03:13:52.208284', '2026-02-17 04:25:40.4864', '1990-03-23', NULL, '93c', NULL, NULL, 2, NULL, 'google', '108559076374625333492');
INSERT INTO public.st_users VALUES (1, 'vince jennings3', 'vince', 'vince.jennings1003@gmail.com', 2, '2024-09-24 07:20:41.0626', '2025-09-28 08:12:47.917242', NULL, 2, '93 Farnham Street, Mornington', '7654', NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (16, 'vince jennings', 'vinceAPPtst', 'vince.apptesting@gmail.com', 1, '2024-11-14 10:47:04.549148', '2026-02-28 09:20:08.180698', '0001-01-26', 1, '93 Farnham Street', '1234', NULL, 2, NULL, 'google', '107191100686708556936');
INSERT INTO public.st_users VALUES (20, 'admin1', 'admin1', 'admin1@test.localhost', 1, '2025-06-19 10:07:04.161727', '2025-08-05 06:36:12.347883', NULL, 1, '93 Farnham St', '1111', NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (38, 'Vince5', 'vince5', 'vince.jennings5@gmail.com', 0, '2025-09-29 01:07:41.120843', '2025-09-29 01:07:41.120843', NULL, NULL, NULL, NULL, NULL, 0, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (17, 'Vince3', 'vince3', 'vince.jennings3@gmail.com', 1, '2024-11-25 03:26:18.444765', '2025-10-08 01:00:43.243508', NULL, 3, '93 Farnham Street', '1234A', NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (39, 'Vince6', 'vince6', 'vince.jennings6@gmail.com', 0, '2025-09-29 01:21:03.126849', '2025-09-29 01:21:29.980711', '1990-06-06', NULL, NULL, NULL, NULL, 2, NULL, NULL, NULL);
INSERT INTO public.st_users VALUES (37, 'Vince4', 'vince4', 'vince.jennings4@gmail.com', 0, '2025-09-28 23:31:54.011031', '2025-09-30 08:46:30.536594', NULL, NULL, NULL, NULL, NULL, 2, NULL, NULL, NULL);


--
-- TOC entry 3376 (class 0 OID 0)
-- Dependencies: 200
-- Name: at_booking_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_booking_users_id_seq', 27, true);


--
-- TOC entry 3377 (class 0 OID 0)
-- Dependencies: 202
-- Name: at_bookings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_bookings_id_seq', 37, true);


--
-- TOC entry 3378 (class 0 OID 0)
-- Dependencies: 204
-- Name: at_group_bookings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_group_bookings_id_seq', 2, true);


--
-- TOC entry 3379 (class 0 OID 0)
-- Dependencies: 206
-- Name: at_trip_cost_groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trip_cost_groups_id_seq', 2, true);


--
-- TOC entry 3380 (class 0 OID 0)
-- Dependencies: 208
-- Name: at_trip_costs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trip_costs_id_seq', 10, true);


--
-- TOC entry 3381 (class 0 OID 0)
-- Dependencies: 210
-- Name: at_trips_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trips_id_seq', 8, true);


--
-- TOC entry 3382 (class 0 OID 0)
-- Dependencies: 212
-- Name: at_user_payments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_user_payments_id_seq', 1, true);


--
-- TOC entry 3383 (class 0 OID 0)
-- Dependencies: 214
-- Name: et_access_level_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_access_level_id_seq', 6, true);


--
-- TOC entry 3384 (class 0 OID 0)
-- Dependencies: 216
-- Name: et_access_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_access_type_id_seq', 3, true);


--
-- TOC entry 3385 (class 0 OID 0)
-- Dependencies: 218
-- Name: et_booking_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_booking_status_id_seq', 4, true);


--
-- TOC entry 3386 (class 0 OID 0)
-- Dependencies: 220
-- Name: et_member_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_member_status_id_seq', 3, true);


--
-- TOC entry 3387 (class 0 OID 0)
-- Dependencies: 222
-- Name: et_resource_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_resource_id_seq', 20, true);


--
-- TOC entry 3388 (class 0 OID 0)
-- Dependencies: 224
-- Name: et_seasons_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_seasons_id_seq', 3, true);


--
-- TOC entry 3389 (class 0 OID 0)
-- Dependencies: 226
-- Name: et_trip_difficulty_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_difficulty_id_seq', 4, true);


--
-- TOC entry 3390 (class 0 OID 0)
-- Dependencies: 228
-- Name: et_trip_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_status_id_seq', 4, true);


--
-- TOC entry 3391 (class 0 OID 0)
-- Dependencies: 230
-- Name: et_trip_types_trip_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_types_trip_type_id_seq', 7, true);


--
-- TOC entry 3392 (class 0 OID 0)
-- Dependencies: 232
-- Name: et_user_account_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_user_account_status_id_seq', 6, true);


--
-- TOC entry 3393 (class 0 OID 0)
-- Dependencies: 234
-- Name: et_user_age_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_user_age_group_id_seq', 5, true);


--
-- TOC entry 3394 (class 0 OID 0)
-- Dependencies: 236
-- Name: st_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_group_id_seq', 4, true);


--
-- TOC entry 3395 (class 0 OID 0)
-- Dependencies: 238
-- Name: st_group_resource_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_group_resource_id_seq', 41, true);


--
-- TOC entry 3396 (class 0 OID 0)
-- Dependencies: 240
-- Name: st_token_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_token_id_seq', 591, true);


--
-- TOC entry 3397 (class 0 OID 0)
-- Dependencies: 242
-- Name: st_user_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_user_group_id_seq', 20, true);


--
-- TOC entry 3398 (class 0 OID 0)
-- Dependencies: 244
-- Name: st_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_users_id_seq', 76, true);


--
-- TOC entry 3115 (class 2606 OID 16432)
-- Name: at_booking_people at_booking_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT at_booking_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3117 (class 2606 OID 16448)
-- Name: at_bookings at_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT at_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3119 (class 2606 OID 16459)
-- Name: at_group_bookings at_group_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_group_bookings
    ADD CONSTRAINT at_group_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3121 (class 2606 OID 16469)
-- Name: at_trip_cost_groups at_trip_cost_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_cost_groups
    ADD CONSTRAINT at_trip_cost_groups_pkey PRIMARY KEY (id);


--
-- TOC entry 3123 (class 2606 OID 16479)
-- Name: at_trip_costs at_trip_costs_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_costs
    ADD CONSTRAINT at_trip_costs_pkey PRIMARY KEY (id);


--
-- TOC entry 3125 (class 2606 OID 16496)
-- Name: at_trips at_trips_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trips
    ADD CONSTRAINT at_trips_pkey PRIMARY KEY (id);


--
-- TOC entry 3127 (class 2606 OID 16506)
-- Name: at_user_payments at_user_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_user_payments
    ADD CONSTRAINT at_user_payments_pkey PRIMARY KEY (id);


--
-- TOC entry 3129 (class 2606 OID 16516)
-- Name: et_access_level et_access_level_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_level
    ADD CONSTRAINT et_access_level_pkey PRIMARY KEY (id);


--
-- TOC entry 3131 (class 2606 OID 16526)
-- Name: et_access_type et_access_type_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_type
    ADD CONSTRAINT et_access_type_pkey PRIMARY KEY (id);


--
-- TOC entry 3133 (class 2606 OID 16536)
-- Name: et_booking_status et_booking_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_booking_status
    ADD CONSTRAINT et_booking_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3135 (class 2606 OID 16546)
-- Name: et_member_status et_member_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_member_status
    ADD CONSTRAINT et_member_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3137 (class 2606 OID 16556)
-- Name: et_resource et_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_resource
    ADD CONSTRAINT et_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3139 (class 2606 OID 16566)
-- Name: et_seasons et_seasons_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_seasons
    ADD CONSTRAINT et_seasons_pkey PRIMARY KEY (id);


--
-- TOC entry 3141 (class 2606 OID 16576)
-- Name: et_trip_difficulty et_trip_difficulty_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_difficulty
    ADD CONSTRAINT et_trip_difficulty_pkey PRIMARY KEY (id);


--
-- TOC entry 3143 (class 2606 OID 16586)
-- Name: et_trip_status et_trip_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_status
    ADD CONSTRAINT et_trip_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3145 (class 2606 OID 16596)
-- Name: et_trip_type et_trip_types_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_type
    ADD CONSTRAINT et_trip_types_pkey PRIMARY KEY (id);


--
-- TOC entry 3147 (class 2606 OID 16609)
-- Name: et_user_account_status et_user_account_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_account_status
    ADD CONSTRAINT et_user_account_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3149 (class 2606 OID 16619)
-- Name: et_user_age_groups et_user_age_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_age_groups
    ADD CONSTRAINT et_user_age_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3151 (class 2606 OID 16630)
-- Name: st_group st_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group
    ADD CONSTRAINT st_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3153 (class 2606 OID 16641)
-- Name: st_group_resource st_group_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group_resource
    ADD CONSTRAINT st_group_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3155 (class 2606 OID 16652)
-- Name: st_token st_token_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_token
    ADD CONSTRAINT st_token_pkey PRIMARY KEY (id);


--
-- TOC entry 3157 (class 2606 OID 16662)
-- Name: st_user_group st_user_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_user_group
    ADD CONSTRAINT st_user_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3159 (class 2606 OID 16679)
-- Name: st_users st_users_email_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_email_key UNIQUE (email);


--
-- TOC entry 3161 (class 2606 OID 16677)
-- Name: st_users st_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3163 (class 2606 OID 16681)
-- Name: st_users st_users_username_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_username_key UNIQUE (username);


--
-- TOC entry 3167 (class 2620 OID 16772)
-- Name: at_booking_people set_modified_timestamp_at_booking_people; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_booking_people BEFORE UPDATE ON public.at_booking_people FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3168 (class 2620 OID 16771)
-- Name: at_bookings set_modified_timestamp_at_bookings; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_bookings BEFORE UPDATE ON public.at_bookings FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3169 (class 2620 OID 16752)
-- Name: at_group_bookings set_modified_timestamp_at_group_bookings; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_group_bookings BEFORE UPDATE ON public.at_group_bookings FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3170 (class 2620 OID 16753)
-- Name: at_trip_cost_groups set_modified_timestamp_at_trip_cost_groups; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trip_cost_groups BEFORE UPDATE ON public.at_trip_cost_groups FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3171 (class 2620 OID 16754)
-- Name: at_trip_costs set_modified_timestamp_at_trip_costs; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trip_costs BEFORE UPDATE ON public.at_trip_costs FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3172 (class 2620 OID 16755)
-- Name: at_trips set_modified_timestamp_at_trips; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_trips BEFORE UPDATE ON public.at_trips FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3173 (class 2620 OID 16756)
-- Name: at_user_payments set_modified_timestamp_at_user_payments; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_at_user_payments BEFORE UPDATE ON public.at_user_payments FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3174 (class 2620 OID 16757)
-- Name: et_access_level set_modified_timestamp_et_access_level; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_access_level BEFORE UPDATE ON public.et_access_level FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3175 (class 2620 OID 16758)
-- Name: et_access_type set_modified_timestamp_et_access_type; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_access_type BEFORE UPDATE ON public.et_access_type FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3176 (class 2620 OID 16774)
-- Name: et_booking_status set_modified_timestamp_et_booking_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_booking_status BEFORE UPDATE ON public.et_booking_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3177 (class 2620 OID 16759)
-- Name: et_member_status set_modified_timestamp_et_member_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_member_status BEFORE UPDATE ON public.et_member_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3178 (class 2620 OID 16760)
-- Name: et_resource set_modified_timestamp_et_resource; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_resource BEFORE UPDATE ON public.et_resource FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3179 (class 2620 OID 16761)
-- Name: et_seasons set_modified_timestamp_et_seasons; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_seasons BEFORE UPDATE ON public.et_seasons FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3180 (class 2620 OID 16762)
-- Name: et_trip_difficulty set_modified_timestamp_et_trip_difficulty; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_difficulty BEFORE UPDATE ON public.et_trip_difficulty FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3181 (class 2620 OID 16763)
-- Name: et_trip_status set_modified_timestamp_et_trip_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_status BEFORE UPDATE ON public.et_trip_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3182 (class 2620 OID 16764)
-- Name: et_trip_type set_modified_timestamp_et_trip_type; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_trip_type BEFORE UPDATE ON public.et_trip_type FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3183 (class 2620 OID 16765)
-- Name: et_user_account_status set_modified_timestamp_et_user_account_status; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_user_account_status BEFORE UPDATE ON public.et_user_account_status FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3184 (class 2620 OID 16766)
-- Name: et_user_age_groups set_modified_timestamp_et_user_age_groups; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_et_user_age_groups BEFORE UPDATE ON public.et_user_age_groups FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3185 (class 2620 OID 16767)
-- Name: st_group set_modified_timestamp_st_group; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_group BEFORE UPDATE ON public.st_group FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3186 (class 2620 OID 16768)
-- Name: st_group_resource set_modified_timestamp_st_group_resource; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_group_resource BEFORE UPDATE ON public.st_group_resource FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3187 (class 2620 OID 16769)
-- Name: st_token set_modified_timestamp_st_token; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_token BEFORE UPDATE ON public.st_token FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3188 (class 2620 OID 16770)
-- Name: st_user_group set_modified_timestamp_st_user_group; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_user_group BEFORE UPDATE ON public.st_user_group FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3189 (class 2620 OID 16773)
-- Name: st_users set_modified_timestamp_st_users; Type: TRIGGER; Schema: public; Owner: myuser
--

CREATE TRIGGER set_modified_timestamp_st_users BEFORE UPDATE ON public.st_users FOR EACH ROW EXECUTE FUNCTION public.update_modified_column();


--
-- TOC entry 3164 (class 2606 OID 16688)
-- Name: at_booking_people booking_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT booking_id_fkey FOREIGN KEY (booking_id) REFERENCES public.at_bookings(id) NOT VALID;


--
-- TOC entry 3166 (class 2606 OID 16698)
-- Name: at_bookings bookings_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT bookings_status_id_fkey FOREIGN KEY (booking_status_id) REFERENCES public.et_booking_status(id) NOT VALID;


--
-- TOC entry 3165 (class 2606 OID 16693)
-- Name: at_booking_people user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (person_id) REFERENCES public.st_users(id) NOT VALID;


--
-- TOC entry 3375 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: myuser
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2026-02-28 22:44:06

--
-- PostgreSQL database dump complete
--

\unrestrict lpwpapLssGMobU8OiciWKYaRc5xaaQXvshTEjZIQN8qg54UBPGUoKqdx4rODbD3

