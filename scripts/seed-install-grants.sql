\set ON_ERROR_STOP on
BEGIN;

-- This replaces only the built-in administrative grants on a new installation.
LOCK TABLE st_users IN SHARE MODE;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM st_users) THEN
        RAISE EXCEPTION 'Apply initial grants before registering any accounts';
    END IF;
END;
$$;

CREATE TEMP TABLE install_grants (
    group_key text NOT NULL,
    resource text NOT NULL,
    scope text NOT NULL,
    methods text[] NOT NULL
) ON COMMIT DROP;

INSERT INTO install_grants VALUES
    ('admin', 'bookingVouchers', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('admin', 'groupBooking', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('admin', 'payments', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('admin', 'refunds', 'any', ARRAY['get', 'put']),
    ('admin', 'trips', 'any', ARRAY['delete', 'post', 'put']),
    ('admin', 'trips', 'own', ARRAY['get']),
    ('admin', 'users', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'auth', 'any', ARRAY['get']),
    ('sysadmin', 'bookingPeople', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'bookings', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'bookingStatus', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'bookingVouchers', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'gmailcert', 'any', ARRAY['get', 'put']),
    ('sysadmin', 'groupBooking', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'myBookings', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'myProfile', 'own', ARRAY['get', 'put']),
    ('sysadmin', 'payments', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'refunds', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'seasons', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityAccessLevel', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityAccessScope', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityGroup', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityGroupResource', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityResource', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'securityUserGroup', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'siteAppearance', 'any', ARRAY['get', 'put']),
    ('sysadmin', 'tripCostGroups', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'tripCosts', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'tripDifficulty', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'trips', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'tripsReport', 'any', ARRAY['get']),
    ('sysadmin', 'tripStatus', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'tripType', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'userAccountStatus', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'userAgeGroups', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'userMemberStatus', 'any', ARRAY['delete', 'get', 'post', 'put']),
    ('sysadmin', 'users', 'any', ARRAY['delete', 'get', 'post', 'put']);

CREATE TEMP TABLE resolved_install_grants ON COMMIT DROP AS
SELECT sg.id AS group_id, r.id AS resource_id, a.id AS access_level_id, s.id AS access_scope_id
FROM install_grants i
CROSS JOIN LATERAL unnest(i.methods) AS m(method)
JOIN st_group sg ON sg.system_key = i.group_key
JOIN et_resource r ON lower(r.name) = lower(i.resource)
JOIN et_access_level a ON lower(a.name) = m.method
JOIN et_access_scope s ON lower(s.name) = i.scope;

DO $$
BEGIN
    IF (SELECT count(*) FROM resolved_install_grants)
       <> (SELECT sum(cardinality(methods)) FROM install_grants) THEN
        RAISE EXCEPTION 'The initial grant list could not be resolved against the schema';
    END IF;
END;
$$;

DELETE FROM st_group_resource g USING st_group sg
WHERE g.group_id = sg.id AND sg.system_key IN ('admin', 'sysadmin');

-- Booking status definitions belong to system administration.
DELETE FROM st_group_resource g USING st_group sg, et_resource r
WHERE g.group_id = sg.id AND sg.system_key = 'user'
  AND g.resource_id = r.id AND lower(r.name) = 'bookingstatus';

INSERT INTO st_group_resource (group_id, resource_id, access_level_id, access_scope_id)
SELECT group_id, resource_id, access_level_id, access_scope_id FROM resolved_install_grants;

COMMIT;
