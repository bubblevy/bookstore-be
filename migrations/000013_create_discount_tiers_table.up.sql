CREATE TABLE IF NOT EXISTS discount_tiers (
    id SERIAL PRIMARY KEY,
    discount_id INT NOT NULL,
    min_spend NUMERIC(12, 2) NOT NULL CHECK (min_spend >= 0),
    reward_type VARCHAR(20) NOT NULL CHECK (reward_type IN ('PERCENTAGE', 'FIXED')),
    reward_value NUMERIC(12, 2) NOT NULL CHECK (reward_value > 0),
    max_reward NUMERIC(12, 2) CHECK (max_reward > 0),
    CONSTRAINT fk_discount_tiers_discount FOREIGN KEY (discount_id) 
        REFERENCES discounts(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_discount_tiers_discount_id ON discount_tiers(discount_id);