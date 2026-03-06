-- Access scope migration
-- Step 1: rename et_access_type -> et_access_scope
-- Step 2: normalize legacy values to scope semantics (admin->any, owner->own)

BEGIN;

DO $$
BEGIN
	IF to_regclass('public.et_access_scope') IS NULL
		AND to_regclass('public.et_access_type') IS NOT NULL THEN
		ALTER TABLE public.et_access_type RENAME TO et_access_scope;
	END IF;
END $$;

DO $$
BEGIN
	IF to_regclass('public.et_access_type_id_seq') IS NOT NULL
		AND to_regclass('public.et_access_scope_id_seq') IS NULL THEN
		ALTER SEQUENCE public.et_access_type_id_seq RENAME TO et_access_scope_id_seq;
	END IF;
END $$;

DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_name = 'et_access_scope'
	) THEN
		UPDATE public.et_access_scope
		SET
			name = CASE
				WHEN LOWER(name) = 'admin' THEN 'any'
				WHEN LOWER(name) = 'owner' THEN 'own'
				ELSE LOWER(name)
			END,
			description = CASE
				WHEN LOWER(name) = 'admin' THEN 'Scope applies to any records'
				WHEN LOWER(name) = 'owner' THEN 'Scope applies only to caller-owned records'
				ELSE description
			END
		WHERE name IS NOT NULL;
	END IF;
END $$;

DO $$
BEGIN
	IF EXISTS (
		SELECT 1
		FROM information_schema.tables
		WHERE table_schema = 'public'
			AND table_name = 'et_access_scope'
	) THEN
		IF EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = 'et_access_scope'
				AND column_name = 'created'
		)
		AND EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
				AND table_name = 'et_access_scope'
				AND column_name = 'modified'
		) THEN
			INSERT INTO public.et_access_scope (name, description, created, modified)
			SELECT 'any', 'Scope applies to any records', NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM public.et_access_scope WHERE LOWER(name) = 'any'
			);

			INSERT INTO public.et_access_scope (name, description, created, modified)
			SELECT 'own', 'Scope applies only to caller-owned records', NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM public.et_access_scope WHERE LOWER(name) = 'own'
			);
		ELSE
			INSERT INTO public.et_access_scope (name, description)
			SELECT 'any', 'Scope applies to any records'
			WHERE NOT EXISTS (
				SELECT 1 FROM public.et_access_scope WHERE LOWER(name) = 'any'
			);

			INSERT INTO public.et_access_scope (name, description)
			SELECT 'own', 'Scope applies only to caller-owned records'
			WHERE NOT EXISTS (
				SELECT 1 FROM public.et_access_scope WHERE LOWER(name) = 'own'
			);
		END IF;
	END IF;
END $$;

COMMIT;
