-- 1. Hapus Tabel Anak (Foreign Key Dependent)
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS payment_status;
DROP TYPE IF EXISTS order_status;