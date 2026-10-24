CREATE TABLE IF NOT EXISTS payment_methods (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    category VARCHAR(50) NOT NULL,
    fee_type VARCHAR(20) NOT NULL DEFAULT 'fixed',
    fee_amount BIGINT NOT NULL DEFAULT 0,
    fee_percent NUMERIC(6, 4) NOT NULL DEFAULT 0,
    image VARCHAR(255) DEFAULT '',
    description TEXT DEFAULT '',
    instructions JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_payment_methods_category ON payment_methods(category);
CREATE INDEX IF NOT EXISTS idx_payment_methods_is_active ON payment_methods(is_active);

DROP TRIGGER IF EXISTS trg_payment_methods_updated_at ON payment_methods;
CREATE TRIGGER trg_payment_methods_updated_at
    BEFORE UPDATE ON payment_methods
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seed Default Payment Methods with Instructions
INSERT INTO payment_methods (code, name, category, fee_type, fee_amount, fee_percent, image, description, instructions, is_active)
VALUES
(
    'bca_va',
    'BCA',
    'virtual_account',
    'fixed',
    5000,
    0,
    'payments/bca_logo.svg',
    'Bayar via ATM BCA, BCA mobile, myBCA, atau KlikBCA',
    $$[
        {"title": "Melalui Internet Banking", "content": "<ol start=\"1\"><li><p>Login ke internet banking klikbca Anda</p></li><li><p>Pilih menu Transfer Dana lalu pilih Transfer ke BCA Virtual Account</p></li><li><p>Pilih Dari Rekening</p></li><li><p>Masukkan Nomor VA BCA lalu klik Lanjutkan</p></li><li><p>Detail transaksi akan ditampilkan, pastikan data sudah sesuai lalu masukkan respon Key BCA Appli 1</p></li><li><p>Klik Kirim</p></li><li><p>Transaksi sukses, simpan bukti transaksi Anda</p></li></ol>"},
        {"title": "Melalui Mobile Banking", "content": "<ol start=\"1\"><li><p>Login pada aplikasi BCA mobile</p></li><li><p>Pilih m-BCA masukkan kode akses m-BCA</p></li><li><p>Pilih m-Transfer</p></li><li><p>Pilih BCA Virtual Account</p></li><li><p>Masukkan Nomor VA BCA lalu klik OK</p></li><li><p>Konfirmasi no virtual account dan rekening pendebetan</p></li><li><p>Periksa kembalian rincian pembayaran anda, lalu klik Ya</p></li><li><p>Masukkan pin m-BCA</p></li><li><p>Pembayaran Selesai</p></li></ol>"},
        {"title": "Melalui ATM BCA", "content": "<ol start=\"1\"><li><p>Masukkan kartu ATM BCA &amp; PIN</p></li><li><p>Pilih menu Transaksi Lainnya &gt; Transfer &gt; Ke Rekening BCA Virtual Account</p></li><li><p>Masukkan Nomor VA BCA</p></li><li><p>Di halaman konfirmasi, pastikan detail pembayaran sudah sesuai seperti Nomor VA, Nama, Produk dan Total Tagihan</p></li><li><p>Jika sudah benar, klik Ya</p></li><li><p>Simpan struk transaksi sebagai bukti pembayaran</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'bni_va',
    'BNI',
    'virtual_account',
    'fixed',
    5000,
    0,
    'payments/bni_logo.svg',
    'Bayar via ATM BNI, BNI Mobile Banking, atau Internet Banking',
    $$[
        {"title": "Melalui Wondr By BNI", "content": "<ol start=\"1\"><li><p>Buka Aplikasi Wondr By BNI</p></li><li><p>Pilih menu Virtual Account</p></li><li><p>Pilih Tujuan Baru</p></li><li><p>Masukkan Nomor Virtual Account</p></li><li><p>Klik tombol Lanjut</p></li><li><p>Akan ditampikan detail transaksi, Pilih rekening sumber dana</p></li><li><p>Klik tombol Lanjut</p></li><li><p>Pastikan semua informasi sudah sesuai kemudian klik Transaksi Sekarang</p></li><li><p>Masukan PIN Anda</p></li><li><p>Transaksi selesai, simpan bukti pembayaran Anda</p></li></ol>"},
        {"title": "Melalui Internet Banking", "content": "<ol start=\"1\"><li><p>Login ke internet banking Bank BNI Anda</p></li><li><p>Pilih menu Transaksi lalu klik menu Virtual Account Billing</p></li><li><p>Masukkan Nomor VA BNI lalu pilih Rekening Debit</p></li><li><p>Detail transaksi akan ditampilkan, pastikan data sudah sesuai</p></li><li><p>Masukkan respon key BNI appli 2</p></li><li><p>Transaksi sukses, simpan bukti transaksi Anda</p></li></ol>"},
        {"title": "Melalui Mobile Banking BNI", "content": "<ol start=\"1\"><li><p>Akses BNI Mobile Banking dari handphone kemudian masukkan User ID dan Password</p></li><li><p>Pilih menu Transfer</p></li><li><p>Pilih menu Virtual Account Billing kemudian pilih rekening debet</p></li><li><p>Masukkan Nomor Virtual Account</p></li><li><p>Tagihan yang harus dibayarkan akan muncul pada layar konfirmasi</p></li><li><p>Konfirmasi transaksi dan masukkan Password Transaksi</p></li><li><p>Pembayaran Anda Telah Berhasil</p></li></ol>"},
        {"title": "Melalui ATM BNI", "content": "<ol start=\"1\"><li><p>Masukkan kartu Anda</p></li><li><p>Pilih Bahasa</p></li><li><p>Masukkan PIN ATM Anda</p></li><li><p>Kemudian, pilih Menu Lainnya</p></li><li><p>Pilih Transfer dan pilih jenis rekening yang akan digunakan (Contoh: Dari rekening Tabungan)</p></li><li><p>Pilih Virtual Account Billing. Masukkan Nomor VA BNI</p></li><li><p>Tagihan yang harus dibayarkan akan muncul pada layar konfirmasi</p></li><li><p>Konfirmasi, apabila telah selesai, lanjutkan transaksi</p></li><li><p>Transaksi Anda telah selesai</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'bri_va',
    'BRI',
    'virtual_account',
    'fixed',
    5000,
    0,
    'payments/bri_logo.svg',
    'Bayar via ATM BRI, BRImo, atau Internet Banking BRI',
    $$[
        {"title": "Melalui Internet Banking", "content": "<ol start=\"1\"><li><p>Login ke internet banking Bank BRI Anda</p></li><li><p>Pilih menu Pembayaran lalu klik menu BRIVA</p></li><li><p>Pilih rekening sumber dan masukkan nomor VA BRI lalu klik Kirim</p></li><li><p>Detail transaksi akan ditampilkan, pastikan data sudah sesuai</p></li><li><p>Klik Lanjutkan</p></li><li><p>Masukkan kata sandi ibanking lalu klik Request untuk mengirim m-PIN ke nomor HP Anda</p></li><li><p>Periksa HP Anda dan masukkan m-PIN yang diterima lalu klik Kirim</p></li><li><p>Transaksi sukses, simpan bukti transaksi Anda</p></li></ol>"},
        {"title": "Melalui Aplikasi BRImo", "content": "<ol start=\"1\"><li><p>Login ke aplikasi BRImo Anda</p></li><li><p>Pilih menu BRIVA</p></li><li><p>Pilih sumber dana dan masukkan Nomor Pembayaran BRI lalu klik Lanjut</p></li><li><p>Klik Lanjut</p></li><li><p>Detail transaksi akan ditampilkan, pastikan data sudah sesuai</p></li><li><p>Klik Lanjutkan</p></li><li><p>Klik Konfirmasi</p></li><li><p>Klik Lanjut</p></li><li><p>Masukkan kata sandi ibanking Anda</p></li><li><p>Klik Lanjut</p></li><li><p>Transaksi sukses, simpan bukti transaksi Anda</p></li></ol>"},
        {"title": "Melalui ATM BRI", "content": "<ol start=\"1\"><li><p>Lakukan pembayaran melalui ATM Bank BRI</p></li><li><p>Pilih menu Transaksi Lain &gt; Pembayaran &gt; Lainnya &gt; Pilih BRIVA</p></li><li><p>Masukkan Nomor VA BRI</p></li><li><p>Pilih Ya untuk memproses pembayaran</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'mandiri_bill',
    'Mandiri Bill Payment',
    'virtual_account',
    'fixed',
    4000,
    0,
    'payments/mandiri_logo.svg',
    'Bayar via Livin'' by Mandiri, ATM Mandiri, atau Internet Banking',
    $$[
        {"title": "Melalui Livin' by Mandiri", "content": "<ol start=\"1\"><li><p>Buka aplikasi Livin' by Mandiri</p></li><li><p>Pilih menu Bayar / Pembayaran</p></li><li><p>Cari penyedia jasa Midtrans / masukkan kode biller</p></li><li><p>Masukkan Nomor Tagihan / Bill Key</p></li><li><p>Konfirmasi nominal pembayaran</p></li><li><p>Masukkan PIN Livin' Anda</p></li><li><p>Transaksi selesai, simpan bukti transaksi</p></li></ol>"},
        {"title": "Melalui ATM Mandiri", "content": "<ol start=\"1\"><li><p>Masukkan kartu ATM Mandiri dan PIN</p></li><li><p>Pilih menu Bayar/Beli &gt; Lainnya &gt; Lainnya</p></li><li><p>Pilih e-Commerce</p></li><li><p>Masukkan kode perusahaan Midtrans dan nomor tagihan</p></li><li><p>Konfirmasi detail pembayaran</p></li><li><p>Pilih YA untuk menyelesaikan transaksi</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'permata_va',
    'Permata',
    'virtual_account',
    'fixed',
    5000,
    0,
    'payments/permata_logo.svg',
    'Bayar via PermataMobile X, ATM Permata, atau ALTO/ATM Bersama',
    $$[
        {"title": "Melalui Internet Banking", "content": "<ol start=\"1\"><li><p>Login ke internet banking Bank Permata Anda</p></li><li><p>Pilih menu Pembayaran lalu klik menu Pembayaran Tagihan</p></li><li><p>Pilih menu Virtual Account</p></li><li><p>Pilih Rekening Asal lalu centang Masukkan Daftar Tagihan Baru</p></li><li><p>Masukkan Nomor VA Permata Bank lalu klik Lanjutkan</p></li><li><p>Detail transaksi akan ditampilkan, pastikan data sudah sesuai lalu klik Lanjutkan</p></li><li><p>Masukkan SMS Token Permata, lalu klik Lanjutkan</p></li><li><p>Transaksi sukses, simpan bukti transaksi Anda</p></li></ol>"},
        {"title": "Melalui ATM Permata", "content": "<ol start=\"1\"><li><p>Masukkan PIN</p></li><li><p>Pilih menu Transaksi Lainnya</p></li><li><p>Pilih menu Pembayaran</p></li><li><p>Pilih menu Pembayaran Lainnya</p></li><li><p>Pilih menu Virtual Account</p></li><li><p>Masukkan Nomor VA Permata Bank lalu tekan Benar</p></li><li><p>Pilih rekening yang menjadi sumber dana yang akan didebet</p></li><li><p>Tekan Ya untuk konfirmasi transaksi</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'cimb_va',
    'CIMB Niaga',
    'virtual_account',
    'fixed',
    5000,
    0,
    'payments/cimbniaga_logo.svg',
    'Bayar via OCTO Mobile, OCTO Clicks, atau ATM CIMB Niaga',
    $$[
        {"title": "Melalui Internet Banking", "content": "<ol start=\"1\"><li><p>Login ke CIMB clicks</p></li><li><p>Pilih menu Bayar Tagihan/Pay Bills</p></li><li><p>Pilih Rekening Sumber/Source Account dan jenis Pembayaran/Payment Type &gt; Virtual Account</p></li><li><p>Masukkan Nomor CIMBNiaga</p></li><li><p>Nomor, nama virtual account dan jumlah billing ditampilkan pada layar</p></li><li><p>Masukkan 6 digit mPIN dan tekan tombol Submit</p></li><li><p>Konfirmasi pembayaran ditampilkan pada layar</p></li></ol>"},
        {"title": "Melalui Mobile Banking", "content": "<ol start=\"1\"><li><p>Login ke Go Mobile</p></li><li><p>Pilih menu Transfer</p></li><li><p>Pilih Transfer to Other CIMB Niaga Account</p></li><li><p>Pilih rekening sumber Anda</p></li><li><p>Pilih CASA dan masukkan Nomor VA CIMBNiaga</p></li><li><p>Masukkan Nominal Nominal sesuai total pembayaran</p></li><li><p>Masukkan Mobile Banking PIN</p></li><li><p>Konfirmasi pembayaran ditampilkan pada layar</p></li></ol>"},
        {"title": "Melalui ATM CIMBNiaga", "content": "<ol start=\"1\"><li><p>Masukkan Kartu ATM dan PIN CIMB Toppers</p></li><li><p>Pilih menu Pembayaran &gt; Lanjut &gt; Virtual Account</p></li><li><p>Masukkan Nomor VA CIMBNiaga</p></li><li><p>Pilih rekening debit</p></li><li><p>Nomor, nama virtual account dan jumlah billing ditampilkan pada layar</p></li><li><p>Pilih OK untuk melakukan pembayaran</p></li><li><p>Konfirmasi pembayaran ditampilkan pada layar</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'qris',
    'QRIS',
    'ewallet',
    'percentage',
    0,
    0.0070,
    'payments/qris_logo.svg',
    'Scan QR via GoPay, OVO, DANA, ShopeePay, BCA, atau mobile banking lainnya',
    $$[
        {"title": "Melalui Semua E-Wallet", "content": "<ol start=\"1\"><li><p>Masuk ke aplikasi E-wallet Yang Anda Gunakan, kemudian klik tombol Scanner atau Bayar</p></li><li><p>Setelah itu, klik ikon Upload QR dari Galeri yang terdapat di kanan atas tampilan</p></li><li><p>Pilih gambar QRIS yang sudah Kamu Unduh dari bubbleshop.id</p></li><li><p>Klik OK untuk melanjutkan ke proses berikutnya</p></li><li><p>Klik Bayar Sekarang untuk menyelesaikan proses transaksi</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'gopay',
    'GoPay',
    'ewallet',
    'percentage',
    0,
    0.0070,
    'payments/gopay_logo.svg',
    'Bayar langsung via aplikasi GoPay atau Gojek',
    $$[
        {"title": "Cara Melakukan Pembayaran", "content": "<ol start=\"1\"><li><p>Klik tombol Bayar Sekarang untuk melanjutkan pembayaran</p></li><li><p>Sistem akan otomatis mengarahkan Anda ke aplikasi GoPay</p></li><li><p>Pastikan nominal pembayaran sesuai dengan total tagihan yang ditampilkan</p></li><li><p>Jika sudah sesuai, tekan tombol Bayar / Konfirmasi</p></li><li><p>Masukkan PIN GoPay Anda untuk memproses transaksi</p></li><li><p>Setelah transaksi berhasil, akan muncul notifikasi pembayaran sukses</p></li><li><p>Silakan simpan atau screenshot bukti pembayaran untuk keperluan arsip atau konfirmasi</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'shopeepay',
    'ShopeePay',
    'ewallet',
    'percentage',
    0,
    0.0070,
    'payments/shopeepay_logo.svg',
    'Bayar langsung via aplikasi Shopee / ShopeePay',
    $$[
        {"title": "Cara Melakukan Pembayaran", "content": "<ol start=\"1\"><li><p>Silakan klik tombol Bayar Sekarang untuk memulai proses pembayaran</p></li><li><p>Setelah tombol ditekan, sistem akan otomatis mengarahkan Anda ke aplikasi Shopee</p></li><li><p>Pastikan jumlah pembayaran sesuai dengan total tagihan yang tertera di aplikasi Shopee</p></li><li><p>Jika nominal sudah benar, lanjutkan dengan menekan tombol Bayar / Konfirmasi</p></li><li><p>Masukkan PIN ShopeePay Anda untuk menyelesaikan transaksi</p></li><li><p>Setelah pembayaran berhasil, akan muncul notifikasi transaksi sukses</p></li><li><p>Silakan simpan atau screenshot bukti pembayaran sebagai arsip atau jika diperlukan untuk konfirmasi</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'indomaret',
    'Indomaret',
    'convenience_store',
    'fixed',
    5000,
    0,
    'payments/indomaret_logo.svg',
    'Bayar di kasir Indomaret / Ceriamart terdekat dengan menunjukkan kode pembayaran',
    $$[
        {"title": "Melalui Gerai Indomaret", "content": "<ol start=\"1\"><li><p>Datang ke gerai Indomaret</p></li><li><p>Sampaikan ke kasir ingin melakukan pembayaran</p></li><li><p>Berikan kode bayar Indomaret ke kasir</p></li><li><p>Bayar sesuai jumlah yang diinfokan oleh kasir</p></li><li><p>Simpan struk bukti pembayaran Anda</p></li></ol>"}
    ]$$::jsonb,
    true
),
(
    'alfamart',
    'Alfamart',
    'convenience_store',
    'fixed',
    5000,
    0,
    'payments/alfamart_logo.svg',
    'Bayar di kasir Alfamart, Alfamidi, atau Dan+Dan terdekat',
    $$[
        {"title": "Melalui Gerai Alfamart", "content": "<ol start=\"1\"><li><p>Datang ke gerai Alfamart</p></li><li><p>Sampaikan ke kasir ingin melakukan pembayaran</p></li><li><p>Berikan kode bayar ke kasir</p></li><li><p>Bayar sesuai jumlah yang diinfokan oleh kasir</p></li><li><p>Simpan struk bukti pembayaran Anda</p></li></ol>"}
    ]$$::jsonb,
    true
)
ON CONFLICT (code) DO UPDATE SET
    image = EXCLUDED.image,
    name = EXCLUDED.name,
    category = EXCLUDED.category,
    fee_type = EXCLUDED.fee_type,
    fee_amount = EXCLUDED.fee_amount,
    fee_percent = EXCLUDED.fee_percent,
    description = EXCLUDED.description,
    instructions = EXCLUDED.instructions;
