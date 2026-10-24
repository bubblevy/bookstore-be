-- Add payment_method_code column and foreign key to payment_methods table
ALTER TABLE payments 
ADD COLUMN IF NOT EXISTS payment_method_code VARCHAR(50) REFERENCES payment_methods(code) ON UPDATE CASCADE ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_payments_payment_method_code ON payments(payment_method_code);

-- Backfill existing payments where possible based on payment_type and bank
UPDATE payments
SET payment_method_code = 'bca_va'
WHERE payment_method_code IS NULL AND payment_type = 'bank_transfer' AND bank = 'bca';

UPDATE payments
SET payment_method_code = 'bni_va'
WHERE payment_method_code IS NULL AND payment_type = 'bank_transfer' AND bank = 'bni';

UPDATE payments
SET payment_method_code = 'bri_va'
WHERE payment_method_code IS NULL AND payment_type = 'bank_transfer' AND bank = 'bri';

UPDATE payments
SET payment_method_code = 'permata_va'
WHERE payment_method_code IS NULL AND payment_type = 'bank_transfer' AND bank = 'permata';

UPDATE payments
SET payment_method_code = 'cimb_va'
WHERE payment_method_code IS NULL AND payment_type = 'bank_transfer' AND bank = 'cimb';

UPDATE payments
SET payment_method_code = 'mandiri_bill'
WHERE payment_method_code IS NULL AND payment_type = 'echannel';

UPDATE payments
SET payment_method_code = 'qris'
WHERE payment_method_code IS NULL AND payment_type = 'qris';

UPDATE payments
SET payment_method_code = 'gopay'
WHERE payment_method_code IS NULL AND payment_type = 'gopay';

UPDATE payments
SET payment_method_code = 'shopeepay'
WHERE payment_method_code IS NULL AND payment_type = 'shopeepay';

UPDATE payments
SET payment_method_code = 'indomaret'
WHERE payment_method_code IS NULL AND (payment_type = 'cstore' AND (store = 'indomaret' OR bank = 'indomaret'));

UPDATE payments
SET payment_method_code = 'alfamart'
WHERE payment_method_code IS NULL AND (payment_type = 'cstore' AND (store = 'alfamart' OR bank = 'alfamart'));
