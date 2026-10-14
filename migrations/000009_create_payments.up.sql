CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    midtrans_transaction_id VARCHAR(100),
    gross_amount NUMERIC(12, 2) NOT NULL CHECK (gross_amount >= 0),
    payment_type VARCHAR(50),              -- qris, bank_transfer, gopay, other
    transaction_status VARCHAR(50) NOT NULL, -- capture, settlement, pending, deny, expire, cancel
    fraud_status VARCHAR(50),              -- accept, challenge, deny
    snap_token VARCHAR(255),
    snap_redirect_url TEXT,
    payment_response_payload JSONB,        -- Payload webhook from Midtrans
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT fk_payments_order FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

-- Indexing 
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_midtrans_transaction_id ON payments(midtrans_transaction_id);
CREATE INDEX IF NOT EXISTS idx_payments_transaction_status ON payments(transaction_status);