-- Adminer 4.8.1 PostgreSQL 13.14 (Debian 13.14-1.pgdg120+2) dump

--\connect "mydatabase";

DROP TABLE IF EXISTS "at_booking_people";
DROP SEQUENCE IF EXISTS at_booking_users_id_seq;
CREATE SEQUENCE at_booking_users_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_booking_people" (
    "id" integer DEFAULT nextval('at_booking_users_id_seq') NOT NULL,
    "owner_id" integer DEFAULT '0' NOT NULL,
    "booking_id" integer DEFAULT '0' NOT NULL,
    "person_id" integer DEFAULT '0' NOT NULL,
    "notes" text,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "at_booking_users_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_booking_people" ("id", "owner_id", "booking_id", "person_id", "notes", "created", "modified") VALUES
(23,	17,	34,	16,	'qqqq',	'2024-12-20 09:14:36.347532',	'2024-12-20 09:14:36.347532'),
(24,	17,	34,	17,	'nnn',	'2024-12-26 06:13:05.649785',	'2024-12-26 06:13:05.649785'),
(10,	16,	10,	2,	'123lkjo69696',	'2024-10-06 10:10:55.298416',	'2024-10-06 10:10:55.298416'),
(12,	16,	10,	3,	'XXXYY',	'2024-10-07 01:36:09.828853',	'2024-10-07 01:36:09.828853'),
(16,	16,	7,	2,	'yyyyyy',	'2024-10-11 06:18:35.343878',	'2024-10-11 06:18:35.343878'),
(21,	16,	7,	3,	'hhhhh',	'2024-10-11 22:19:35.215926',	'2024-10-11 22:19:35.215926'),
(22,	16,	13,	1,	'nnnnn',	'2024-10-11 23:52:30.138021',	'2024-10-11 23:52:30.138021'),
(4,	16,	7,	1,	'A first booking',	'2024-10-04 09:08:20.931727',	'2024-10-04 09:08:20.931727');

DROP TABLE IF EXISTS "at_bookings";
DROP SEQUENCE IF EXISTS at_bookings_id_seq;
CREATE SEQUENCE at_bookings_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_bookings" (
    "id" integer DEFAULT nextval('at_bookings_id_seq') NOT NULL,
    "owner_id" integer DEFAULT '0' NOT NULL,
    "notes" text,
    "from_date" timestamp,
    "to_date" timestamp,
    "booking_status_id" integer DEFAULT '0' NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "trip_id" integer DEFAULT '0' NOT NULL,
    "booking_date" date,
    "group_booking_id" integer,
    "booking_price" numeric(8,2),
    "payment_date" date,
    CONSTRAINT "at_bookings_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_bookings" ("id", "owner_id", "notes", "from_date", "to_date", "booking_status_id", "created", "modified", "trip_id", "booking_date", "group_booking_id", "booking_price", "payment_date") VALUES
(10,	16,	'Booking 2',	'2024-10-06 00:00:00',	'2024-10-20 00:00:00',	1,	'2024-10-06 09:14:26.265978',	'2024-12-20 09:08:11.157029',	2,	NULL,	NULL,	NULL,	NULL),
(18,	16,	'Vince''s booking',	'0001-01-01 00:00:00',	'0001-01-01 00:00:00',	1,	'2024-10-16 02:51:04.680147',	'2024-12-20 09:08:11.157029',	2,	NULL,	NULL,	NULL,	NULL),
(14,	16,	'a random booking',	'2024-11-02 00:00:00',	'2024-11-01 00:00:00',	1,	'2024-10-13 22:44:38.713194',	'2024-12-20 09:08:11.157029',	1,	NULL,	NULL,	NULL,	NULL),
(27,	16,	'wow more bookings',	'2024-12-01 00:00:00',	'2024-12-05 00:00:00',	1,	'2024-10-18 00:12:21.727307',	'2024-12-20 09:08:11.157029',	2,	NULL,	NULL,	NULL,	NULL),
(13,	16,	'another one',	'2024-11-01 00:00:00',	'2024-12-01 00:00:00',	3,	'2024-10-11 23:46:36.105431',	'2024-12-20 09:08:11.157029',	1,	NULL,	NULL,	200.10,	'2024-10-28'),
(7,	16,	'A first booking',	'2024-11-03 00:00:00',	'2024-11-07 00:00:00',	1,	'2024-09-29 10:05:59.254752',	'2024-12-20 09:08:11.157029',	1,	NULL,	NULL,	213.45,	NULL),
(34,	17,	'kkkeeeY',	'2024-11-01 00:00:00',	'2024-12-01 00:00:00',	1,	'2024-12-20 09:10:07.684579',	'2025-01-01 08:23:16.960475',	1,	NULL,	NULL,	NULL,	NULL);

DELIMITER ;;

--CREATE TRIGGER "update_at_bookings_modified" BEFORE UPDATE ON "public"."at_bookings" FOR EACH ROW EXECUTE FUNCTION update_modified_column();;

DELIMITER ;

DROP TABLE IF EXISTS "at_group_bookings";
DROP SEQUENCE IF EXISTS at_group_bookings_id_seq;
CREATE SEQUENCE at_group_bookings_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_group_bookings" (
    "id" integer DEFAULT nextval('at_group_bookings_id_seq') NOT NULL,
    "group_name" character varying(255) NOT NULL,
    "owner_id" integer DEFAULT '0' NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "at_group_bookings_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_group_bookings" ("id", "group_name", "owner_id", "created", "modified") VALUES
(1,	'Vince''s group',	0,	'2024-10-29 23:01:14.192148',	'2024-10-29 23:01:14.192148');

DROP TABLE IF EXISTS "at_trip_cost_groups";
DROP SEQUENCE IF EXISTS at_trip_cost_groups_id_seq;
CREATE SEQUENCE at_trip_cost_groups_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_trip_cost_groups" (
    "id" integer DEFAULT nextval('at_trip_cost_groups_id_seq') NOT NULL,
    "description" character varying(50) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "at_trip_cost_groups_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_trip_cost_groups" ("id", "description", "created", "modified") VALUES
(1,	'Lodge winter rate',	'2024-10-26 06:56:50.644474',	'2024-10-26 06:56:50.644474');

DROP TABLE IF EXISTS "at_trip_costs";
DROP SEQUENCE IF EXISTS at_trip_costs_id_seq;
CREATE SEQUENCE at_trip_costs_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_trip_costs" (
    "id" integer DEFAULT nextval('at_trip_costs_id_seq') NOT NULL,
    "trip_cost_group_id" integer NOT NULL,
    "description" character varying(50),
    "member_status_id" integer NOT NULL,
    "user_age_group_id" integer NOT NULL,
    "season_id" integer NOT NULL,
    "amount" numeric(10,2) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "at_trip_costs_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_trip_costs" ("id", "trip_cost_group_id", "description", "member_status_id", "user_age_group_id", "season_id", "amount", "created", "modified") VALUES
(4,	1,	NULL,	1,	2,	2,	42.00,	'2024-10-28 06:36:22.313324',	'2024-10-28 06:36:22.313324'),
(5,	1,	NULL,	2,	2,	2,	63.00,	'2024-10-28 06:46:00.905423',	'2024-10-28 06:46:00.905423'),
(2,	1,	NULL,	1,	3,	2,	11.00,	'2024-10-26 07:17:22.832664',	'2024-10-26 07:17:22.832664'),
(3,	1,	NULL,	1,	4,	2,	21.00,	'2024-10-26 07:19:42.892956',	'2024-10-26 07:19:42.892956'),
(6,	1,	NULL,	1,	1,	2,	5.00,	'2025-01-14 07:07:35.112759',	'2025-01-14 07:07:35.112759'),
(7,	1,	NULL,	2,	1,	2,	6.00,	'2025-01-14 07:07:53.892965',	'2025-01-14 07:07:53.892965'),
(8,	1,	NULL,	2,	3,	2,	13.00,	'2025-01-14 07:08:06.32252',	'2025-01-14 07:08:06.32252'),
(9,	1,	NULL,	2,	4,	2,	25.00,	'2025-01-14 07:08:21.262404',	'2025-01-14 07:08:21.262404');

DROP TABLE IF EXISTS "at_trips";
DROP SEQUENCE IF EXISTS at_trips_id_seq;
CREATE SEQUENCE at_trips_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_trips" (
    "id" integer DEFAULT nextval('at_trips_id_seq') NOT NULL,
    "owner_id" integer DEFAULT '0' NOT NULL,
    "trip_name" text NOT NULL,
    "location" text,
    "from_date" date,
    "to_date" date,
    "max_participants" integer DEFAULT '0' NOT NULL,
    "created" timestamp DEFAULT now(),
    "modified" timestamp DEFAULT now(),
    "trip_status_id" integer DEFAULT '0' NOT NULL,
    "trip_type_id" integer DEFAULT '0' NOT NULL,
    "difficulty_level_id" integer,
    "trip_cost_group_id" integer,
    CONSTRAINT "at_trips_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "at_trips" ("id", "owner_id", "trip_name", "location", "from_date", "to_date", "max_participants", "created", "modified", "trip_status_id", "trip_type_id", "difficulty_level_id", "trip_cost_group_id") VALUES
(7,	16,	'3rd trip',	NULL,	'2024-10-18',	'2024-10-19',	1,	'2024-10-18 00:11:25.195676',	'2024-10-18 00:11:25.195676',	1,	2,	1,	1),
(2,	16,	'A new trip',	NULL,	'2024-12-01',	'2024-12-11',	3,	'2024-10-11 21:38:01.098156',	'2024-10-11 21:38:01.098156',	1,	1,	2,	1),
(1,	16,	'Trip1a',	NULL,	'2024-11-01',	'2024-12-01',	3,	'2024-10-10 10:37:23.829893',	'2024-10-10 10:37:23.829893',	3,	4,	3,	1),
(6,	16,	'A fantastic trip a!',	NULL,	'2024-10-17',	'2024-10-17',	11,	'2024-10-17 23:57:28.0255',	'2024-10-17 23:57:28.0255',	1,	1,	1,	1);

DROP TABLE IF EXISTS "at_user_payments";
DROP SEQUENCE IF EXISTS at_user_payments_id_seq;
CREATE SEQUENCE at_user_payments_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."at_user_payments" (
    "id" integer DEFAULT nextval('at_user_payments_id_seq') NOT NULL,
    "user_id" integer NOT NULL,
    "booking_id" integer NOT NULL,
    "payment_date" date NOT NULL,
    "amount" numeric(10,2) NOT NULL,
    "payment_method" character varying(255),
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "at_user_payments_pkey" PRIMARY KEY ("id")
) WITH (oids = false);


DROP TABLE IF EXISTS "et_access_level";
DROP SEQUENCE IF EXISTS et_access_level_id_seq;
CREATE SEQUENCE et_access_level_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_access_level" (
    "id" integer DEFAULT nextval('et_access_level_id_seq') NOT NULL,
    "name" character varying(45),
    "description" character varying(45),
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_access_level_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_access_level" ("id", "name", "description", "created", "modified") VALUES
(1,	'none',	NULL,	'2024-11-24 23:23:04.959786',	'2024-11-24 23:23:04.959786'),
(2,	'get',	NULL,	'2024-11-24 23:23:32.409579',	'2024-11-24 23:23:32.409579'),
(3,	'put',	NULL,	'2024-11-24 23:23:32.409579',	'2024-11-24 23:23:32.409579'),
(4,	'post',	NULL,	'2024-11-24 23:23:32.409579',	'2024-11-24 23:23:32.409579'),
(5,	'delete',	NULL,	'2024-11-24 23:23:32.409579',	'2024-11-24 23:23:32.409579');

DROP TABLE IF EXISTS "et_access_type";
DROP SEQUENCE IF EXISTS et_access_type_id_seq;
CREATE SEQUENCE et_access_type_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_access_type" (
    "id" integer DEFAULT nextval('et_access_type_id_seq') NOT NULL,
    "name" character varying(45),
    "description" character varying(45),
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_access_type_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_access_type" ("id", "name", "description", "created", "modified") VALUES
(2,	'owner',	NULL,	'2024-11-24 23:15:13.207337',	'2024-11-24 23:15:13.207337'),
(1,	'admin',	NULL,	'2024-11-24 23:14:57.462385',	'2024-11-24 23:14:57.462385');

DROP TABLE IF EXISTS "et_booking_status";
DROP SEQUENCE IF EXISTS et_booking_status_id_seq;
CREATE SEQUENCE et_booking_status_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_booking_status" (
    "id" integer DEFAULT nextval('et_booking_status_id_seq') NOT NULL,
    "status" character varying(50) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_booking_status_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_booking_status" ("id", "status", "created", "modified") VALUES
(1,	'New',	'2024-09-29 10:11:30.906822',	'2024-09-29 10:11:30.906822'),
(2,	'Cancelled',	'2024-09-30 08:29:27.323855',	'2024-09-30 08:29:27.323855'),
(3,	'Paid',	'2024-09-30 08:29:41.411134',	'2024-09-30 08:29:41.411134');

DROP TABLE IF EXISTS "et_member_status";
DROP SEQUENCE IF EXISTS et_member_status_id_seq;
CREATE SEQUENCE et_member_status_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_member_status" (
    "id" integer DEFAULT nextval('et_member_status_id_seq') NOT NULL,
    "status" character varying(255) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_member_status_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_member_status" ("id", "status", "created", "modified") VALUES
(1,	'Yes',	'2025-01-12 08:36:38.416883',	'2025-01-12 08:36:38.416883'),
(2,	'No',	'2025-01-12 08:36:12.929545',	'2025-01-12 08:36:12.929545');

DROP TABLE IF EXISTS "et_resource";
DROP SEQUENCE IF EXISTS et_resource_id_seq;
CREATE SEQUENCE et_resource_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_resource" (
    "id" integer DEFAULT nextval('et_resource_id_seq') NOT NULL,
    "name" character varying(45),
    "description" character varying(45),
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_resource_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_resource" ("id", "name", "description", "created", "modified") VALUES
(1,	'users',	NULL,	'2024-11-24 23:20:21.603342',	'2024-11-24 23:20:21.603342'),
(2,	'trips',	NULL,	'2024-11-24 23:20:40.311542',	'2024-11-24 23:20:40.311542'),
(3,	'bookings',	NULL,	'2024-11-24 23:20:48.161546',	'2024-11-24 23:20:48.161546'),
(8,	'auth',	'auth url',	'2024-12-05 07:13:21.44181',	'2024-12-05 07:13:21.44181'),
(9,	'tripStatus',	'trip Status',	'2024-12-05 07:51:55.13568',	'2024-12-05 07:51:55.13568'),
(10,	'tripDifficulty',	'trip Difficulty',	'2024-12-05 07:52:04.899949',	'2024-12-05 07:52:04.899949'),
(11,	'bookingStatus',	'Booking Status',	'2024-12-13 07:18:33.355895',	'2024-12-13 07:18:33.355895'),
(13,	'userAgeGroups',	'User Age Groups',	'2024-12-13 07:30:39.813523',	'2024-12-13 07:30:39.813523'),
(14,	'userAccountStatus',	'User Account Status',	'2024-12-13 07:30:58.923303',	'2024-12-13 07:30:58.923303'),
(16,	'tripsReport',	'trips Report (Participant Status)',	'2024-12-14 05:37:54.996572',	'2024-12-14 05:37:54.996572'),
(17,	'bookingPeople',	'Booking People',	'2024-12-20 09:12:42.840489',	'2024-12-20 09:12:42.840489'),
(15,	'userMemberStatus',	'User Member Status',	'2024-12-13 07:31:12.61181',	'2024-12-13 07:31:12.61181');

DROP TABLE IF EXISTS "et_seasons";
DROP SEQUENCE IF EXISTS et_seasons_id_seq;
CREATE SEQUENCE et_seasons_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_seasons" (
    "id" integer DEFAULT nextval('et_seasons_id_seq') NOT NULL,
    "season" character varying(255) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "start_day" integer,
    "length" integer,
    CONSTRAINT "et_seasons_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_seasons" ("id", "season", "created", "modified", "start_day", "length") VALUES
(2,	'Winter',	'2024-10-26 05:28:39.461983',	'2024-10-26 05:28:39.461983',	NULL,	NULL),
(1,	'Summer',	'2024-10-26 05:28:32.741256',	'2024-10-26 05:28:32.741256',	NULL,	NULL);

DROP TABLE IF EXISTS "et_trip_difficulty";
DROP SEQUENCE IF EXISTS et_trip_difficulty_id_seq;
CREATE SEQUENCE et_trip_difficulty_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_trip_difficulty" (
    "id" integer DEFAULT nextval('et_trip_difficulty_id_seq') NOT NULL,
    "level" character varying(50) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "level_short" character varying(3),
    "description" character varying(255),
    CONSTRAINT "et_trip_difficulty_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_trip_difficulty" ("id", "level", "created", "modified", "level_short", "description") VALUES
(1,	'Medium to Fit',	'2024-10-26 06:03:17.094307',	'2024-10-26 06:03:17.094307',	'MF',	'Up to 8 hours per day, pace faster than M, off track and above bush line travel to be expected.'),
(2,	'Easy',	'2024-10-26 06:03:53.30417',	'2024-10-26 06:03:53.30417',	'E',	'Up to 4 hours per day, pace slower than EM.'),
(3,	'Slow medium',	'2024-10-28 06:35:23.182334',	'2024-10-28 06:35:23.182334',	'SM',	'Medium trip at a slower pace than the standard pace');

DROP TABLE IF EXISTS "et_trip_status";
DROP SEQUENCE IF EXISTS et_trip_status_id_seq;
CREATE SEQUENCE et_trip_status_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_trip_status" (
    "id" integer DEFAULT nextval('et_trip_status_id_seq') NOT NULL,
    "status" character varying(50) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_trip_status_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_trip_status" ("id", "status", "created", "modified") VALUES
(1,	'New',	'2024-10-10 10:34:38.851101',	'2024-10-10 10:34:38.851101'),
(2,	'Cancelled',	'2024-10-10 10:34:49.204346',	'2024-10-10 10:34:49.204346'),
(3,	'Completed',	'2024-10-10 10:34:56.149138',	'2024-10-10 10:34:56.149138');

DROP TABLE IF EXISTS "et_trip_type";
DROP SEQUENCE IF EXISTS et_trip_types_trip_type_id_seq;
CREATE SEQUENCE et_trip_types_trip_type_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_trip_type" (
    "id" integer DEFAULT nextval('et_trip_types_trip_type_id_seq') NOT NULL,
    "type" character varying(255) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_trip_types_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_trip_type" ("id", "type", "created", "modified") VALUES
(1,	'Hiking',	'2024-10-26 05:39:39.691348',	'2024-10-26 05:39:39.691348'),
(2,	'Skiing',	'2024-10-26 05:39:45.72734',	'2024-10-26 05:39:45.72734'),
(3,	'Cycling',	'2024-10-26 05:39:52.463626',	'2024-10-26 05:39:52.463626'),
(4,	'Camping',	'2024-10-26 05:40:03.152673',	'2024-10-26 05:40:03.152673'),
(5,	'Rafting',	'2024-10-26 05:40:08.886972',	'2024-10-26 05:40:08.886972'),
(6,	'Climbing',	'2024-10-26 05:40:15.566278',	'2024-10-26 05:40:15.566278');

DROP TABLE IF EXISTS "et_user_account_status";
DROP SEQUENCE IF EXISTS et_user_account_status_id_seq;
CREATE SEQUENCE et_user_account_status_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_user_account_status" (
    "id" integer DEFAULT nextval('et_user_account_status_id_seq') NOT NULL,
    "status" character varying(255) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "description" character varying(255),
    CONSTRAINT "et_user_account_status_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_user_account_status" ("id", "status", "created", "modified", "description") VALUES
(0,	'New',	'2025-01-12 07:47:04.2678',	'2025-01-12 07:47:04.2678',	'A new account that has just been created by a user. It is not yet verified or activated. Needs to be activated by an admin.'),
(1,	'Verified',	'2024-12-01 09:36:35.177495',	'2024-12-01 09:36:35.177495',	'The email address has been verified. An Admin now needs to activate the account.'),
(2,	'Active',	'2024-12-01 09:36:50.00299',	'2024-12-01 09:36:50.00299',	'An account that has been activated, and is currently active.'),
(3,	'Disabled',	'2025-01-12 08:28:36.131138',	'2025-01-12 08:28:36.131138',	'An account that has been disabled.'),
(4,	'Reset',	'2025-01-12 08:29:16.932961',	'2025-01-12 08:29:16.932961',	'The account is flagged for a password reset. The user will be informed at the next login.');

DROP TABLE IF EXISTS "et_user_age_groups";
DROP SEQUENCE IF EXISTS et_user_age_group_id_seq;
CREATE SEQUENCE et_user_age_group_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."et_user_age_groups" (
    "id" integer DEFAULT nextval('et_user_age_group_id_seq') NOT NULL,
    "age_group" character varying(255) NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "et_user_age_group_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "et_user_age_groups" ("id", "age_group", "created", "modified") VALUES
(1,	'Infant',	'2024-10-26 07:08:21.941336',	'2024-10-26 07:08:21.941336'),
(2,	'Adult',	'2024-10-26 07:08:36.386124',	'2024-10-26 07:08:36.386124'),
(3,	'Child',	'2024-10-26 07:08:42.383391',	'2024-10-26 07:08:42.383391'),
(4,	'Youth',	'2024-10-26 07:09:03.922737',	'2024-10-26 07:09:03.922737');

DROP TABLE IF EXISTS "st_group";
DROP SEQUENCE IF EXISTS st_group_id_seq;
CREATE SEQUENCE st_group_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."st_group" (
    "id" integer DEFAULT nextval('st_group_id_seq') NOT NULL,
    "name" character varying(45),
    "description" character varying(45),
    "admin_flag" boolean DEFAULT false,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "st_group_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "st_group" ("id", "name", "description", "admin_flag", "created", "modified") VALUES
(1,	'Sys Admin',	'System admin',	't',	'2024-11-24 23:31:26.62785',	'2024-11-24 23:31:26.62785'),
(3,	'App Admin',	'Application admin',	'f',	'2024-12-05 23:46:11.129967',	'2024-12-05 23:46:11.129967'),
(2,	'User',	'app users',	'f',	'2024-12-05 06:57:30.344557',	'2024-12-05 06:57:30.344557');

DROP TABLE IF EXISTS "st_group_resource";
DROP SEQUENCE IF EXISTS st_group_resource_id_seq;
CREATE SEQUENCE st_group_resource_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."st_group_resource" (
    "id" integer DEFAULT nextval('st_group_resource_id_seq') NOT NULL,
    "group_id" integer NOT NULL,
    "resource_id" integer NOT NULL,
    "access_level_id" integer NOT NULL,
    "access_type_id" integer NOT NULL,
    "admin_flag" boolean DEFAULT false,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "st_group_resource_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "st_group_resource" ("id", "group_id", "resource_id", "access_level_id", "access_type_id", "admin_flag", "created", "modified") VALUES
(1,	1,	1,	2,	1,	'f',	'2024-11-24 23:34:35.86487',	'2024-11-24 23:34:35.86487'),
(3,	1,	1,	3,	1,	'f',	'2024-11-24 23:35:01.83461',	'2024-11-24 23:35:01.83461'),
(4,	1,	1,	4,	1,	'f',	'2024-11-24 23:35:17.568932',	'2024-11-24 23:35:17.568932'),
(5,	1,	1,	5,	1,	'f',	'2024-11-24 23:35:39.544196',	'2024-11-24 23:35:39.544196'),
(6,	1,	2,	2,	1,	'f',	'2024-11-24 23:40:19.506411',	'2024-11-24 23:40:19.506411'),
(7,	1,	2,	3,	1,	'f',	'2024-11-24 23:42:30.994064',	'2024-11-24 23:42:30.994064'),
(8,	2,	2,	2,	2,	'f',	'2024-12-05 06:59:42.806108',	'2024-12-05 06:59:42.806108'),
(12,	2,	8,	2,	2,	'f',	'2024-12-05 07:13:57.298371',	'2024-12-05 07:13:57.298371'),
(13,	2,	9,	2,	2,	'f',	'2024-12-05 07:52:39.946898',	'2024-12-05 07:52:39.946898'),
(18,	2,	3,	5,	2,	'f',	'2024-12-05 07:56:52.363959',	'2024-12-05 07:56:52.363959'),
(17,	2,	3,	4,	2,	'f',	'2024-12-05 07:56:43.626614',	'2024-12-05 07:56:43.626614'),
(16,	2,	3,	3,	2,	'f',	'2024-12-05 07:56:32.357164',	'2024-12-05 07:56:32.357164'),
(15,	2,	3,	2,	2,	'f',	'2024-12-05 07:56:06.118582',	'2024-12-05 07:56:06.118582'),
(14,	2,	10,	2,	2,	'f',	'2024-12-05 07:52:49.566355',	'2024-12-05 07:52:49.566355'),
(19,	2,	11,	2,	2,	'f',	'2024-12-13 07:26:15.856341',	'2024-12-13 07:26:15.856341'),
(20,	2,	13,	2,	2,	'f',	'2024-12-13 07:31:44.260408',	'2024-12-13 07:31:44.260408'),
(21,	2,	15,	2,	2,	'f',	'2024-12-13 07:32:38.319088',	'2024-12-13 07:32:38.319088'),
(22,	2,	14,	2,	2,	'f',	'2024-12-13 07:32:54.10676',	'2024-12-13 07:32:54.10676'),
(23,	2,	1,	2,	2,	'f',	'2024-12-13 07:51:52.949741',	'2024-12-13 07:51:52.949741'),
(24,	2,	16,	2,	2,	'f',	'2024-12-14 05:38:15.734325',	'2024-12-14 05:38:15.734325'),
(25,	2,	17,	2,	2,	'f',	'2024-12-20 09:13:11.812551',	'2024-12-20 09:13:11.812551'),
(26,	2,	17,	3,	2,	'f',	'2024-12-20 09:13:24.920667',	'2024-12-20 09:13:24.920667'),
(27,	2,	17,	4,	2,	'f',	'2024-12-20 09:13:32.890183',	'2024-12-20 09:13:32.890183'),
(28,	2,	17,	5,	2,	'f',	'2024-12-20 09:13:42.950509',	'2024-12-20 09:13:42.950509');

DROP TABLE IF EXISTS "st_token";
DROP SEQUENCE IF EXISTS st_token_id_seq;
CREATE SEQUENCE st_token_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."st_token" (
    "id" integer DEFAULT nextval('st_token_id_seq') NOT NULL,
    "user_id" integer NOT NULL,
    "name" character varying(45) NOT NULL,
    "host" character varying(45),
    "token" character varying(45),
    "valid_from" timestamptz,
    "valid_to" timestamptz,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "token_valid" boolean DEFAULT false NOT NULL,
    CONSTRAINT "st_token_pkey" PRIMARY KEY ("id")
) WITH (oids = false);


DROP TABLE IF EXISTS "st_user_group";
DROP SEQUENCE IF EXISTS st_user_group_id_seq;
CREATE SEQUENCE st_user_group_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."st_user_group" (
    "id" integer DEFAULT nextval('st_user_group_id_seq') NOT NULL,
    "user_id" integer NOT NULL,
    "group_id" integer NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "st_user_group_pkey" PRIMARY KEY ("id")
) WITH (oids = false);

INSERT INTO "st_user_group" ("id", "user_id", "group_id", "created", "modified") VALUES
(1,	16,	1,	'2024-11-24 23:31:58.500845',	'2024-11-24 23:31:58.500845'),
(2,	17,	2,	'2024-11-28 03:54:26.342778',	'2024-11-28 03:54:26.342778'),
(3,	3,	2,	'2024-12-27 03:14:27.828175',	'2024-12-27 03:14:27.828175'),
(4,	1,	2,	'2024-12-27 03:14:42.095962',	'2024-12-27 03:14:42.095962'),
(5,	2,	2,	'2024-12-27 03:14:50.895662',	'2024-12-27 03:14:50.895662'),
(6,	15,	2,	'2024-12-27 03:15:03.932908',	'2024-12-27 03:15:03.932908');

DROP TABLE IF EXISTS "st_users";
DROP SEQUENCE IF EXISTS st_users_id_seq;
CREATE SEQUENCE st_users_id_seq INCREMENT 1 MINVALUE 1 MAXVALUE 2147483647 CACHE 1;

CREATE TABLE "public"."st_users" (
    "id" integer DEFAULT nextval('st_users_id_seq') NOT NULL,
    "name" character varying(255) NOT NULL,
    "username" character varying(255) NOT NULL,
    "email" character varying(255) NOT NULL,
    "member_status_id" integer DEFAULT '0' NOT NULL,
    "created" timestamp DEFAULT CURRENT_TIMESTAMP,
    "modified" timestamp DEFAULT CURRENT_TIMESTAMP,
    "user_birth_date" date,
    "user_age_group_id" integer,
    "salt" bytea,
    "verifier" bytea,
    "user_address" character varying(255),
    "member_code" character varying(20),
    "user_password" character varying(45),
    "user_account_status_id" integer DEFAULT '0' NOT NULL,
    "user_account_hidden" boolean,
    CONSTRAINT "st_users_email_key" UNIQUE ("email"),
    CONSTRAINT "st_users_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "st_users_username_key" UNIQUE ("username")
) WITH (oids = false);

INSERT INTO "st_users" ("id", "name", "username", "email", "member_status_id", "created", "modified", "user_birth_date", "user_age_group_id", "salt", "verifier", "user_address", "member_code", "user_password", "user_account_status_id", "user_account_hidden") VALUES
(16,	'Vince2',	'vince2',	'vince.jennings2@gmail.com',	1,	'2024-11-14 10:47:04.549148',	'2025-01-14 03:16:49.450163',	NULL,	1,	'\x6e13b8cf5c83e5ac',	'\x0201163202da345a8efc7b5ed45e51f13ca1e29a266617d86ff8e7f2316ad46d064081b04939baed0174e4f01f01273800ead90b86d50aadfba5101fc6edda2d393400e5cdd369e29723a97d7c94b9a0e32c97ff84dd76f7d89a4fba3ed310dc34a49c9d4fe0ac8d2026f6e30a7ba4400f7ca2a6c95995cca8ab18aa3276dc3bee40d1e240573ab2acb91595aa51c3577ce0d93bda274029702323f30e54467b9416c0727f2f4d237812ffbaceef49c2325cc2e1c3686069d74f8bbda8799599e0283048257875f92518886497f959832e1c148839cc546aba56a8e4591653dbac879db32388a96f7fb5ab99d950883c99bf3d01df55fefa0009b2405851c99206fe9cba482931e2a42de17b62d76f500d08c16f44263434c7e762f169574035563b50c6095df438d2a9fab92c3258ccd6484795a11466d7c051ac88a26bb493fa4af4cc7d33b31864968868e0cf4c83342d6c9ed73ff87a2c9cda1a8ede587a2010d359019d7a87290023933a680b98cc0d77831bdc8760f433efcc5bcf914ac7',	'93 Farnham Street',	'1234',	NULL,	3,	't'),
(2,	'Donna Jennings12',	'donna',	'dj0040@gmail.com',	1,	'2024-10-04 09:08:52.345413',	'2025-01-14 07:10:06.688522',	NULL,	1,	NULL,	NULL,	'93 Farnham Street',	'1234B',	NULL,	1,	NULL),
(1,	'vince jennings3',	'vince',	'vince.jennings@gmail.com',	2,	'2024-09-24 07:20:41.0626',	'2025-01-14 07:10:51.970566',	NULL,	2,	NULL,	NULL,	'93 Farnham Street, Mornington',	'7654',	NULL,	1,	NULL),
(3,	'Dylan Jennings1',	'Dylan',	'dylan@dt.net.nz',	1,	'2024-10-04 09:09:11.226469',	'2025-01-14 07:11:07.063766',	NULL,	4,	NULL,	NULL,	'93 Farnham Street',	'12347',	NULL,	1,	't'),
(15,	'Vince1',	'vince1',	'vince.jennings1@gmail.com',	1,	'2024-11-14 00:12:08.297532',	'2024-12-01 09:46:07.659601',	NULL,	1,	'\x8f1a00b2822fe33d',	'\x023f5f5b535a9ca04fe4bb95373f5a673103c1f033b2af4d3c8659fcff502ffea811668bd0531f976824ef1d2dbc50eb3ca9e4704e33601e081f621fd0c075d7cdd5fe49fb55ec672ee7773697dfc4e51b2682d5c349ef8368daaec799b07d62aa720eda12c198e2fcca6b860e304b1552bab7810a04fcc1e5d8e09ad61a67ae9711ed8df454347ec724a010d535723d319fda04b21747cfd1accf66efa4d9db969751c53600d58093b5b63dbc3fabadfa8d01b47077112d0039d2d162452371c77d6f7b61f9585d180109dc2ce8f0aca5d0e47cc393889e52f450678afd00de5cc691a20c920a9f9e603147b6485c2572d1f528ce7f31fb0bd634ed3359b7f5505fcc55bd6180d4877f1f08dc9da0faf7d7353b494c493d1e0f0ba3698fd8f7ab2a301e08acd9cfb4aef8e9d61ef136a91bc0de504f7f54d9b82a3498b991c5f34c79466b955e200a0ddbf66c6eaa769f4620b3fd3d5a3beda7297f039026a8197601e6fa8ca325382a26537c46b2d34569c053238df4889964d0f013b5fbb0e5',	'93 Farnham Street',	'1',	NULL,	2,	NULL),
(17,	'Vince3',	'vince3',	'vince.jennings3@gmail.com',	1,	'2024-11-25 03:26:18.444765',	'2025-01-14 07:11:43.894139',	NULL,	3,	'\x9eb1c2a76444a9e3',	'\x025105d39fdf717ae3f733502ca3021cdaded71e783c4d49aab09c630597f17688f9a4247bd362e2201a2c1b97f7bce2a2702afd6eff379571314c6d19426ef9f6fd1bafbe2083c5af420110ba5d7c1749d412fc95401570f0ff5e44cb23ad7fbbf7308ca882797ff5f749052a05489a599d95919d7a59ba3f2a1e99f32a067c34f947e012b65887dcc066f3cf47dfec7c4c2328bebd2e32afdfe52367a2036161e860c5b54aa70f83c271f81fc178757a1b2705657ac5bb7be79e0ca6c26733a4927602787e71850f7899a1749a9e40818d09994ecd0f60a16c03efce3fc78aaba1f06d5557eab664fc772ebcbeb315fce5bb94ca972c65ab01676784c7d2c8e3d5fbc2941209e37878f47132db8348f67a49d613dde45c57632c1a2dbb199d25b008025c543fe9cca7de85932311caa476347cf58b5b42f76dfbe836848fe5d7a9e4bb1522ea3afa9f8e6f6ef010d3a5e6be154d0b0693e2d335eceb8658d8826c153c87e4805e8bad85bc2c5547b35fab0490b5a7141c5317998ccfc06496cc',	'93 Farnham Street',	'1234A',	NULL,	3,	NULL);

DELIMITER ;;

--CREATE TRIGGER "update_st_users_modified" BEFORE UPDATE ON "public"."st_users" FOR EACH ROW EXECUTE FUNCTION update_modified_column();;

DELIMITER ;

DROP VIEW IF EXISTS "vt_trips";
CREATE TABLE "vt_trips" ("id" integer, "owner_id" integer, "trip_name" text, "location" text, "from_date" date, "to_date" date, "max_participants" integer, "created" timestamp, "modified" timestamp, "trip_status_id" integer, "trip_type_id" integer, "difficulty_level_id" integer);


ALTER TABLE ONLY "public"."at_booking_people" ADD CONSTRAINT "booking_id_fkey" FOREIGN KEY (booking_id) REFERENCES at_bookings(id) NOT VALID NOT DEFERRABLE;
ALTER TABLE ONLY "public"."at_booking_people" ADD CONSTRAINT "user_id_fkey" FOREIGN KEY (person_id) REFERENCES st_users(id) NOT VALID NOT DEFERRABLE;

ALTER TABLE ONLY "public"."at_bookings" ADD CONSTRAINT "bookings_status_id_fkey" FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id) NOT VALID NOT DEFERRABLE;

DROP TABLE IF EXISTS "vt_trips";
CREATE VIEW "vt_trips" AS SELECT at_trips.id,
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
   FROM at_trips;

-- 2025-02-18 08:21:25.996253+00



-- User the following to reset the sequences to the highest id in the table.

--SELECT nextval('et_resource_id_seq'::regclass);
--SELECT setval('et_resource_id_seq'::regclass, (SELECT MAX(id) FROM et_resource));


CREATE OR REPLACE FUNCTION vj_execute_multiple_queries(queries text[])
RETURNS void AS $$
DECLARE
    query text;
BEGIN
    FOREACH query IN ARRAY queries
    LOOP
        EXECUTE query;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

WITH query_list AS (
	SELECT array_agg(format('SELECT setval(' || substring(column_default, 'nextval\((.*)\)') || ', (SELECT MAX(' || quote_ident(column_name) || ') FROM ' || quote_ident(table_name) || '));')) as queries
	--SELECT format('SELECT setval(' || substring(column_default, 'nextval\((.*)\)') || ', (SELECT MAX(' || quote_ident(column_name) || ') FROM ' || quote_ident(table_name) || '));') as queries
	FROM information_schema.columns
	WHERE column_default LIKE 'nextval%'

	--SELECT array_agg(format('SELECT ' || column_default || ', (SELECT MAX(' || quote_ident(column_name) || ') FROM ' || quote_ident(table_name) || ');')) as queries
	--SELECT format('SELECT ' || column_default || ', (SELECT MAX(' || quote_ident(column_name) || ') FROM ' || quote_ident(table_name) || ');') as queries
	--FROM information_schema.columns
	--WHERE column_default LIKE 'nextval%'
)
--SELECT vj_execute_multiple_queries(queries)
SELECT queries
FROM query_list;
