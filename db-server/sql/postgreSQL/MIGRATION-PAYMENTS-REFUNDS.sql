-- ============================================================================
-- Migration: Payments and refunds record model
-- ============================================================================
-- Purpose:
--   - Keep payments explicitly tied to bookings
--   - Add refunds explicitly tied to payments
--   - Add supporting indexes and FK constraints
--
-- Notes:
--   - Idempotent and safe to rerun.
--   - Uses existing table at_payments as the booking-payment ledger.
-- ============================================================================

BEGIN;

-- Ensure required payment columns exist.
ALTER TABLE public.at_payments
    ADD COLUMN IF NOT EXISTS booking_id integer,
    ADD COLUMN IF NOT EXISTS payment_date timestamp without time zone,
    ADD COLUMN IF NOT EXISTS amount numeric(10,2),
    ADD COLUMN IF NOT EXISTS payment_method character varying(255),
    ADD COLUMN IF NOT EXISTS stripe_session_id character varying(255),
    ADD COLUMN IF NOT EXISTS created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    ADD COLUMN IF NOT EXISTS modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE public.at_payments
    DROP COLUMN IF EXISTS user_id;

-- Ensure payment core fields are not null for new and existing data.
UPDATE public.at_payments
   SET payment_date = COALESCE(payment_date, CURRENT_TIMESTAMP),
       amount = COALESCE(amount, 0),
       payment_method = COALESCE(NULLIF(payment_method, ''), 'unknown')
 WHERE payment_date IS NULL
    OR amount IS NULL
    OR payment_method IS NULL
    OR payment_method = '';

ALTER TABLE public.at_payments
    ALTER COLUMN booking_id SET NOT NULL,
    ALTER COLUMN payment_date SET NOT NULL,
    ALTER COLUMN amount SET NOT NULL;

-- Add/refresh payments FK constraints.
ALTER TABLE public.at_payments
    DROP CONSTRAINT IF EXISTS at_payments_booking_id_fkey;

ALTER TABLE public.at_payments
    ADD CONSTRAINT at_payments_booking_id_fkey
    FOREIGN KEY (booking_id) REFERENCES public.at_bookings(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

CREATE INDEX IF NOT EXISTS idx_at_payments_booking_id
    ON public.at_payments (booking_id);

CREATE INDEX IF NOT EXISTS idx_at_payments_payment_date
    ON public.at_payments (payment_date);

CREATE UNIQUE INDEX IF NOT EXISTS uq_at_payments_stripe_session_id
    ON public.at_payments (stripe_session_id)
    WHERE stripe_session_id IS NOT NULL;

-- Create refunds table related to payments.
CREATE TABLE IF NOT EXISTS public.at_refunds (
    id serial PRIMARY KEY,
    payment_id integer NOT NULL,
    refund_date timestamp without time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
    amount numeric(10,2) NOT NULL,
    reason text,
    external_ref character varying(255),
    created timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    modified timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT at_refunds_amount_check CHECK (amount >= 0)
);

ALTER TABLE public.at_refunds
    DROP CONSTRAINT IF EXISTS at_refunds_payment_id_fkey;

ALTER TABLE public.at_refunds
    ADD CONSTRAINT at_refunds_payment_id_fkey
    FOREIGN KEY (payment_id) REFERENCES public.at_payments(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

CREATE INDEX IF NOT EXISTS idx_at_refunds_payment_id
    ON public.at_refunds (payment_id);

CREATE INDEX IF NOT EXISTS idx_at_refunds_refund_date
    ON public.at_refunds (refund_date);

COMMIT;

-- Verification: payment/refund relationship metadata
SELECT
    tc.table_name,
    kcu.column_name,
    tc.constraint_name,
    rc.delete_rule,
    rc.update_rule
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
   AND tc.constraint_schema = kcu.constraint_schema
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
   AND tc.constraint_schema = rc.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = 'public'
  AND tc.constraint_name IN (
            'at_payments_booking_id_fkey',
            'at_refunds_payment_id_fkey'
  )
ORDER BY tc.table_name, kcu.column_name;