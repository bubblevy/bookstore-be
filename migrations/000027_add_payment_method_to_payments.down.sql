DROP INDEX IF EXISTS idx_payments_payment_method_code;
ALTER TABLE payments DROP COLUMN IF EXISTS payment_method_code;
