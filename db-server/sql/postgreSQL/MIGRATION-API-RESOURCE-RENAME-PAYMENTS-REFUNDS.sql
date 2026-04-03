-- Migration: align API resource names with renamed endpoints.
--   userPayments  -> payments
--   paymentRefunds -> refunds
-- Also seeds refunds permissions from payments when refunds did not exist.

DO $$
DECLARE
    payments_resource_id integer;
    refunds_resource_id integer;
BEGIN
    -- Rename legacy payments resource name if present.
    UPDATE public.et_resource
    SET name = 'payments',
        modified = CURRENT_TIMESTAMP
    WHERE lower(name) = 'userpayments';

    -- Rename legacy refunds resource name if present.
    UPDATE public.et_resource
    SET name = 'refunds',
        modified = CURRENT_TIMESTAMP
    WHERE lower(name) = 'paymentrefunds';

    SELECT id INTO payments_resource_id
    FROM public.et_resource
    WHERE lower(name) = 'payments'
    ORDER BY id
    LIMIT 1;

    SELECT id INTO refunds_resource_id
    FROM public.et_resource
    WHERE lower(name) = 'refunds'
    ORDER BY id
    LIMIT 1;

    -- If refunds is missing entirely, create it.
    IF refunds_resource_id IS NULL THEN
        INSERT INTO public.et_resource (name, description, created, modified)
        VALUES ('refunds', 'API resource', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
        RETURNING id INTO refunds_resource_id;
    END IF;

    -- Seed refunds access from payments access so role behavior stays consistent.
    IF payments_resource_id IS NOT NULL AND refunds_resource_id IS NOT NULL THEN
        INSERT INTO public.st_group_resource (group_id, resource_id, access_level_id, access_scope_id, created, modified)
        SELECT sgr.group_id, refunds_resource_id, sgr.access_level_id, sgr.access_scope_id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
        FROM public.st_group_resource AS sgr
        WHERE sgr.resource_id = payments_resource_id
          AND NOT EXISTS (
              SELECT 1
              FROM public.st_group_resource AS existing
              WHERE existing.group_id = sgr.group_id
                AND existing.resource_id = refunds_resource_id
                AND existing.access_level_id = sgr.access_level_id
                AND existing.access_scope_id = sgr.access_scope_id
          );
    END IF;
END $$;
