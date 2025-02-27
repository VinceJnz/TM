--
-- PostgreSQL database dump
--

-- Dumped from database version 13.14 (Debian 13.14-1.pgdg120+2)
-- Dumped by pg_dump version 17.1

-- Started on 2025-02-27 19:36:56

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

DROP DATABASE mydatabase;
--
-- TOC entry 3337 (class 1262 OID 16384)
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

-- *not* creating schema, since initdb creates it


ALTER SCHEMA public OWNER TO myuser;

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
    payment_date date
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
    trip_cost_group_id integer
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
    user_account_hidden boolean
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
-- TOC entry 3287 (class 0 OID 16419)
-- Dependencies: 201
-- Data for Name: at_booking_people; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_booking_people VALUES (23, 17, 34, 16, 'qqqq', '2024-12-20 09:14:36.347532', '2024-12-20 09:14:36.347532');
INSERT INTO public.at_booking_people VALUES (24, 17, 34, 17, 'nnn', '2024-12-26 06:13:05.649785', '2024-12-26 06:13:05.649785');
INSERT INTO public.at_booking_people VALUES (10, 16, 10, 2, '123lkjo69696', '2024-10-06 10:10:55.298416', '2024-10-06 10:10:55.298416');
INSERT INTO public.at_booking_people VALUES (12, 16, 10, 3, 'XXXYY', '2024-10-07 01:36:09.828853', '2024-10-07 01:36:09.828853');
INSERT INTO public.at_booking_people VALUES (16, 16, 7, 2, 'yyyyyy', '2024-10-11 06:18:35.343878', '2024-10-11 06:18:35.343878');
INSERT INTO public.at_booking_people VALUES (21, 16, 7, 3, 'hhhhh', '2024-10-11 22:19:35.215926', '2024-10-11 22:19:35.215926');
INSERT INTO public.at_booking_people VALUES (22, 16, 13, 1, 'nnnnn', '2024-10-11 23:52:30.138021', '2024-10-11 23:52:30.138021');
INSERT INTO public.at_booking_people VALUES (4, 16, 7, 1, 'A first booking', '2024-10-04 09:08:20.931727', '2024-10-04 09:08:20.931727');


--
-- TOC entry 3289 (class 0 OID 16435)
-- Dependencies: 203
-- Data for Name: at_bookings; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_bookings VALUES (10, 16, 'Booking 2', '2024-10-06 00:00:00', '2024-10-20 00:00:00', 1, '2024-10-06 09:14:26.265978', '2024-12-20 09:08:11.157029', 2, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (18, 16, 'Vince''s booking', '0001-01-01 00:00:00', '0001-01-01 00:00:00', 1, '2024-10-16 02:51:04.680147', '2024-12-20 09:08:11.157029', 2, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (14, 16, 'a random booking', '2024-11-02 00:00:00', '2024-11-01 00:00:00', 1, '2024-10-13 22:44:38.713194', '2024-12-20 09:08:11.157029', 1, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (27, 16, 'wow more bookings', '2024-12-01 00:00:00', '2024-12-05 00:00:00', 1, '2024-10-18 00:12:21.727307', '2024-12-20 09:08:11.157029', 2, NULL, NULL, NULL, NULL);
INSERT INTO public.at_bookings VALUES (13, 16, 'another one', '2024-11-01 00:00:00', '2024-12-01 00:00:00', 3, '2024-10-11 23:46:36.105431', '2024-12-20 09:08:11.157029', 1, NULL, NULL, 200.10, '2024-10-28');
INSERT INTO public.at_bookings VALUES (7, 16, 'A first booking', '2024-11-03 00:00:00', '2024-11-07 00:00:00', 1, '2024-09-29 10:05:59.254752', '2024-12-20 09:08:11.157029', 1, NULL, NULL, 213.45, NULL);
INSERT INTO public.at_bookings VALUES (34, 17, 'kkkeeeY', '2024-11-01 00:00:00', '2024-12-01 00:00:00', 1, '2024-12-20 09:10:07.684579', '2025-01-01 08:23:16.960475', 1, NULL, NULL, NULL, NULL);


--
-- TOC entry 3291 (class 0 OID 16451)
-- Dependencies: 205
-- Data for Name: at_group_bookings; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_group_bookings VALUES (1, 'Vince''s group', 0, '2024-10-29 23:01:14.192148', '2024-10-29 23:01:14.192148');


--
-- TOC entry 3293 (class 0 OID 16462)
-- Dependencies: 207
-- Data for Name: at_trip_cost_groups; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_trip_cost_groups VALUES (1, 'Lodge winter rate', '2024-10-26 06:56:50.644474', '2024-10-26 06:56:50.644474');


--
-- TOC entry 3295 (class 0 OID 16472)
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
-- TOC entry 3297 (class 0 OID 16482)
-- Dependencies: 211
-- Data for Name: at_trips; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.at_trips VALUES (7, 16, '3rd trip', NULL, '2024-10-18', '2024-10-19', 1, '2024-10-18 00:11:25.195676', '2024-10-18 00:11:25.195676', 1, 2, 1, 1);
INSERT INTO public.at_trips VALUES (2, 16, 'A new trip', NULL, '2024-12-01', '2024-12-11', 3, '2024-10-11 21:38:01.098156', '2024-10-11 21:38:01.098156', 1, 1, 2, 1);
INSERT INTO public.at_trips VALUES (1, 16, 'Trip1a', NULL, '2024-11-01', '2024-12-01', 3, '2024-10-10 10:37:23.829893', '2024-10-10 10:37:23.829893', 3, 4, 3, 1);
INSERT INTO public.at_trips VALUES (6, 16, 'A fantastic trip a!', NULL, '2024-10-17', '2024-10-17', 11, '2024-10-17 23:57:28.0255', '2024-10-17 23:57:28.0255', 1, 1, 1, 1);


--
-- TOC entry 3299 (class 0 OID 16499)
-- Dependencies: 213
-- Data for Name: at_user_payments; Type: TABLE DATA; Schema: public; Owner: myuser
--



--
-- TOC entry 3301 (class 0 OID 16509)
-- Dependencies: 215
-- Data for Name: et_access_level; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_access_level VALUES (1, 'none', NULL, '2024-11-24 23:23:04.959786', '2024-11-24 23:23:04.959786');
INSERT INTO public.et_access_level VALUES (2, 'get', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (3, 'put', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (4, 'post', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');
INSERT INTO public.et_access_level VALUES (5, 'delete', NULL, '2024-11-24 23:23:32.409579', '2024-11-24 23:23:32.409579');


--
-- TOC entry 3303 (class 0 OID 16519)
-- Dependencies: 217
-- Data for Name: et_access_type; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_access_type VALUES (2, 'owner', NULL, '2024-11-24 23:15:13.207337', '2024-11-24 23:15:13.207337');
INSERT INTO public.et_access_type VALUES (1, 'admin', NULL, '2024-11-24 23:14:57.462385', '2024-11-24 23:14:57.462385');


--
-- TOC entry 3305 (class 0 OID 16529)
-- Dependencies: 219
-- Data for Name: et_booking_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_booking_status VALUES (1, 'New', '2024-09-29 10:11:30.906822', '2024-09-29 10:11:30.906822');
INSERT INTO public.et_booking_status VALUES (2, 'Cancelled', '2024-09-30 08:29:27.323855', '2024-09-30 08:29:27.323855');
INSERT INTO public.et_booking_status VALUES (3, 'Paid', '2024-09-30 08:29:41.411134', '2024-09-30 08:29:41.411134');


--
-- TOC entry 3307 (class 0 OID 16539)
-- Dependencies: 221
-- Data for Name: et_member_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_member_status VALUES (1, 'Yes', '2025-01-12 08:36:38.416883', '2025-01-12 08:36:38.416883');
INSERT INTO public.et_member_status VALUES (2, 'No', '2025-01-12 08:36:12.929545', '2025-01-12 08:36:12.929545');


--
-- TOC entry 3309 (class 0 OID 16549)
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


--
-- TOC entry 3311 (class 0 OID 16559)
-- Dependencies: 225
-- Data for Name: et_seasons; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_seasons VALUES (2, 'Winter', '2024-10-26 05:28:39.461983', '2024-10-26 05:28:39.461983', NULL, NULL);
INSERT INTO public.et_seasons VALUES (1, 'Summer', '2024-10-26 05:28:32.741256', '2024-10-26 05:28:32.741256', NULL, NULL);


--
-- TOC entry 3313 (class 0 OID 16569)
-- Dependencies: 227
-- Data for Name: et_trip_difficulty; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_trip_difficulty VALUES (1, 'Medium to Fit', '2024-10-26 06:03:17.094307', '2024-10-26 06:03:17.094307', 'MF', 'Up to 8 hours per day, pace faster than M, off track and above bush line travel to be expected.');
INSERT INTO public.et_trip_difficulty VALUES (2, 'Easy', '2024-10-26 06:03:53.30417', '2024-10-26 06:03:53.30417', 'E', 'Up to 4 hours per day, pace slower than EM.');
INSERT INTO public.et_trip_difficulty VALUES (3, 'Slow medium', '2024-10-28 06:35:23.182334', '2024-10-28 06:35:23.182334', 'SM', 'Medium trip at a slower pace than the standard pace');


--
-- TOC entry 3315 (class 0 OID 16579)
-- Dependencies: 229
-- Data for Name: et_trip_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_trip_status VALUES (1, 'New', '2024-10-10 10:34:38.851101', '2024-10-10 10:34:38.851101');
INSERT INTO public.et_trip_status VALUES (2, 'Cancelled', '2024-10-10 10:34:49.204346', '2024-10-10 10:34:49.204346');
INSERT INTO public.et_trip_status VALUES (3, 'Completed', '2024-10-10 10:34:56.149138', '2024-10-10 10:34:56.149138');


--
-- TOC entry 3317 (class 0 OID 16589)
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
-- TOC entry 3319 (class 0 OID 16599)
-- Dependencies: 233
-- Data for Name: et_user_account_status; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_user_account_status VALUES (0, 'New', '2025-01-12 07:47:04.2678', '2025-01-12 07:47:04.2678', 'A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin.');
INSERT INTO public.et_user_account_status VALUES (1, 'Verified', '2024-12-01 09:36:35.177495', '2024-12-01 09:36:35.177495', 'The email address has been verified. An Admin now needs to activate the account.');
INSERT INTO public.et_user_account_status VALUES (2, 'Active', '2024-12-01 09:36:50.00299', '2024-12-01 09:36:50.00299', 'An account that has been activated, and is currently active.');
INSERT INTO public.et_user_account_status VALUES (3, 'Disabled', '2025-01-12 08:28:36.131138', '2025-01-12 08:28:36.131138', 'An account that has been disabled.');
INSERT INTO public.et_user_account_status VALUES (4, 'Reset', '2025-01-12 08:29:16.932961', '2025-01-12 08:29:16.932961', 'The account is flagged for a password reset. The user will be informed at the next login.');


--
-- TOC entry 3321 (class 0 OID 16612)
-- Dependencies: 235
-- Data for Name: et_user_age_groups; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.et_user_age_groups VALUES (1, 'Infant', '2024-10-26 07:08:21.941336', '2024-10-26 07:08:21.941336');
INSERT INTO public.et_user_age_groups VALUES (2, 'Adult', '2024-10-26 07:08:36.386124', '2024-10-26 07:08:36.386124');
INSERT INTO public.et_user_age_groups VALUES (3, 'Child', '2024-10-26 07:08:42.383391', '2024-10-26 07:08:42.383391');
INSERT INTO public.et_user_age_groups VALUES (4, 'Youth', '2024-10-26 07:09:03.922737', '2024-10-26 07:09:03.922737');


--
-- TOC entry 3323 (class 0 OID 16622)
-- Dependencies: 237
-- Data for Name: st_group; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_group VALUES (1, 'Sys Admin', 'System admin', true, '2024-11-24 23:31:26.62785', '2024-11-24 23:31:26.62785');
INSERT INTO public.st_group VALUES (3, 'App Admin', 'Application admin', false, '2024-12-05 23:46:11.129967', '2024-12-05 23:46:11.129967');
INSERT INTO public.st_group VALUES (2, 'User', 'app users', false, '2024-12-05 06:57:30.344557', '2024-12-05 06:57:30.344557');


--
-- TOC entry 3325 (class 0 OID 16633)
-- Dependencies: 239
-- Data for Name: st_group_resource; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_group_resource VALUES (1, 1, 1, 2, 1, false, '2024-11-24 23:34:35.86487', '2024-11-24 23:34:35.86487');
INSERT INTO public.st_group_resource VALUES (3, 1, 1, 3, 1, false, '2024-11-24 23:35:01.83461', '2024-11-24 23:35:01.83461');
INSERT INTO public.st_group_resource VALUES (4, 1, 1, 4, 1, false, '2024-11-24 23:35:17.568932', '2024-11-24 23:35:17.568932');
INSERT INTO public.st_group_resource VALUES (5, 1, 1, 5, 1, false, '2024-11-24 23:35:39.544196', '2024-11-24 23:35:39.544196');
INSERT INTO public.st_group_resource VALUES (6, 1, 2, 2, 1, false, '2024-11-24 23:40:19.506411', '2024-11-24 23:40:19.506411');
INSERT INTO public.st_group_resource VALUES (7, 1, 2, 3, 1, false, '2024-11-24 23:42:30.994064', '2024-11-24 23:42:30.994064');
INSERT INTO public.st_group_resource VALUES (8, 2, 2, 2, 2, false, '2024-12-05 06:59:42.806108', '2024-12-05 06:59:42.806108');
INSERT INTO public.st_group_resource VALUES (12, 2, 8, 2, 2, false, '2024-12-05 07:13:57.298371', '2024-12-05 07:13:57.298371');
INSERT INTO public.st_group_resource VALUES (13, 2, 9, 2, 2, false, '2024-12-05 07:52:39.946898', '2024-12-05 07:52:39.946898');
INSERT INTO public.st_group_resource VALUES (18, 2, 3, 5, 2, false, '2024-12-05 07:56:52.363959', '2024-12-05 07:56:52.363959');
INSERT INTO public.st_group_resource VALUES (17, 2, 3, 4, 2, false, '2024-12-05 07:56:43.626614', '2024-12-05 07:56:43.626614');
INSERT INTO public.st_group_resource VALUES (16, 2, 3, 3, 2, false, '2024-12-05 07:56:32.357164', '2024-12-05 07:56:32.357164');
INSERT INTO public.st_group_resource VALUES (15, 2, 3, 2, 2, false, '2024-12-05 07:56:06.118582', '2024-12-05 07:56:06.118582');
INSERT INTO public.st_group_resource VALUES (14, 2, 10, 2, 2, false, '2024-12-05 07:52:49.566355', '2024-12-05 07:52:49.566355');
INSERT INTO public.st_group_resource VALUES (19, 2, 11, 2, 2, false, '2024-12-13 07:26:15.856341', '2024-12-13 07:26:15.856341');
INSERT INTO public.st_group_resource VALUES (20, 2, 13, 2, 2, false, '2024-12-13 07:31:44.260408', '2024-12-13 07:31:44.260408');
INSERT INTO public.st_group_resource VALUES (21, 2, 15, 2, 2, false, '2024-12-13 07:32:38.319088', '2024-12-13 07:32:38.319088');
INSERT INTO public.st_group_resource VALUES (22, 2, 14, 2, 2, false, '2024-12-13 07:32:54.10676', '2024-12-13 07:32:54.10676');
INSERT INTO public.st_group_resource VALUES (23, 2, 1, 2, 2, false, '2024-12-13 07:51:52.949741', '2024-12-13 07:51:52.949741');
INSERT INTO public.st_group_resource VALUES (24, 2, 16, 2, 2, false, '2024-12-14 05:38:15.734325', '2024-12-14 05:38:15.734325');
INSERT INTO public.st_group_resource VALUES (25, 2, 17, 2, 2, false, '2024-12-20 09:13:11.812551', '2024-12-20 09:13:11.812551');
INSERT INTO public.st_group_resource VALUES (26, 2, 17, 3, 2, false, '2024-12-20 09:13:24.920667', '2024-12-20 09:13:24.920667');
INSERT INTO public.st_group_resource VALUES (27, 2, 17, 4, 2, false, '2024-12-20 09:13:32.890183', '2024-12-20 09:13:32.890183');
INSERT INTO public.st_group_resource VALUES (28, 2, 17, 5, 2, false, '2024-12-20 09:13:42.950509', '2024-12-20 09:13:42.950509');


--
-- TOC entry 3327 (class 0 OID 16644)
-- Dependencies: 241
-- Data for Name: st_token; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_token VALUES (6, 16, 'session', '172.21.0.1:35340', '6c7b4b0a-9a97-41b5-87f4-24ed14a4fb6b', '2025-02-27 06:07:49.856164+00', '2025-02-28 06:07:49.856164+00', '2025-02-27 06:07:49.858964', '2025-02-27 06:07:49.858964', true);


--
-- TOC entry 3329 (class 0 OID 16655)
-- Dependencies: 243
-- Data for Name: st_user_group; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_user_group VALUES (1, 16, 1, '2024-11-24 23:31:58.500845', '2024-11-24 23:31:58.500845');
INSERT INTO public.st_user_group VALUES (2, 17, 2, '2024-11-28 03:54:26.342778', '2024-11-28 03:54:26.342778');
INSERT INTO public.st_user_group VALUES (3, 3, 2, '2024-12-27 03:14:27.828175', '2024-12-27 03:14:27.828175');
INSERT INTO public.st_user_group VALUES (4, 1, 2, '2024-12-27 03:14:42.095962', '2024-12-27 03:14:42.095962');
INSERT INTO public.st_user_group VALUES (5, 2, 2, '2024-12-27 03:14:50.895662', '2024-12-27 03:14:50.895662');
INSERT INTO public.st_user_group VALUES (6, 15, 2, '2024-12-27 03:15:03.932908', '2024-12-27 03:15:03.932908');


--
-- TOC entry 3331 (class 0 OID 16665)
-- Dependencies: 245
-- Data for Name: st_users; Type: TABLE DATA; Schema: public; Owner: myuser
--

INSERT INTO public.st_users VALUES (16, 'Vince2', 'vince2', 'vince.jennings2@gmail.com', 1, '2024-11-14 10:47:04.549148', '2025-01-14 03:16:49.450163', NULL, 1, '\x6e13b8cf5c83e5ac', '\x0201163202da345a8efc7b5ed45e51f13ca1e29a266617d86ff8e7f2316ad46d064081b04939baed0174e4f01f01273800ead90b86d50aadfba5101fc6edda2d393400e5cdd369e29723a97d7c94b9a0e32c97ff84dd76f7d89a4fba3ed310dc34a49c9d4fe0ac8d2026f6e30a7ba4400f7ca2a6c95995cca8ab18aa3276dc3bee40d1e240573ab2acb91595aa51c3577ce0d93bda274029702323f30e54467b9416c0727f2f4d237812ffbaceef49c2325cc2e1c3686069d74f8bbda8799599e0283048257875f92518886497f959832e1c148839cc546aba56a8e4591653dbac879db32388a96f7fb5ab99d950883c99bf3d01df55fefa0009b2405851c99206fe9cba482931e2a42de17b62d76f500d08c16f44263434c7e762f169574035563b50c6095df438d2a9fab92c3258ccd6484795a11466d7c051ac88a26bb493fa4af4cc7d33b31864968868e0cf4c83342d6c9ed73ff87a2c9cda1a8ede587a2010d359019d7a87290023933a680b98cc0d77831bdc8760f433efcc5bcf914ac7', '93 Farnham Street', '1234', NULL, 3, true);
INSERT INTO public.st_users VALUES (2, 'Donna Jennings12', 'donna', 'dj0040@gmail.com', 1, '2024-10-04 09:08:52.345413', '2025-01-14 07:10:06.688522', NULL, 1, NULL, NULL, '93 Farnham Street', '1234B', NULL, 1, NULL);
INSERT INTO public.st_users VALUES (1, 'vince jennings3', 'vince', 'vince.jennings@gmail.com', 2, '2024-09-24 07:20:41.0626', '2025-01-14 07:10:51.970566', NULL, 2, NULL, NULL, '93 Farnham Street, Mornington', '7654', NULL, 1, NULL);
INSERT INTO public.st_users VALUES (3, 'Dylan Jennings1', 'Dylan', 'dylan@dt.net.nz', 1, '2024-10-04 09:09:11.226469', '2025-01-14 07:11:07.063766', NULL, 4, NULL, NULL, '93 Farnham Street', '12347', NULL, 1, true);
INSERT INTO public.st_users VALUES (15, 'Vince1', 'vince1', 'vince.jennings1@gmail.com', 1, '2024-11-14 00:12:08.297532', '2024-12-01 09:46:07.659601', NULL, 1, '\x8f1a00b2822fe33d', '\x023f5f5b535a9ca04fe4bb95373f5a673103c1f033b2af4d3c8659fcff502ffea811668bd0531f976824ef1d2dbc50eb3ca9e4704e33601e081f621fd0c075d7cdd5fe49fb55ec672ee7773697dfc4e51b2682d5c349ef8368daaec799b07d62aa720eda12c198e2fcca6b860e304b1552bab7810a04fcc1e5d8e09ad61a67ae9711ed8df454347ec724a010d535723d319fda04b21747cfd1accf66efa4d9db969751c53600d58093b5b63dbc3fabadfa8d01b47077112d0039d2d162452371c77d6f7b61f9585d180109dc2ce8f0aca5d0e47cc393889e52f450678afd00de5cc691a20c920a9f9e603147b6485c2572d1f528ce7f31fb0bd634ed3359b7f5505fcc55bd6180d4877f1f08dc9da0faf7d7353b494c493d1e0f0ba3698fd8f7ab2a301e08acd9cfb4aef8e9d61ef136a91bc0de504f7f54d9b82a3498b991c5f34c79466b955e200a0ddbf66c6eaa769f4620b3fd3d5a3beda7297f039026a8197601e6fa8ca325382a26537c46b2d34569c053238df4889964d0f013b5fbb0e5', '93 Farnham Street', '1', NULL, 2, NULL);
INSERT INTO public.st_users VALUES (17, 'Vince3', 'vince3', 'vince.jennings3@gmail.com', 1, '2024-11-25 03:26:18.444765', '2025-01-14 07:11:43.894139', NULL, 3, '\x9eb1c2a76444a9e3', '\x025105d39fdf717ae3f733502ca3021cdaded71e783c4d49aab09c630597f17688f9a4247bd362e2201a2c1b97f7bce2a2702afd6eff379571314c6d19426ef9f6fd1bafbe2083c5af420110ba5d7c1749d412fc95401570f0ff5e44cb23ad7fbbf7308ca882797ff5f749052a05489a599d95919d7a59ba3f2a1e99f32a067c34f947e012b65887dcc066f3cf47dfec7c4c2328bebd2e32afdfe52367a2036161e860c5b54aa70f83c271f81fc178757a1b2705657ac5bb7be79e0ca6c26733a4927602787e71850f7899a1749a9e40818d09994ecd0f60a16c03efce3fc78aaba1f06d5557eab664fc772ebcbeb315fce5bb94ca972c65ab01676784c7d2c8e3d5fbc2941209e37878f47132db8348f67a49d613dde45c57632c1a2dbb199d25b008025c543fe9cca7de85932311caa476347cf58b5b42f76dfbe836848fe5d7a9e4bb1522ea3afa9f8e6f6ef010d3a5e6be154d0b0693e2d335eceb8658d8826c153c87e4805e8bad85bc2c5547b35fab0490b5a7141c5317998ccfc06496cc', '93 Farnham Street', '1234A', NULL, 3, NULL);


--
-- TOC entry 3339 (class 0 OID 0)
-- Dependencies: 200
-- Name: at_booking_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_booking_users_id_seq', 1, false);


--
-- TOC entry 3340 (class 0 OID 0)
-- Dependencies: 202
-- Name: at_bookings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_bookings_id_seq', 1, false);


--
-- TOC entry 3341 (class 0 OID 0)
-- Dependencies: 204
-- Name: at_group_bookings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_group_bookings_id_seq', 1, false);


--
-- TOC entry 3342 (class 0 OID 0)
-- Dependencies: 206
-- Name: at_trip_cost_groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trip_cost_groups_id_seq', 1, false);


--
-- TOC entry 3343 (class 0 OID 0)
-- Dependencies: 208
-- Name: at_trip_costs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trip_costs_id_seq', 1, false);


--
-- TOC entry 3344 (class 0 OID 0)
-- Dependencies: 210
-- Name: at_trips_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_trips_id_seq', 1, false);


--
-- TOC entry 3345 (class 0 OID 0)
-- Dependencies: 212
-- Name: at_user_payments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.at_user_payments_id_seq', 1, false);


--
-- TOC entry 3346 (class 0 OID 0)
-- Dependencies: 214
-- Name: et_access_level_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_access_level_id_seq', 1, false);


--
-- TOC entry 3347 (class 0 OID 0)
-- Dependencies: 216
-- Name: et_access_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_access_type_id_seq', 1, false);


--
-- TOC entry 3348 (class 0 OID 0)
-- Dependencies: 218
-- Name: et_booking_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_booking_status_id_seq', 1, false);


--
-- TOC entry 3349 (class 0 OID 0)
-- Dependencies: 220
-- Name: et_member_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_member_status_id_seq', 1, false);


--
-- TOC entry 3350 (class 0 OID 0)
-- Dependencies: 222
-- Name: et_resource_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_resource_id_seq', 18, true);


--
-- TOC entry 3351 (class 0 OID 0)
-- Dependencies: 224
-- Name: et_seasons_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_seasons_id_seq', 1, false);


--
-- TOC entry 3352 (class 0 OID 0)
-- Dependencies: 226
-- Name: et_trip_difficulty_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_difficulty_id_seq', 1, false);


--
-- TOC entry 3353 (class 0 OID 0)
-- Dependencies: 228
-- Name: et_trip_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_status_id_seq', 1, false);


--
-- TOC entry 3354 (class 0 OID 0)
-- Dependencies: 230
-- Name: et_trip_types_trip_type_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_trip_types_trip_type_id_seq', 1, false);


--
-- TOC entry 3355 (class 0 OID 0)
-- Dependencies: 232
-- Name: et_user_account_status_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_user_account_status_id_seq', 1, false);


--
-- TOC entry 3356 (class 0 OID 0)
-- Dependencies: 234
-- Name: et_user_age_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.et_user_age_group_id_seq', 1, false);


--
-- TOC entry 3357 (class 0 OID 0)
-- Dependencies: 236
-- Name: st_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_group_id_seq', 1, false);


--
-- TOC entry 3358 (class 0 OID 0)
-- Dependencies: 238
-- Name: st_group_resource_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_group_resource_id_seq', 1, false);


--
-- TOC entry 3359 (class 0 OID 0)
-- Dependencies: 240
-- Name: st_token_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_token_id_seq', 6, true);


--
-- TOC entry 3360 (class 0 OID 0)
-- Dependencies: 242
-- Name: st_user_group_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_user_group_id_seq', 1, false);


--
-- TOC entry 3361 (class 0 OID 0)
-- Dependencies: 244
-- Name: st_users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: myuser
--

SELECT pg_catalog.setval('public.st_users_id_seq', 1, false);


--
-- TOC entry 3103 (class 2606 OID 16432)
-- Name: at_booking_people at_booking_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT at_booking_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3105 (class 2606 OID 16448)
-- Name: at_bookings at_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT at_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3107 (class 2606 OID 16459)
-- Name: at_group_bookings at_group_bookings_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_group_bookings
    ADD CONSTRAINT at_group_bookings_pkey PRIMARY KEY (id);


--
-- TOC entry 3109 (class 2606 OID 16469)
-- Name: at_trip_cost_groups at_trip_cost_groups_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_cost_groups
    ADD CONSTRAINT at_trip_cost_groups_pkey PRIMARY KEY (id);


--
-- TOC entry 3111 (class 2606 OID 16479)
-- Name: at_trip_costs at_trip_costs_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trip_costs
    ADD CONSTRAINT at_trip_costs_pkey PRIMARY KEY (id);


--
-- TOC entry 3113 (class 2606 OID 16496)
-- Name: at_trips at_trips_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_trips
    ADD CONSTRAINT at_trips_pkey PRIMARY KEY (id);


--
-- TOC entry 3115 (class 2606 OID 16506)
-- Name: at_user_payments at_user_payments_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_user_payments
    ADD CONSTRAINT at_user_payments_pkey PRIMARY KEY (id);


--
-- TOC entry 3117 (class 2606 OID 16516)
-- Name: et_access_level et_access_level_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_level
    ADD CONSTRAINT et_access_level_pkey PRIMARY KEY (id);


--
-- TOC entry 3119 (class 2606 OID 16526)
-- Name: et_access_type et_access_type_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_access_type
    ADD CONSTRAINT et_access_type_pkey PRIMARY KEY (id);


--
-- TOC entry 3121 (class 2606 OID 16536)
-- Name: et_booking_status et_booking_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_booking_status
    ADD CONSTRAINT et_booking_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3123 (class 2606 OID 16546)
-- Name: et_member_status et_member_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_member_status
    ADD CONSTRAINT et_member_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3125 (class 2606 OID 16556)
-- Name: et_resource et_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_resource
    ADD CONSTRAINT et_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3127 (class 2606 OID 16566)
-- Name: et_seasons et_seasons_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_seasons
    ADD CONSTRAINT et_seasons_pkey PRIMARY KEY (id);


--
-- TOC entry 3129 (class 2606 OID 16576)
-- Name: et_trip_difficulty et_trip_difficulty_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_difficulty
    ADD CONSTRAINT et_trip_difficulty_pkey PRIMARY KEY (id);


--
-- TOC entry 3131 (class 2606 OID 16586)
-- Name: et_trip_status et_trip_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_status
    ADD CONSTRAINT et_trip_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3133 (class 2606 OID 16596)
-- Name: et_trip_type et_trip_types_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_trip_type
    ADD CONSTRAINT et_trip_types_pkey PRIMARY KEY (id);


--
-- TOC entry 3135 (class 2606 OID 16609)
-- Name: et_user_account_status et_user_account_status_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_account_status
    ADD CONSTRAINT et_user_account_status_pkey PRIMARY KEY (id);


--
-- TOC entry 3137 (class 2606 OID 16619)
-- Name: et_user_age_groups et_user_age_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.et_user_age_groups
    ADD CONSTRAINT et_user_age_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3139 (class 2606 OID 16630)
-- Name: st_group st_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group
    ADD CONSTRAINT st_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3141 (class 2606 OID 16641)
-- Name: st_group_resource st_group_resource_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_group_resource
    ADD CONSTRAINT st_group_resource_pkey PRIMARY KEY (id);


--
-- TOC entry 3143 (class 2606 OID 16652)
-- Name: st_token st_token_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_token
    ADD CONSTRAINT st_token_pkey PRIMARY KEY (id);


--
-- TOC entry 3145 (class 2606 OID 16662)
-- Name: st_user_group st_user_group_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_user_group
    ADD CONSTRAINT st_user_group_pkey PRIMARY KEY (id);


--
-- TOC entry 3147 (class 2606 OID 16679)
-- Name: st_users st_users_email_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_email_key UNIQUE (email);


--
-- TOC entry 3149 (class 2606 OID 16677)
-- Name: st_users st_users_pkey; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_pkey PRIMARY KEY (id);


--
-- TOC entry 3151 (class 2606 OID 16681)
-- Name: st_users st_users_username_key; Type: CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.st_users
    ADD CONSTRAINT st_users_username_key UNIQUE (username);


--
-- TOC entry 3152 (class 2606 OID 16688)
-- Name: at_booking_people booking_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT booking_id_fkey FOREIGN KEY (booking_id) REFERENCES public.at_bookings(id) NOT VALID;


--
-- TOC entry 3154 (class 2606 OID 16698)
-- Name: at_bookings bookings_status_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_bookings
    ADD CONSTRAINT bookings_status_id_fkey FOREIGN KEY (booking_status_id) REFERENCES public.et_booking_status(id) NOT VALID;


--
-- TOC entry 3153 (class 2606 OID 16693)
-- Name: at_booking_people user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: myuser
--

ALTER TABLE ONLY public.at_booking_people
    ADD CONSTRAINT user_id_fkey FOREIGN KEY (person_id) REFERENCES public.st_users(id) NOT VALID;


--
-- TOC entry 3338 (class 0 OID 0)
-- Dependencies: 4
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: myuser
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


-- Completed on 2025-02-27 19:36:56

--
-- PostgreSQL database dump complete
--

