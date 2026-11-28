-- Fresh installation schema through 0039_security_admin_lockout.sql.
-- Generated from disposable migrations and reviewed installation grants.
SELECT set_config('search_path', format('%I, pg_temp', current_schema()), true);
SET LOCAL check_function_bodies = false;
--
-- PostgreSQL database dump
--




--
-- Name: tm_install_rehearsal; Type: SCHEMA; Schema: -; Owner: -
--



--
-- Name: advance_account_version(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION advance_account_version() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN NEW.version:=OLD.version+1;RETURN NEW;END $$;


--
-- Name: advance_resource_version(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION advance_resource_version() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN NEW.version:=OLD.version+1;RETURN NEW;END $$;


--
-- Name: assign_member_code(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION assign_member_code() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE candidate text;attempt integer;
BEGIN
 IF NULLIF(btrim(NEW.member_code),'') IS NOT NULL THEN NEW.member_code:=upper(btrim(NEW.member_code));RETURN NEW;END IF;
 FOR attempt IN 0..9 LOOP
  candidate:='TM-'||to_char(CURRENT_DATE,'YYYYMM')||'-'||lpad(to_hex(NEW.id),GREATEST(5,length(to_hex(NEW.id))),'0');
  IF attempt>0 THEN candidate:=candidate||'-'||attempt::text;END IF;
  IF NOT EXISTS(SELECT 1 FROM st_users WHERE lower(btrim(member_code))=lower(candidate)) THEN NEW.member_code:=candidate;RETURN NEW;END IF;
 END LOOP;
 RAISE EXCEPTION 'member code generation exhausted' USING ERRCODE='23505';
END $$;


--
-- Name: enforce_security_admin_guard(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION enforce_security_admin_guard() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
DECLARE
    was_armed boolean;
BEGIN
    -- Serialize final checks. Write skews at stricter isolation levels fail
    -- with a serialization error because every check writes this row.
    UPDATE st_security_admin_guard SET armed = armed WHERE singleton
    RETURNING armed INTO STRICT was_armed;

    IF has_effective_security_admin() THEN
        IF NOT was_armed THEN
            UPDATE st_security_admin_guard SET armed = true WHERE singleton;
        END IF;
    ELSIF was_armed THEN
        RAISE EXCEPTION 'At least one active security administrator is required'
            USING ERRCODE = '23514', CONSTRAINT = 'effective_security_admin_required';
    END IF;
    RETURN NULL;
END;
$$;


--
-- Name: guard_trip_capacity(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION guard_trip_capacity() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE occupied bigint;
BEGIN
 SELECT COALESCE(sum(CASE WHEN b.booking_status_id IN (3,4) THEN
  (SELECT count(*) FROM at_booking_people p WHERE p.booking_id=b.id)
 ELSE COALESCE((SELECT r.seats FROM at_checkout_reservations r WHERE r.booking_id=b.id AND r.expires_at>CURRENT_TIMESTAMP),0) END),0)
 INTO occupied FROM at_bookings b WHERE b.trip_id=OLD.id AND b.booking_status_id<>2;
 IF occupied>0 AND (NEW.max_participants IS NULL OR NEW.max_participants<occupied) THEN
  RAISE EXCEPTION 'capacity cannot be less than confirmed or reserved occupancy' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;


--
-- Name: has_effective_security_admin(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION has_effective_security_admin() RETURNS boolean
    LANGUAGE sql STABLE
    SET search_path FROM CURRENT
    AS $$
    SELECT EXISTS (
        SELECT u.id
        FROM st_users u
        JOIN st_user_group m ON m.user_id = u.id
        JOIN st_group_resource g ON g.group_id = m.group_id
        JOIN et_resource r ON r.id = g.resource_id
        JOIN et_access_level a ON a.id = g.access_level_id
        JOIN et_access_scope s ON s.id = g.access_scope_id
        WHERE u.user_account_status_id = 2
          AND lower(s.name) = 'any'
          AND lower(r.name) IN ('securitygroup', 'securitygroupresource', 'securityusergroup')
          AND lower(a.name) IN ('get', 'post', 'put', 'delete')
        GROUP BY u.id
        HAVING count(DISTINCT (lower(r.name), lower(a.name))) = 12
    );
$$;


--
-- Name: protect_checkout_inputs(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION protect_checkout_inputs() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
DECLARE booking integer;
BEGIN
 IF TG_TABLE_NAME='at_bookings' THEN
  IF (NEW.trip_id,NEW.owner_id,NEW.from_date,NEW.to_date) IS NOT DISTINCT FROM (OLD.trip_id,OLD.owner_id,OLD.from_date,OLD.to_date) THEN
   IF NEW.booking_price IS NOT DISTINCT FROM OLD.booking_price THEN RETURN NEW; END IF;
   -- Only attachment of the stored provider quote may write its snapshot price.
   IF EXISTS(SELECT 1 FROM at_checkout_attempts a WHERE a.booking_id=OLD.id AND a.state IN ('attached','paid') AND a.session_id=NEW.stripe_session_id AND NEW.booking_price=a.amount_minor::numeric/100) THEN RETURN NEW; END IF;
  END IF;
  booking:=OLD.id;
 ELSE
  IF TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.booking_id AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
  IF TG_OP='DELETE' THEN booking:=OLD.booking_id; ELSE booking:=NEW.booking_id; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=booking AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;


--
-- Name: protect_checkout_voucher(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION protect_checkout_voucher() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
 IF NEW.voucher_code IS DISTINCT FROM OLD.voucher_code AND EXISTS (
   SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.id AND state<>'expired'
 ) THEN
   RAISE EXCEPTION 'checkout voucher is frozen' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;


--
-- Name: protect_financial_audit(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION protect_financial_audit() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN RAISE EXCEPTION 'financial audit history is immutable' USING ERRCODE='42501'; END $$;


--
-- Name: protect_group_system_key(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION protect_group_system_key() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.system_key IS NOT NULL THEN
            RAISE EXCEPTION 'built-in security groups cannot be deleted' USING ERRCODE = '23514';
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.system_key IS NOT NULL THEN
            RAISE EXCEPTION 'system keys are assigned by migrations' USING ERRCODE = '23514';
        END IF;
    ELSIF NEW.system_key IS DISTINCT FROM OLD.system_key THEN
        RAISE EXCEPTION 'group system keys cannot change' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: protect_provider_ledger(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION protect_provider_ledger() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
 IF TG_TABLE_NAME='at_payments' THEN
  IF OLD.stripe_session_id IS NOT NULL THEN RAISE EXCEPTION 'provider payment is immutable' USING ERRCODE='23514'; END IF;
 ELSE
  IF OLD.provider_refund_id IS NOT NULL THEN RAISE EXCEPTION 'provider refund is immutable' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;RETURN NEW;
END $$;


--
-- Name: queue_account_review(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION queue_account_review() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
BEGIN
 INSERT INTO st_account_review_requests(user_id) VALUES(NEW.id) ON CONFLICT DO NOTHING;
 RETURN NEW;
END $$;


--
-- Name: record_account_change(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION record_account_change() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
DECLARE previous jsonb; following jsonb; identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN identity:=OLD.id;previous:=jsonb_build_object('username',OLD.username,'email',OLD.email,'status',OLD.user_account_status_id); END IF;
 IF TG_OP<>'DELETE' THEN identity:=NEW.id;following:=jsonb_build_object('username',NEW.username,'email',NEW.email,'status',NEW.user_account_status_id); END IF;
 IF TG_OP='UPDATE' AND previous=following THEN RETURN NEW; END IF;
 INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),'account',identity,TG_OP,previous,COALESCE(following,'{}'::jsonb)||jsonb_build_object('reason',NULLIF(current_setting('tm.change_reason',true),'')));
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;


--
-- Name: record_email_change(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION record_email_change() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
BEGIN
 IF OLD.email IS DISTINCT FROM NEW.email THEN
  INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
  VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),'profile_email',NEW.id,'UPDATE',jsonb_build_object('email',OLD.email),jsonb_build_object('email',NEW.email));
 END IF;
 RETURN NEW;
END $$;


--
-- Name: record_financial_change(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION record_financial_change() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
DECLARE previous jsonb; following jsonb; identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN previous:=to_jsonb(OLD);identity:=OLD.id; END IF;
 IF TG_OP<>'DELETE' THEN following:=to_jsonb(NEW);identity:=NEW.id; END IF;
 INSERT INTO at_financial_audit(actor,reason,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),
 COALESCE(NULLIF(current_setting('tm.reason',true),''),'direct database change'),TG_TABLE_NAME,identity,TG_OP,previous,following);
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;


--
-- Name: record_security_change(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION record_security_change() RETURNS trigger
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
DECLARE previous jsonb;following jsonb;identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN previous:=to_jsonb(OLD);identity:=OLD.id;END IF;
 IF TG_OP<>'DELETE' THEN following:=to_jsonb(NEW);identity:=NEW.id;END IF;
 INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),TG_TABLE_NAME,identity,TG_OP,previous,following);
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END $$;


--
-- Name: redact_notification_payloads(integer, integer, integer); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION redact_notification_payloads(delivered_days integer, orphan_days integer, batch_limit integer) RETURNS bigint
    LANGUAGE plpgsql SECURITY DEFINER
    SET search_path FROM CURRENT
    AS $$
DECLARE affected bigint;
BEGIN
 IF delivered_days NOT BETWEEN 1 AND 3650 OR orphan_days NOT BETWEEN 1 AND 3650 OR batch_limit NOT BETWEEN 1 AND 500 THEN RAISE EXCEPTION 'invalid retention bounds' USING ERRCODE='22023'; END IF;
 WITH expired AS(SELECT id FROM at_notification_outbox WHERE recipient<>'' AND
 ((sent_at IS NOT NULL AND sent_at<CURRENT_TIMESTAMP-make_interval(days=>delivered_days)) OR
 (booking_id IS NULL AND account_id IS NULL AND created<CURRENT_TIMESTAMP-make_interval(days=>orphan_days)))
 ORDER BY id FOR UPDATE SKIP LOCKED LIMIT batch_limit)
 UPDATE at_notification_outbox n SET recipient='',subject='',body='' FROM expired e WHERE n.id=e.id;
 GET DIAGNOSTICS affected=ROW_COUNT;RETURN affected;
END $$;


--
-- Name: update_modified_column(); Type: FUNCTION; Schema: current; Owner: -
--

CREATE FUNCTION update_modified_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN NEW.modified=CURRENT_TIMESTAMP;RETURN NEW;END $$;




--
-- Name: at_booking_people; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_booking_people (
    id integer NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    booking_id integer DEFAULT 0 NOT NULL,
    person_id integer DEFAULT 0 NOT NULL,
    notes text,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    submitted_name text,
    mobile_phone text,
    home_address text,
    membership_category text,
    emergency_contact_name text,
    emergency_contact_phone text,
    dietary_requirements text,
    questionnaire_revision integer,
    trip_answers jsonb DEFAULT '{}'::jsonb NOT NULL,
    trip_answer_history jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT at_booking_people_membership_category_check CHECK ((membership_category = ANY (ARRAY['member'::text, 'non_member'::text, 'affiliated_member'::text]))),
    CONSTRAINT at_booking_people_questionnaire_revision_check CHECK ((questionnaire_revision > 0)),
    CONSTRAINT at_booking_people_trip_answer_history_check CHECK ((jsonb_typeof(trip_answer_history) = 'object'::text)),
    CONSTRAINT at_booking_people_trip_answers_check CHECK ((jsonb_typeof(trip_answers) = 'object'::text))
);


--
-- Name: at_booking_people_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_booking_people ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_booking_people_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_booking_vouchers; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_booking_vouchers (
    id integer NOT NULL,
    code text NOT NULL,
    discount_percent numeric,
    fixed_cost numeric,
    expiry_date date,
    is_active boolean DEFAULT true,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL,
    CONSTRAINT at_booking_vouchers_discount_percent_check CHECK (((discount_percent >= (0)::numeric) AND (discount_percent <= (100)::numeric))),
    CONSTRAINT at_booking_vouchers_fixed_cost_check CHECK ((fixed_cost >= (0)::numeric))
);


--
-- Name: at_booking_vouchers_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_booking_vouchers ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_booking_vouchers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_bookings; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_bookings (
    id integer NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    notes text,
    from_date timestamp without time zone,
    to_date timestamp without time zone,
    booking_status_id integer DEFAULT 1 NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    trip_id integer DEFAULT 0 NOT NULL,
    booking_date date,
    group_booking_id integer,
    booking_price numeric(8,2),
    payment_date date,
    amount_paid bigint,
    currency character(3),
    stripe_session_id character varying(100),
    voucher_code text DEFAULT ''::text NOT NULL,
    free_confirmed_at timestamp with time zone,
    requires_participant_details boolean DEFAULT true NOT NULL,
    CONSTRAINT at_bookings_booking_price_check CHECK ((booking_price >= (0)::numeric)),
    CONSTRAINT at_bookings_check CHECK ((to_date > from_date)),
    CONSTRAINT booking_free_confirmation_price CHECK (((free_confirmed_at IS NULL) OR ((booking_price IS NOT NULL) AND (booking_price = (0)::numeric)))),
    CONSTRAINT booking_voucher_code_length CHECK ((length(voucher_code) <= 64))
);


--
-- Name: at_bookings_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_bookings ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_bookings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_checkout_attempts; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_checkout_attempts (
    id text NOT NULL,
    booking_id integer NOT NULL,
    amount_minor bigint NOT NULL,
    email text NOT NULL,
    title text NOT NULL,
    description text NOT NULL,
    domain text NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    session_id text,
    checkout_url text,
    state text DEFAULT 'pending'::text NOT NULL,
    reconcile_after timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reconcile_failures integer DEFAULT 0 NOT NULL,
    voucher_id integer,
    voucher_cutoff timestamp with time zone,
    CONSTRAINT at_checkout_attempts_amount_minor_check CHECK ((amount_minor > 0)),
    CONSTRAINT checkout_voucher_snapshot CHECK (((voucher_id IS NULL) = (voucher_cutoff IS NULL)))
);


--
-- Name: at_checkout_quotes; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_checkout_quotes (
    session_id text NOT NULL,
    booking_id integer NOT NULL,
    amount_minor bigint NOT NULL,
    currency text NOT NULL,
    CONSTRAINT at_checkout_quotes_amount_minor_check CHECK ((amount_minor > 0)),
    CONSTRAINT at_checkout_quotes_currency_check CHECK (((currency = lower(currency)) AND (length(currency) = 3)))
);


--
-- Name: at_checkout_reconciliations; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_checkout_reconciliations (
    id bigint NOT NULL,
    attempt_id text NOT NULL,
    actor_id integer NOT NULL,
    outcome text NOT NULL,
    reconciled_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: at_checkout_reconciliations_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

CREATE SEQUENCE at_checkout_reconciliations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: at_checkout_reconciliations_id_seq; Type: SEQUENCE OWNED BY; Schema: current; Owner: -
--

ALTER SEQUENCE at_checkout_reconciliations_id_seq OWNED BY at_checkout_reconciliations.id;


--
-- Name: at_checkout_reservations; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_checkout_reservations (
    booking_id integer NOT NULL,
    seats integer NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    CONSTRAINT at_checkout_reservations_seats_check CHECK ((seats > 0))
);


--
-- Name: at_financial_audit; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_financial_audit (
    id bigint NOT NULL,
    recorded_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    actor text NOT NULL,
    reason text NOT NULL,
    resource text NOT NULL,
    record_id integer NOT NULL,
    operation text NOT NULL,
    before_data jsonb,
    after_data jsonb,
    request_id text DEFAULT NULLIF(current_setting('tm.request_id'::text, true), ''::text)
);


--
-- Name: at_financial_audit_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_financial_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME at_financial_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_group_bookings; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_group_bookings (
    id integer NOT NULL,
    group_name character varying(255) NOT NULL,
    owner_id integer DEFAULT 0 NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL
);


--
-- Name: at_group_bookings_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_group_bookings ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_group_bookings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_notification_outbox; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_notification_outbox (
    id bigint NOT NULL,
    booking_id integer,
    kind text NOT NULL,
    recipient text NOT NULL,
    subject text NOT NULL,
    body text NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    available_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    sent_at timestamp with time zone,
    account_id integer
);


--
-- Name: at_notification_outbox_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_notification_outbox ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_notification_outbox_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_payment_reviews; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_payment_reviews (
    session_id text NOT NULL,
    booking_id integer NOT NULL,
    amount_minor bigint NOT NULL,
    currency text NOT NULL,
    reason text NOT NULL,
    first_seen timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    last_seen timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    failures integer DEFAULT 1 NOT NULL,
    resolved_at timestamp with time zone,
    resolution text,
    actor text,
    evidence text,
    CONSTRAINT at_payment_reviews_amount_minor_check CHECK ((amount_minor > 0))
);


--
-- Name: at_payments; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_payments (
    id integer NOT NULL,
    booking_id integer NOT NULL,
    payment_date date NOT NULL,
    amount numeric(10,2) NOT NULL,
    payment_method character varying(255),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    stripe_session_id text,
    CONSTRAINT at_payments_amount_check CHECK ((amount >= (0)::numeric))
);


--
-- Name: at_payments_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_payments ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_payments_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_provider_refunds; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_provider_refunds (
    id text NOT NULL,
    payment_id integer NOT NULL,
    actor text NOT NULL,
    amount_minor bigint,
    status text NOT NULL,
    checked_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reconcile_after timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    reconcile_failures integer DEFAULT 0 NOT NULL,
    first_seen timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT at_provider_refunds_amount_minor_check CHECK ((amount_minor > 0)),
    CONSTRAINT at_provider_refunds_status_check CHECK ((status = ANY (ARRAY['pending'::text, 'requires_action'::text, 'succeeded'::text, 'failed'::text, 'canceled'::text])))
);


--
-- Name: at_refunds; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_refunds (
    id integer NOT NULL,
    payment_id integer NOT NULL,
    refund_date timestamp with time zone NOT NULL,
    amount numeric(10,2) NOT NULL,
    reason text,
    external_ref text,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    provider_refund_id text,
    CONSTRAINT at_refunds_amount_check CHECK ((amount >= (0)::numeric))
);


--
-- Name: at_refunds_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_refunds ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_refunds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_security_audit; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_security_audit (
    id bigint NOT NULL,
    recorded_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    actor text NOT NULL,
    request_id text,
    resource text NOT NULL,
    record_id integer NOT NULL,
    operation text NOT NULL,
    before_data jsonb,
    after_data jsonb
);


--
-- Name: at_security_audit_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_security_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME at_security_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_stripe_events; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_stripe_events (
    event_id text NOT NULL,
    session_id text NOT NULL,
    processed_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: at_trip_cost_groups; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_trip_cost_groups (
    id integer NOT NULL,
    description character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL
);


--
-- Name: at_trip_cost_groups_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_trip_cost_groups ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_trip_cost_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_trip_costs; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_trip_costs (
    id integer NOT NULL,
    trip_cost_group_id integer NOT NULL,
    description character varying(50),
    member_status_id integer NOT NULL,
    user_age_group_id integer NOT NULL,
    season_id integer NOT NULL,
    amount numeric(10,2) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL,
    CONSTRAINT at_trip_costs_amount_check CHECK ((amount >= (0)::numeric))
);


--
-- Name: at_trip_costs_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_trip_costs ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_trip_costs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_trip_questionnaires; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_trip_questionnaires (
    trip_id integer NOT NULL,
    revision integer NOT NULL,
    questions jsonb NOT NULL,
    CONSTRAINT at_trip_questionnaires_questions_check CHECK ((jsonb_typeof(questions) = 'array'::text)),
    CONSTRAINT at_trip_questionnaires_revision_check CHECK ((revision > 0))
);


--
-- Name: at_trips; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_trips (
    id integer NOT NULL,
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
    description text,
    version bigint DEFAULT 1 NOT NULL,
    CONSTRAINT at_trips_check CHECK ((to_date >= from_date)),
    CONSTRAINT at_trips_max_participants_check CHECK ((max_participants >= 0))
);


--
-- Name: at_trips_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE at_trips ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME at_trips_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_voucher_redemptions; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE at_voucher_redemptions (
    booking_id integer NOT NULL,
    voucher_id integer NOT NULL,
    voucher_code text NOT NULL,
    source text NOT NULL,
    reference text NOT NULL,
    used_at timestamp with time zone NOT NULL,
    amount_minor bigint NOT NULL,
    CONSTRAINT at_voucher_redemptions_amount_minor_check CHECK ((amount_minor >= 0)),
    CONSTRAINT at_voucher_redemptions_check CHECK (((source = 'free'::text) = (amount_minor = 0))),
    CONSTRAINT at_voucher_redemptions_reference_check CHECK ((reference <> ''::text)),
    CONSTRAINT at_voucher_redemptions_source_check CHECK ((source = ANY (ARRAY['manual'::text, 'stripe'::text, 'free'::text]))),
    CONSTRAINT at_voucher_redemptions_voucher_code_check CHECK ((voucher_code <> ''::text)),
    CONSTRAINT at_voucher_redemptions_voucher_id_check CHECK ((voucher_id > 0))
);


--
-- Name: et_access_level; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_access_level (
    id integer NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_access_level_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_access_level ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_access_level_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_access_scope; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_access_scope (
    id integer NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_access_scope_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_access_scope ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_access_scope_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_booking_status; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_booking_status (
    id integer NOT NULL,
    status character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_booking_status_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_booking_status ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_booking_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_member_status; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_member_status (
    id integer NOT NULL,
    status character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_member_status_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_member_status ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_member_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_resource; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_resource (
    id integer NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_resource_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_resource ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_resource_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_seasons; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_seasons (
    id integer NOT NULL,
    season character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    start_day integer,
    length integer
);


--
-- Name: et_seasons_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_seasons ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_seasons_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_trip_difficulty; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_trip_difficulty (
    id integer NOT NULL,
    level character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    level_short character varying(3),
    description character varying(255)
);


--
-- Name: et_trip_difficulty_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_trip_difficulty ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_trip_difficulty_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_trip_status; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_trip_status (
    id integer NOT NULL,
    status character varying(50) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_trip_status_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_trip_status ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_trip_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_trip_type; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_trip_type (
    id integer NOT NULL,
    type character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    default_questions jsonb DEFAULT '[{"key": "need_transport", "type": "yes_no", "label": "Do you need transport?", "required": true}, {"key": "offer_transport", "type": "yes_no", "label": "Can you offer transport?", "required": true}, {"key": "transport_details", "type": "text", "label": "Transport details, including one-way arrangements or seats available", "required": false}, {"key": "stayed_before", "type": "yes_no", "label": "Have you stayed at the lodge or hut before?", "required": false}]'::jsonb NOT NULL,
    CONSTRAINT et_trip_type_default_questions_check CHECK ((jsonb_typeof(default_questions) = 'array'::text))
);


--
-- Name: et_trip_type_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_trip_type ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_trip_type_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_user_account_status; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_user_account_status (
    id integer NOT NULL,
    status character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    description character varying(255)
);


--
-- Name: et_user_account_status_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_user_account_status ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_user_account_status_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: et_user_age_groups; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE et_user_age_groups (
    id integer NOT NULL,
    age_group character varying(255) NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: et_user_age_groups_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE et_user_age_groups ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME et_user_age_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_account_review_requests; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_account_review_requests (
    user_id integer NOT NULL,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    dispatched_at timestamp with time zone
);


--
-- Name: st_appearance_revision; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_appearance_revision (
    version bigint NOT NULL,
    settings jsonb NOT NULL,
    published_by integer,
    published_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    CONSTRAINT st_appearance_revision_version_check CHECK ((version > 0))
);


--
-- Name: st_auth_rate_limits; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_auth_rate_limits (
    key_hash text NOT NULL,
    window_start timestamp with time zone NOT NULL,
    attempts integer NOT NULL
);


--
-- Name: st_email_changes; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_email_changes (
    user_id integer NOT NULL,
    old_email text NOT NULL,
    new_email text NOT NULL,
    old_digest bytea NOT NULL,
    new_digest bytea NOT NULL,
    created timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    expires_at timestamp with time zone DEFAULT (CURRENT_TIMESTAMP + '00:15:00'::interval) NOT NULL,
    attempts integer DEFAULT 0 NOT NULL,
    CONSTRAINT st_email_changes_attempts_check CHECK (((attempts >= 0) AND (attempts <= 5)))
);


--
-- Name: st_group; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_group (
    id integer NOT NULL,
    name character varying(45),
    description character varying(45),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    role text,
    version bigint DEFAULT 1 NOT NULL,
    system_key text
);


--
-- Name: st_group_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_group ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_group_resource; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_group_resource (
    id integer NOT NULL,
    group_id integer NOT NULL,
    resource_id integer NOT NULL,
    access_level_id integer NOT NULL,
    access_scope_id integer NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL
);


--
-- Name: st_group_resource_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_group_resource ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_group_resource_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_security_admin_guard; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_security_admin_guard (
    singleton boolean NOT NULL,
    armed boolean NOT NULL,
    CONSTRAINT st_security_admin_guard_singleton_check CHECK (singleton)
);


--
-- Name: st_token; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_token (
    id integer NOT NULL,
    user_id integer NOT NULL,
    name character varying(45) NOT NULL,
    host character varying(45),
    token text,
    valid_from timestamp with time zone,
    valid_to timestamp with time zone,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    token_valid boolean DEFAULT false NOT NULL,
    session_data json
);


--
-- Name: st_token_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_token ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_user_credentials; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_user_credentials (
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


--
-- Name: st_user_credentials_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_user_credentials ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_user_credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_user_group; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_user_group (
    id integer NOT NULL,
    user_id integer NOT NULL,
    group_id integer NOT NULL,
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    version bigint DEFAULT 1 NOT NULL
);


--
-- Name: st_user_group_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_user_group ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_user_group_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: st_users; Type: TABLE; Schema: current; Owner: -
--

CREATE TABLE st_users (
    id integer NOT NULL,
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
    provider_id character varying(255),
    version bigint DEFAULT 1 NOT NULL
);


--
-- Name: st_users_id_seq; Type: SEQUENCE; Schema: current; Owner: -
--

ALTER TABLE st_users ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME st_users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: at_checkout_reconciliations id; Type: DEFAULT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reconciliations ALTER COLUMN id SET DEFAULT nextval('at_checkout_reconciliations_id_seq'::regclass);


--
-- PostgreSQL database dump complete
--


--
-- PostgreSQL database dump
--




--
-- Data for Name: et_access_level; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_access_level (id, name, description, created, modified) VALUES (1, 'GET', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_access_level (id, name, description, created, modified) VALUES (2, 'POST', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_access_level (id, name, description, created, modified) VALUES (3, 'PUT', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_access_level (id, name, description, created, modified) VALUES (4, 'DELETE', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: et_access_scope; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_access_scope (id, name, description, created, modified) VALUES (1, 'own', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_access_scope (id, name, description, created, modified) VALUES (2, 'any', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: et_booking_status; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_booking_status (id, status, created, modified) VALUES (1, 'Not paid', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_booking_status (id, status, created, modified) VALUES (2, 'Cancelled', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_booking_status (id, status, created, modified) VALUES (3, 'Paid', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_booking_status (id, status, created, modified) VALUES (4, 'Partially paid', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: et_member_status; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_member_status (id, status, created, modified) VALUES (0, 'Unspecified', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_member_status (id, status, created, modified) VALUES (1, 'Member', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: et_resource; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_resource (id, name, description, created, modified) VALUES (1, 'auth', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (2, 'myBookings', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (3, 'bookings', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (4, 'bookingPeople', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (5, 'trips', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (6, 'tripsReport', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (7, 'users', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (8, 'userAgeGroups', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (9, 'payments', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (10, 'refunds', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (11, 'userMemberStatus', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (12, 'userAccountStatus', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (13, 'groupBooking', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (14, 'bookingStatus', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (15, 'bookingVouchers', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (16, 'tripType', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (17, 'tripStatus', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (18, 'tripDifficulty', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (19, 'tripCosts', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (20, 'tripCostGroups', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (21, 'seasons', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (22, 'securityResource', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (23, 'securityGroup', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (24, 'securityGroupResource', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (25, 'securityUserGroup', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (26, 'securityAccessLevel', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (27, 'securityAccessScope', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (28, 'gmailcert', NULL, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (29, 'myProfile', 'Read and edit own profile', '2026-11-28 04:34:19.490965', '2026-11-28 04:34:19.490965');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (30, 'operations', NULL, '2026-11-28 04:34:19.634443', '2026-11-28 04:34:19.634443');
INSERT INTO et_resource (id, name, description, created, modified) VALUES (31, 'siteAppearance', NULL, '2026-11-28 04:34:19.897692', '2026-11-28 04:34:19.897692');


--
-- Data for Name: et_seasons; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_seasons (id, season, created, modified, start_day, length) VALUES (1, 'All year', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1, 365);


--
-- Data for Name: et_trip_difficulty; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_trip_difficulty (id, level, created, modified, level_short, description) VALUES (1, 'Easy', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 'E', 'Easy trip');


--
-- Data for Name: et_trip_status; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_trip_status (id, status, created, modified) VALUES (1, 'Planned', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_trip_status (id, status, created, modified) VALUES (2, 'Open', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_trip_status (id, status, created, modified) VALUES (3, 'Closed', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: et_trip_type; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_trip_type (id, type, created, modified, default_questions) VALUES (1, 'Trip', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', '[{"key": "need_transport", "type": "yes_no", "label": "Do you need transport?", "required": true}, {"key": "offer_transport", "type": "yes_no", "label": "Can you offer transport?", "required": true}, {"key": "transport_details", "type": "text", "label": "Transport details, including one-way arrangements or seats available", "required": false}, {"key": "stayed_before", "type": "yes_no", "label": "Have you stayed at the lodge or hut before?", "required": false}]');


--
-- Data for Name: et_user_account_status; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (0, 'New', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (1, 'Verified', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (2, 'Active', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (3, 'Disabled', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (4, 'Password reset required', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (5, 'For deletion', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);
INSERT INTO et_user_account_status (id, status, created, modified, description) VALUES (6, 'Passkey reset required', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', NULL);


--
-- Data for Name: et_user_age_groups; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (1, 'adult', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (2, 'youth', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (3, 'child', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (4, 'senior', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (5, 'infant', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');
INSERT INTO et_user_age_groups (id, age_group, created, modified) VALUES (6, 'toddler', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811');


--
-- Data for Name: st_group; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO st_group (id, name, description, created, modified, role, version, system_key) VALUES (1, 'User', 'Member access', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.911077', NULL, 2, 'user');
INSERT INTO st_group (id, name, description, created, modified, role, version, system_key) VALUES (2, 'Admin', 'Trip administration', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.911077', NULL, 2, 'admin');
INSERT INTO st_group (id, name, description, created, modified, role, version, system_key) VALUES (3, 'SysAdmin', 'Security administration', '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.911077', NULL, 2, 'sysadmin');


--
-- Data for Name: st_group_resource; Type: TABLE DATA; Schema: current; Owner: -
--

INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (197, 1, 1, 1, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (198, 1, 2, 1, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (199, 1, 3, 1, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (200, 1, 4, 1, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (201, 1, 2, 2, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (202, 1, 3, 2, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (203, 1, 4, 2, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (204, 1, 2, 3, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (205, 1, 3, 3, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (206, 1, 4, 3, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (207, 1, 2, 4, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (208, 1, 3, 4, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (209, 1, 4, 4, 1, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (210, 1, 5, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (211, 1, 8, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (212, 1, 11, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (214, 1, 16, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (215, 1, 17, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (216, 1, 18, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (217, 1, 19, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (218, 1, 20, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (219, 1, 21, 1, 2, '2026-11-28 04:34:19.355811', '2026-11-28 04:34:19.355811', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (220, 1, 29, 1, 1, '2026-11-28 04:34:19.490965', '2026-11-28 04:34:19.490965', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (221, 1, 29, 3, 1, '2026-11-28 04:34:19.490965', '2026-11-28 04:34:19.490965', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (227, 1, 29, 2, 1, '2026-11-28 04:34:19.695798', '2026-11-28 04:34:19.695798', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (230, 1, 6, 1, 2, '2026-11-28 04:34:19.7797', '2026-11-28 04:34:19.7797', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (233, 3, 31, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (234, 3, 29, 1, 1, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (235, 3, 28, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (236, 3, 27, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (237, 3, 26, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (238, 3, 25, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (239, 3, 24, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (240, 3, 23, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (241, 3, 22, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (242, 3, 21, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (243, 3, 20, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (244, 3, 19, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (245, 3, 18, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (246, 3, 17, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (247, 3, 16, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (248, 3, 15, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (249, 2, 15, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (250, 3, 14, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (251, 3, 13, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (252, 2, 13, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (253, 3, 12, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (254, 3, 11, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (255, 3, 10, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (256, 2, 10, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (257, 3, 9, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (258, 2, 9, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (259, 3, 8, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (260, 3, 7, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (261, 2, 7, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (262, 3, 6, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (263, 2, 5, 1, 1, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (264, 3, 5, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (265, 3, 4, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (266, 3, 3, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (267, 3, 2, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (268, 3, 1, 1, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (269, 3, 27, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (270, 3, 26, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (271, 3, 25, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (272, 3, 24, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (273, 3, 23, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (274, 3, 22, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (275, 3, 21, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (276, 3, 20, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (277, 3, 19, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (278, 3, 18, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (279, 3, 17, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (280, 3, 16, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (281, 3, 15, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (282, 2, 15, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (283, 3, 14, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (284, 3, 13, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (285, 2, 13, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (286, 3, 12, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (287, 3, 11, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (288, 3, 10, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (289, 3, 9, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (290, 2, 9, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (291, 3, 8, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (292, 3, 7, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (293, 2, 7, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (294, 3, 5, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (295, 2, 5, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (296, 3, 4, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (297, 3, 3, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (298, 3, 2, 2, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (299, 3, 31, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (300, 3, 29, 3, 1, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (301, 3, 28, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (302, 3, 27, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (303, 3, 26, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (304, 3, 25, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (305, 3, 24, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (306, 3, 23, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (307, 3, 22, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (308, 3, 21, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (309, 3, 20, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (310, 3, 19, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (311, 3, 18, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (312, 3, 17, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (313, 3, 16, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (314, 3, 15, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (315, 2, 15, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (316, 3, 14, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (317, 3, 13, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (318, 2, 13, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (319, 3, 12, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (320, 3, 11, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (321, 3, 10, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (322, 2, 10, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (323, 3, 9, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (324, 2, 9, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (325, 3, 8, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (326, 3, 7, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (327, 2, 7, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (328, 3, 5, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (329, 2, 5, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (330, 3, 4, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (331, 3, 3, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (332, 3, 2, 3, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (333, 3, 27, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (334, 3, 26, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (335, 3, 25, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (336, 3, 24, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (337, 3, 23, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (338, 3, 22, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (339, 3, 21, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (340, 3, 20, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (341, 3, 19, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (342, 3, 18, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (343, 3, 17, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (344, 3, 16, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (345, 3, 15, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (346, 2, 15, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (347, 3, 14, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (348, 3, 13, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (349, 2, 13, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (350, 3, 12, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (351, 3, 11, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (352, 3, 10, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (353, 3, 9, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (354, 2, 9, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (355, 3, 8, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (356, 3, 7, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (357, 2, 7, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (358, 3, 5, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (359, 2, 5, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (360, 3, 4, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (361, 3, 3, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);
INSERT INTO st_group_resource (id, group_id, resource_id, access_level_id, access_scope_id, created, modified, version) VALUES (362, 3, 2, 4, 2, '2026-11-28 04:35:00.092267', '2026-11-28 04:35:00.092267', 1);


--
-- Name: et_access_level_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_access_level_id_seq', 4, true);


--
-- Name: et_access_scope_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_access_scope_id_seq', 2, true);


--
-- Name: et_booking_status_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_booking_status_id_seq', 4, true);


--
-- Name: et_member_status_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_member_status_id_seq', 1, true);


--
-- Name: et_resource_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_resource_id_seq', 31, true);


--
-- Name: et_seasons_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_seasons_id_seq', 1, true);


--
-- Name: et_trip_difficulty_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_trip_difficulty_id_seq', 1, true);


--
-- Name: et_trip_status_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_trip_status_id_seq', 3, true);


--
-- Name: et_trip_type_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_trip_type_id_seq', 1, true);


--
-- Name: et_user_account_status_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_user_account_status_id_seq', 6, true);


--
-- Name: et_user_age_groups_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('et_user_age_groups_id_seq', 6, true);


--
-- Name: st_group_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('st_group_id_seq', 3, true);


--
-- Name: st_group_resource_id_seq; Type: SEQUENCE SET; Schema: current; Owner: -
--

SELECT pg_catalog.setval('st_group_resource_id_seq', 362, true);


--
-- PostgreSQL database dump complete
--


--
-- PostgreSQL database dump
--





--
-- Name: at_booking_people at_booking_people_booking_id_person_id_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_people
    ADD CONSTRAINT at_booking_people_booking_id_person_id_key UNIQUE (booking_id, person_id);


--
-- Name: at_booking_people at_booking_people_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_people
    ADD CONSTRAINT at_booking_people_pkey PRIMARY KEY (id);


--
-- Name: at_booking_vouchers at_booking_vouchers_code_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_vouchers
    ADD CONSTRAINT at_booking_vouchers_code_key UNIQUE (code);


--
-- Name: at_booking_vouchers at_booking_vouchers_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_vouchers
    ADD CONSTRAINT at_booking_vouchers_pkey PRIMARY KEY (id);


--
-- Name: at_bookings at_bookings_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_bookings
    ADD CONSTRAINT at_bookings_pkey PRIMARY KEY (id);


--
-- Name: at_checkout_attempts at_checkout_attempts_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_attempts
    ADD CONSTRAINT at_checkout_attempts_pkey PRIMARY KEY (id);


--
-- Name: at_checkout_attempts at_checkout_attempts_session_id_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_attempts
    ADD CONSTRAINT at_checkout_attempts_session_id_key UNIQUE (session_id);


--
-- Name: at_checkout_quotes at_checkout_quotes_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_quotes
    ADD CONSTRAINT at_checkout_quotes_pkey PRIMARY KEY (session_id);


--
-- Name: at_checkout_reconciliations at_checkout_reconciliations_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reconciliations
    ADD CONSTRAINT at_checkout_reconciliations_pkey PRIMARY KEY (id);


--
-- Name: at_checkout_reservations at_checkout_reservations_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reservations
    ADD CONSTRAINT at_checkout_reservations_pkey PRIMARY KEY (booking_id);


--
-- Name: at_financial_audit at_financial_audit_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_financial_audit
    ADD CONSTRAINT at_financial_audit_pkey PRIMARY KEY (id);


--
-- Name: at_group_bookings at_group_bookings_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_group_bookings
    ADD CONSTRAINT at_group_bookings_pkey PRIMARY KEY (id);


--
-- Name: at_notification_outbox at_notification_outbox_booking_id_kind_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_notification_outbox
    ADD CONSTRAINT at_notification_outbox_booking_id_kind_key UNIQUE (booking_id, kind);


--
-- Name: at_notification_outbox at_notification_outbox_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_notification_outbox
    ADD CONSTRAINT at_notification_outbox_pkey PRIMARY KEY (id);


--
-- Name: at_payment_reviews at_payment_reviews_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_payment_reviews
    ADD CONSTRAINT at_payment_reviews_pkey PRIMARY KEY (session_id);


--
-- Name: at_payments at_payments_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_payments
    ADD CONSTRAINT at_payments_pkey PRIMARY KEY (id);


--
-- Name: at_provider_refunds at_provider_refunds_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_provider_refunds
    ADD CONSTRAINT at_provider_refunds_pkey PRIMARY KEY (id);


--
-- Name: at_refunds at_refunds_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_refunds
    ADD CONSTRAINT at_refunds_pkey PRIMARY KEY (id);


--
-- Name: at_security_audit at_security_audit_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_security_audit
    ADD CONSTRAINT at_security_audit_pkey PRIMARY KEY (id);


--
-- Name: at_stripe_events at_stripe_events_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_stripe_events
    ADD CONSTRAINT at_stripe_events_pkey PRIMARY KEY (event_id);


--
-- Name: at_trip_cost_groups at_trip_cost_groups_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_cost_groups
    ADD CONSTRAINT at_trip_cost_groups_pkey PRIMARY KEY (id);


--
-- Name: at_trip_costs at_trip_costs_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_pkey PRIMARY KEY (id);


--
-- Name: at_trip_costs at_trip_costs_trip_cost_group_id_member_status_id_user_age__key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_trip_cost_group_id_member_status_id_user_age__key UNIQUE (trip_cost_group_id, member_status_id, user_age_group_id, season_id);


--
-- Name: at_trip_questionnaires at_trip_questionnaires_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_questionnaires
    ADD CONSTRAINT at_trip_questionnaires_pkey PRIMARY KEY (trip_id, revision);


--
-- Name: at_trips at_trips_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_pkey PRIMARY KEY (id);


--
-- Name: at_voucher_redemptions at_voucher_redemptions_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_voucher_redemptions
    ADD CONSTRAINT at_voucher_redemptions_pkey PRIMARY KEY (booking_id);


--
-- Name: et_access_level et_access_level_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_access_level
    ADD CONSTRAINT et_access_level_pkey PRIMARY KEY (id);


--
-- Name: et_access_scope et_access_scope_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_access_scope
    ADD CONSTRAINT et_access_scope_pkey PRIMARY KEY (id);


--
-- Name: et_booking_status et_booking_status_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_booking_status
    ADD CONSTRAINT et_booking_status_pkey PRIMARY KEY (id);


--
-- Name: et_member_status et_member_status_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_member_status
    ADD CONSTRAINT et_member_status_pkey PRIMARY KEY (id);


--
-- Name: et_resource et_resource_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_resource
    ADD CONSTRAINT et_resource_pkey PRIMARY KEY (id);


--
-- Name: et_seasons et_seasons_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_seasons
    ADD CONSTRAINT et_seasons_pkey PRIMARY KEY (id);


--
-- Name: et_trip_difficulty et_trip_difficulty_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_trip_difficulty
    ADD CONSTRAINT et_trip_difficulty_pkey PRIMARY KEY (id);


--
-- Name: et_trip_status et_trip_status_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_trip_status
    ADD CONSTRAINT et_trip_status_pkey PRIMARY KEY (id);


--
-- Name: et_trip_type et_trip_type_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_trip_type
    ADD CONSTRAINT et_trip_type_pkey PRIMARY KEY (id);


--
-- Name: et_user_account_status et_user_account_status_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_user_account_status
    ADD CONSTRAINT et_user_account_status_pkey PRIMARY KEY (id);


--
-- Name: et_user_age_groups et_user_age_groups_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY et_user_age_groups
    ADD CONSTRAINT et_user_age_groups_pkey PRIMARY KEY (id);


--
-- Name: st_account_review_requests st_account_review_requests_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_account_review_requests
    ADD CONSTRAINT st_account_review_requests_pkey PRIMARY KEY (user_id);


--
-- Name: st_appearance_revision st_appearance_revision_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_appearance_revision
    ADD CONSTRAINT st_appearance_revision_pkey PRIMARY KEY (version);


--
-- Name: st_auth_rate_limits st_auth_rate_limits_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_auth_rate_limits
    ADD CONSTRAINT st_auth_rate_limits_pkey PRIMARY KEY (key_hash, window_start);


--
-- Name: st_email_changes st_email_changes_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_email_changes
    ADD CONSTRAINT st_email_changes_pkey PRIMARY KEY (user_id);


--
-- Name: st_group st_group_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group
    ADD CONSTRAINT st_group_pkey PRIMARY KEY (id);


--
-- Name: st_group_resource st_group_resource_group_id_resource_id_access_level_id_acce_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_group_id_resource_id_access_level_id_acce_key UNIQUE (group_id, resource_id, access_level_id, access_scope_id);


--
-- Name: st_group_resource st_group_resource_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_pkey PRIMARY KEY (id);


--
-- Name: st_group st_group_system_key_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group
    ADD CONSTRAINT st_group_system_key_key UNIQUE (system_key);


--
-- Name: st_security_admin_guard st_security_admin_guard_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_security_admin_guard
    ADD CONSTRAINT st_security_admin_guard_pkey PRIMARY KEY (singleton);


--
-- Name: st_token st_token_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_token
    ADD CONSTRAINT st_token_pkey PRIMARY KEY (id);


--
-- Name: st_user_credentials st_user_credentials_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_credentials
    ADD CONSTRAINT st_user_credentials_pkey PRIMARY KEY (id);


--
-- Name: st_user_group st_user_group_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_group
    ADD CONSTRAINT st_user_group_pkey PRIMARY KEY (id);


--
-- Name: st_user_group st_user_group_user_id_group_id_key; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_group
    ADD CONSTRAINT st_user_group_user_id_group_id_key UNIQUE (user_id, group_id);


--
-- Name: st_users st_users_pkey; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT st_users_pkey PRIMARY KEY (id);


--
-- Name: st_users users_email_unique; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT users_email_unique UNIQUE (email);


--
-- Name: st_users users_username_unique; Type: CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT users_username_unique UNIQUE (username);


--
-- Name: at_booking_vouchers voucher_expiry_required; Type: CHECK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE at_booking_vouchers
    ADD CONSTRAINT voucher_expiry_required CHECK ((expiry_date IS NOT NULL)) NOT VALID;


--
-- Name: account_review_recipient; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX account_review_recipient ON at_notification_outbox USING btree (account_id, kind, recipient) WHERE ((account_id IS NOT NULL) AND (recipient <> ''::text));


--
-- Name: booking_owner; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX booking_owner ON at_bookings USING btree (owner_id);


--
-- Name: booking_stripe_session; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX booking_stripe_session ON at_bookings USING btree (stripe_session_id) WHERE (stripe_session_id IS NOT NULL);


--
-- Name: booking_trip; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX booking_trip ON at_bookings USING btree (trip_id);


--
-- Name: checkout_attempt_booking; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX checkout_attempt_booking ON at_checkout_attempts USING btree (booking_id, expires_at);


--
-- Name: checkout_one_active_attempt; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX checkout_one_active_attempt ON at_checkout_attempts USING btree (booking_id) WHERE (state = ANY (ARRAY['pending'::text, 'attached'::text]));


--
-- Name: checkout_recovery_due; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX checkout_recovery_due ON at_checkout_attempts USING btree (reconcile_after, id) WHERE (state = ANY (ARRAY['pending'::text, 'attached'::text]));


--
-- Name: checkout_reservations_expiry; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX checkout_reservations_expiry ON at_checkout_reservations USING btree (expires_at);


--
-- Name: financial_audit_record; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX financial_audit_record ON at_financial_audit USING btree (resource, record_id, id);


--
-- Name: idx_auth_rate_limit_expiry; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX idx_auth_rate_limit_expiry ON st_auth_rate_limits USING btree (window_start);


--
-- Name: notification_outbox_due; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX notification_outbox_due ON at_notification_outbox USING btree (available_at, id) WHERE (sent_at IS NULL);


--
-- Name: participant_booking; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX participant_booking ON at_booking_people USING btree (booking_id);


--
-- Name: payment_booking; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX payment_booking ON at_payments USING btree (booking_id);


--
-- Name: payment_reviews_open; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX payment_reviews_open ON at_payment_reviews USING btree (last_seen) WHERE (resolved_at IS NULL);


--
-- Name: refund_payment; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX refund_payment ON at_refunds USING btree (payment_id);


--
-- Name: refund_provider_identity; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX refund_provider_identity ON at_refunds USING btree (provider_refund_id) WHERE (provider_refund_id IS NOT NULL);


--
-- Name: refund_recovery_due; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX refund_recovery_due ON at_provider_refunds USING btree (reconcile_after, id) WHERE (status = ANY (ARRAY['pending'::text, 'requires_action'::text]));


--
-- Name: security_audit_record; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX security_audit_record ON at_security_audit USING btree (resource, record_id, id);


--
-- Name: token_expiry; Type: INDEX; Schema: current; Owner: -
--

CREATE INDEX token_expiry ON st_token USING btree (valid_to);


--
-- Name: uq_at_payments_stripe_session_id; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX uq_at_payments_stripe_session_id ON at_payments USING btree (stripe_session_id) WHERE (stripe_session_id IS NOT NULL);


--
-- Name: uq_st_token_name_digest; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX uq_st_token_name_digest ON st_token USING btree (name, token);


--
-- Name: users_email_normalized; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX users_email_normalized ON st_users USING btree (lower((email)::text));


--
-- Name: users_member_code_unique; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX users_member_code_unique ON st_users USING btree (lower(btrim((member_code)::text))) WHERE (NULLIF(btrim((member_code)::text), ''::text) IS NOT NULL);


--
-- Name: users_username_case_insensitive_unique; Type: INDEX; Schema: current; Owner: -
--

CREATE UNIQUE INDEX users_username_case_insensitive_unique ON st_users USING btree (lower((username)::text));


--
-- Name: st_users advance_account_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_account_version BEFORE UPDATE ON st_users FOR EACH ROW EXECUTE FUNCTION advance_account_version();


--
-- Name: at_booking_vouchers advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_booking_vouchers FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: at_group_bookings advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_group_bookings FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: at_trip_cost_groups advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_trip_cost_groups FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: at_trip_costs advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_trip_costs FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: at_trips advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_trips FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: st_group advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_group FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: st_group_resource advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_group_resource FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: st_user_group advance_resource_version; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_user_group FOR EACH ROW EXECUTE FUNCTION advance_resource_version();


--
-- Name: st_users assign_member_code; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER assign_member_code BEFORE INSERT ON st_users FOR EACH ROW EXECUTE FUNCTION assign_member_code();


--
-- Name: st_users audit_account; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_account AFTER INSERT OR DELETE OR UPDATE ON st_users FOR EACH ROW EXECUTE FUNCTION record_account_change();


--
-- Name: st_group audit_group_definition; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_group_definition AFTER INSERT OR DELETE OR UPDATE ON st_group FOR EACH ROW EXECUTE FUNCTION record_security_change();


--
-- Name: st_group_resource audit_group_resource; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_group_resource AFTER INSERT OR DELETE OR UPDATE ON st_group_resource FOR EACH ROW EXECUTE FUNCTION record_security_change();


--
-- Name: at_payments audit_payment; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_payment AFTER INSERT OR DELETE OR UPDATE ON at_payments FOR EACH ROW EXECUTE FUNCTION record_financial_change();


--
-- Name: st_users audit_profile_email; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_profile_email AFTER UPDATE OF email ON st_users FOR EACH ROW EXECUTE FUNCTION record_email_change();


--
-- Name: at_refunds audit_refund; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_refund AFTER INSERT OR DELETE OR UPDATE ON at_refunds FOR EACH ROW EXECUTE FUNCTION record_financial_change();


--
-- Name: st_user_group audit_user_group; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER audit_user_group AFTER INSERT OR DELETE OR UPDATE ON st_user_group FOR EACH ROW EXECUTE FUNCTION record_security_change();


--
-- Name: at_bookings checkout_booking_inputs; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER checkout_booking_inputs BEFORE UPDATE ON at_bookings FOR EACH ROW EXECUTE FUNCTION protect_checkout_inputs();


--
-- Name: at_bookings checkout_booking_voucher; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER checkout_booking_voucher BEFORE UPDATE OF voucher_code ON at_bookings FOR EACH ROW EXECUTE FUNCTION protect_checkout_voucher();


--
-- Name: at_booking_people checkout_participant_inputs; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER checkout_participant_inputs BEFORE INSERT OR DELETE OR UPDATE ON at_booking_people FOR EACH ROW EXECUTE FUNCTION protect_checkout_inputs();


--
-- Name: at_financial_audit immutable_financial_audit; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER immutable_financial_audit BEFORE DELETE OR UPDATE OR TRUNCATE ON at_financial_audit FOR EACH STATEMENT EXECUTE FUNCTION protect_financial_audit();


--
-- Name: at_payments immutable_provider_payment; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER immutable_provider_payment BEFORE DELETE OR UPDATE ON at_payments FOR EACH ROW EXECUTE FUNCTION protect_provider_ledger();


--
-- Name: at_refunds immutable_provider_refund; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER immutable_provider_refund BEFORE DELETE OR UPDATE ON at_refunds FOR EACH ROW EXECUTE FUNCTION protect_provider_ledger();


--
-- Name: at_security_audit immutable_security_audit; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER immutable_security_audit BEFORE DELETE OR UPDATE OR TRUNCATE ON at_security_audit FOR EACH STATEMENT EXECUTE FUNCTION protect_financial_audit();


--
-- Name: st_group_resource preserve_security_admin_grants; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_grants AFTER INSERT OR DELETE OR UPDATE ON st_group_resource DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: st_user_group preserve_security_admin_memberships; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_memberships AFTER INSERT OR DELETE OR UPDATE ON st_user_group DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: et_access_level preserve_security_admin_methods; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_methods AFTER DELETE OR UPDATE OF name ON et_access_level DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: et_resource preserve_security_admin_resources; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_resources AFTER DELETE OR UPDATE OF name ON et_resource DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: et_access_scope preserve_security_admin_scopes; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_scopes AFTER DELETE OR UPDATE OF name ON et_access_scope DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: st_users preserve_security_admin_users; Type: TRIGGER; Schema: current; Owner: -
--

CREATE CONSTRAINT TRIGGER preserve_security_admin_users AFTER DELETE OR UPDATE OF user_account_status_id ON st_users DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();


--
-- Name: st_group protect_group_system_key; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER protect_group_system_key BEFORE INSERT OR DELETE OR UPDATE OF system_key ON st_group FOR EACH ROW EXECUTE FUNCTION protect_group_system_key();


--
-- Name: st_users queue_new_account_review; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER queue_new_account_review AFTER INSERT ON st_users FOR EACH ROW EXECUTE FUNCTION queue_account_review();


--
-- Name: at_booking_people set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_booking_people FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_booking_vouchers set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_booking_vouchers FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_bookings set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_bookings FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_group_bookings set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_group_bookings FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_payments set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_payments FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_refunds set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_refunds FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_trip_cost_groups set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_trip_cost_groups FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_trip_costs set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_trip_costs FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_trips set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON at_trips FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_access_level set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_access_level FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_access_scope set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_access_scope FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_booking_status set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_booking_status FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_member_status set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_member_status FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_resource set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_resource FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_seasons set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_seasons FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_trip_difficulty set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_trip_difficulty FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_trip_status set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_trip_status FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_trip_type set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_trip_type FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_user_account_status set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_user_account_status FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: et_user_age_groups set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON et_user_age_groups FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_group set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_group FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_group_resource set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_group_resource FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_token set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_token FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_user_credentials set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_user_credentials FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_user_group set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_user_group FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: st_users set_modified; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER set_modified BEFORE UPDATE ON st_users FOR EACH ROW EXECUTE FUNCTION update_modified_column();


--
-- Name: at_trips trip_capacity_guard; Type: TRIGGER; Schema: current; Owner: -
--

CREATE TRIGGER trip_capacity_guard BEFORE UPDATE OF max_participants ON at_trips FOR EACH ROW EXECUTE FUNCTION guard_trip_capacity();


--
-- Name: at_booking_people at_booking_people_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_people
    ADD CONSTRAINT at_booking_people_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE CASCADE;


--
-- Name: at_booking_people at_booking_people_owner_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_people
    ADD CONSTRAINT at_booking_people_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES st_users(id) ON DELETE RESTRICT;


--
-- Name: at_booking_people at_booking_people_person_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_booking_people
    ADD CONSTRAINT at_booking_people_person_id_fkey FOREIGN KEY (person_id) REFERENCES st_users(id) ON DELETE RESTRICT;


--
-- Name: at_bookings at_bookings_booking_status_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_bookings
    ADD CONSTRAINT at_bookings_booking_status_id_fkey FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id) ON DELETE RESTRICT;


--
-- Name: at_bookings at_bookings_group_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_bookings
    ADD CONSTRAINT at_bookings_group_booking_id_fkey FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id) ON DELETE RESTRICT;


--
-- Name: at_bookings at_bookings_owner_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_bookings
    ADD CONSTRAINT at_bookings_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES st_users(id) ON DELETE RESTRICT;


--
-- Name: at_bookings at_bookings_trip_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_bookings
    ADD CONSTRAINT at_bookings_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES at_trips(id) ON DELETE RESTRICT;


--
-- Name: at_checkout_attempts at_checkout_attempts_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_attempts
    ADD CONSTRAINT at_checkout_attempts_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE RESTRICT;


--
-- Name: at_checkout_quotes at_checkout_quotes_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_quotes
    ADD CONSTRAINT at_checkout_quotes_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE RESTRICT;


--
-- Name: at_checkout_reconciliations at_checkout_reconciliations_actor_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reconciliations
    ADD CONSTRAINT at_checkout_reconciliations_actor_id_fkey FOREIGN KEY (actor_id) REFERENCES st_users(id);


--
-- Name: at_checkout_reconciliations at_checkout_reconciliations_attempt_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reconciliations
    ADD CONSTRAINT at_checkout_reconciliations_attempt_id_fkey FOREIGN KEY (attempt_id) REFERENCES at_checkout_attempts(id);


--
-- Name: at_checkout_reservations at_checkout_reservations_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_checkout_reservations
    ADD CONSTRAINT at_checkout_reservations_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE RESTRICT;


--
-- Name: at_group_bookings at_group_bookings_owner_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_group_bookings
    ADD CONSTRAINT at_group_bookings_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES st_users(id) ON DELETE RESTRICT;


--
-- Name: at_notification_outbox at_notification_outbox_account_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_notification_outbox
    ADD CONSTRAINT at_notification_outbox_account_id_fkey FOREIGN KEY (account_id) REFERENCES st_users(id) ON DELETE SET NULL;


--
-- Name: at_notification_outbox at_notification_outbox_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_notification_outbox
    ADD CONSTRAINT at_notification_outbox_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE SET NULL;


--
-- Name: at_payment_reviews at_payment_reviews_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_payment_reviews
    ADD CONSTRAINT at_payment_reviews_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id);


--
-- Name: at_payment_reviews at_payment_reviews_session_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_payment_reviews
    ADD CONSTRAINT at_payment_reviews_session_id_fkey FOREIGN KEY (session_id) REFERENCES at_checkout_quotes(session_id);


--
-- Name: at_payments at_payments_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_payments
    ADD CONSTRAINT at_payments_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE RESTRICT;


--
-- Name: at_provider_refunds at_provider_refunds_payment_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_provider_refunds
    ADD CONSTRAINT at_provider_refunds_payment_id_fkey FOREIGN KEY (payment_id) REFERENCES at_payments(id);


--
-- Name: at_refunds at_refunds_payment_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_refunds
    ADD CONSTRAINT at_refunds_payment_id_fkey FOREIGN KEY (payment_id) REFERENCES at_payments(id) ON DELETE RESTRICT;


--
-- Name: at_stripe_events at_stripe_events_session_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_stripe_events
    ADD CONSTRAINT at_stripe_events_session_id_fkey FOREIGN KEY (session_id) REFERENCES at_checkout_quotes(session_id);


--
-- Name: at_trip_costs at_trip_costs_member_status_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_member_status_id_fkey FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) ON DELETE RESTRICT;


--
-- Name: at_trip_costs at_trip_costs_season_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_season_id_fkey FOREIGN KEY (season_id) REFERENCES et_seasons(id) ON DELETE RESTRICT;


--
-- Name: at_trip_costs at_trip_costs_trip_cost_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_trip_cost_group_id_fkey FOREIGN KEY (trip_cost_group_id) REFERENCES at_trip_cost_groups(id) ON DELETE RESTRICT;


--
-- Name: at_trip_costs at_trip_costs_user_age_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_costs
    ADD CONSTRAINT at_trip_costs_user_age_group_id_fkey FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) ON DELETE RESTRICT;


--
-- Name: at_trip_questionnaires at_trip_questionnaires_trip_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trip_questionnaires
    ADD CONSTRAINT at_trip_questionnaires_trip_id_fkey FOREIGN KEY (trip_id) REFERENCES at_trips(id) ON DELETE CASCADE;


--
-- Name: at_trips at_trips_difficulty_level_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_difficulty_level_id_fkey FOREIGN KEY (difficulty_level_id) REFERENCES et_trip_difficulty(id) ON DELETE RESTRICT;


--
-- Name: at_trips at_trips_owner_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES st_users(id) ON DELETE RESTRICT;


--
-- Name: at_trips at_trips_trip_cost_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_trip_cost_group_id_fkey FOREIGN KEY (trip_cost_group_id) REFERENCES at_trip_cost_groups(id) ON DELETE RESTRICT;


--
-- Name: at_trips at_trips_trip_status_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_trip_status_id_fkey FOREIGN KEY (trip_status_id) REFERENCES et_trip_status(id) ON DELETE RESTRICT;


--
-- Name: at_trips at_trips_trip_type_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_trips
    ADD CONSTRAINT at_trips_trip_type_id_fkey FOREIGN KEY (trip_type_id) REFERENCES et_trip_type(id) ON DELETE RESTRICT;


--
-- Name: at_voucher_redemptions at_voucher_redemptions_booking_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY at_voucher_redemptions
    ADD CONSTRAINT at_voucher_redemptions_booking_id_fkey FOREIGN KEY (booking_id) REFERENCES at_bookings(id) ON DELETE RESTRICT;


--
-- Name: st_account_review_requests st_account_review_requests_user_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_account_review_requests
    ADD CONSTRAINT st_account_review_requests_user_id_fkey FOREIGN KEY (user_id) REFERENCES st_users(id) ON DELETE CASCADE;


--
-- Name: st_appearance_revision st_appearance_revision_published_by_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_appearance_revision
    ADD CONSTRAINT st_appearance_revision_published_by_fkey FOREIGN KEY (published_by) REFERENCES st_users(id);


--
-- Name: st_email_changes st_email_changes_user_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_email_changes
    ADD CONSTRAINT st_email_changes_user_id_fkey FOREIGN KEY (user_id) REFERENCES st_users(id) ON DELETE CASCADE;


--
-- Name: st_group_resource st_group_resource_access_level_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_access_level_id_fkey FOREIGN KEY (access_level_id) REFERENCES et_access_level(id) ON DELETE RESTRICT;


--
-- Name: st_group_resource st_group_resource_access_scope_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_access_scope_id_fkey FOREIGN KEY (access_scope_id) REFERENCES et_access_scope(id) ON DELETE RESTRICT;


--
-- Name: st_group_resource st_group_resource_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_group_id_fkey FOREIGN KEY (group_id) REFERENCES st_group(id) ON DELETE CASCADE;


--
-- Name: st_group_resource st_group_resource_resource_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_group_resource
    ADD CONSTRAINT st_group_resource_resource_id_fkey FOREIGN KEY (resource_id) REFERENCES et_resource(id) ON DELETE CASCADE;


--
-- Name: st_user_credentials st_user_credentials_user_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_credentials
    ADD CONSTRAINT st_user_credentials_user_id_fkey FOREIGN KEY (user_id) REFERENCES st_users(id) ON DELETE CASCADE;


--
-- Name: st_user_group st_user_group_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_group
    ADD CONSTRAINT st_user_group_group_id_fkey FOREIGN KEY (group_id) REFERENCES st_group(id) ON DELETE CASCADE;


--
-- Name: st_user_group st_user_group_user_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_user_group
    ADD CONSTRAINT st_user_group_user_id_fkey FOREIGN KEY (user_id) REFERENCES st_users(id) ON DELETE CASCADE;


--
-- Name: st_users st_users_member_status_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT st_users_member_status_id_fkey FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) ON DELETE RESTRICT;


--
-- Name: st_users st_users_user_account_status_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT st_users_user_account_status_id_fkey FOREIGN KEY (user_account_status_id) REFERENCES et_user_account_status(id) ON DELETE RESTRICT;


--
-- Name: st_users st_users_user_age_group_id_fkey; Type: FK CONSTRAINT; Schema: current; Owner: -
--

ALTER TABLE ONLY st_users
    ADD CONSTRAINT st_users_user_age_group_id_fkey FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--


INSERT INTO st_security_admin_guard (singleton, armed) VALUES (true, false);
ALTER TABLE schema_migrations ADD COLUMN compatible_since text;
