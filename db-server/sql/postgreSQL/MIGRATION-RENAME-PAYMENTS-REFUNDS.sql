-- ============================================================================
-- Migration: Rename payment tables and standardize payment ledger shape
-- ============================================================================
-- Purpose:
--   - Rename at_user_payments -> at_payments
--   - Rename at_payment_refunds / at_payments_refunds -> at_refunds
--   - Ensure at_payments has stripe_session_id for idempotent Stripe upserts
--
-- Notes:
--   - Idempotent and safe to rerun.
-- ============================================================================

BEGIN;

-- Rename payments table if still using legacy name.
DO $$
BEGIN
    IF to_regclass('public.at_payments') IS NULL
       AND to_regclass('public.at_user_payments') IS NOT NULL THEN
        ALTER TABLE public.at_user_payments RENAME TO at_payments;
    END IF;
END $$;

-- Rename refunds table if still using older names.
DO $$
BEGIN
    IF to_regclass('public.at_refunds') IS NULL
       AND to_regclass('public.at_payment_refunds') IS NOT NULL THEN
        ALTER TABLE public.at_payment_refunds RENAME TO at_refunds;
    ELSIF to_regclass('public.at_refunds') IS NULL
       AND to_regclass('public.at_payments_refunds') IS NOT NULL THEN
        ALTER TABLE public.at_payments_refunds RENAME TO at_refunds;
    END IF;
END $$;

-- Ensure payment table has needed columns.
ALTER TABLE public.at_payments
    ADD COLUMN IF NOT EXISTS stripe_session_id character varying(255),
    ADD COLUMN IF NOT EXISTS modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE public.at_payments
    DROP COLUMN IF EXISTS user_id;

-- Normalize FK names after rename.
ALTER TABLE public.at_payments
    DROP CONSTRAINT IF EXISTS at_user_payments_booking_id_fkey;
ALTER TABLE public.at_payments
    DROP CONSTRAINT IF EXISTS at_payments_booking_id_fkey;
ALTER TABLE public.at_payments
    ADD CONSTRAINT at_payments_booking_id_fkey
    FOREIGN KEY (booking_id) REFERENCES public.at_bookings(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE public.at_refunds
    DROP CONSTRAINT IF EXISTS at_payment_refunds_payment_id_fkey;
ALTER TABLE public.at_refunds
    DROP CONSTRAINT IF EXISTS at_payments_refunds_payment_id_fkey;
ALTER TABLE public.at_refunds
    DROP CONSTRAINT IF EXISTS at_refunds_payment_id_fkey;
ALTER TABLE public.at_refunds
    ADD CONSTRAINT at_refunds_payment_id_fkey
    FOREIGN KEY (payment_id) REFERENCES public.at_payments(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Rename legacy indexes when present and ensure target indexes exist.
DO $$
BEGIN
    IF to_regclass('public.idx_at_user_payments_booking_id') IS NOT NULL
       AND to_regclass('public.idx_at_payments_booking_id') IS NULL THEN
        ALTER INDEX public.idx_at_user_payments_booking_id RENAME TO idx_at_payments_booking_id;
    END IF;

    IF to_regclass('public.idx_at_user_payments_payment_date') IS NOT NULL
       AND to_regclass('public.idx_at_payments_payment_date') IS NULL THEN
        ALTER INDEX public.idx_at_user_payments_payment_date RENAME TO idx_at_payments_payment_date;
    END IF;

    IF to_regclass('public.idx_at_payment_refunds_payment_id') IS NOT NULL
       AND to_regclass('public.idx_at_refunds_payment_id') IS NULL THEN
        ALTER INDEX public.idx_at_payment_refunds_payment_id RENAME TO idx_at_refunds_payment_id;
    END IF;

    IF to_regclass('public.idx_at_payment_refunds_refund_date') IS NOT NULL
       AND to_regclass('public.idx_at_refunds_refund_date') IS NULL THEN
        ALTER INDEX public.idx_at_payment_refunds_refund_date RENAME TO idx_at_refunds_refund_date;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_at_payments_booking_id
    ON public.at_payments (booking_id);

CREATE INDEX IF NOT EXISTS idx_at_payments_payment_date
    ON public.at_payments (payment_date);

CREATE INDEX IF NOT EXISTS idx_at_refunds_payment_id
    ON public.at_refunds (payment_id);

CREATE INDEX IF NOT EXISTS idx_at_refunds_refund_date
    ON public.at_refunds (refund_date);

CREATE UNIQUE INDEX IF NOT EXISTS uq_at_payments_stripe_session_id
    ON public.at_payments (stripe_session_id)
    WHERE stripe_session_id IS NOT NULL;

COMMIT;

-- Verification
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_name IN ('at_payments', 'at_refunds')
ORDER BY table_name;

SELECT
    tc.table_name,
    tc.constraint_name,
    rc.delete_rule,
    rc.update_rule
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
   AND tc.constraint_schema = rc.constraint_schema
WHERE tc.table_schema = 'public'
  AND tc.constraint_name IN (
      'at_payments_booking_id_fkey',
      'at_refunds_payment_id_fkey'
  )
ORDER BY tc.constraint_name;