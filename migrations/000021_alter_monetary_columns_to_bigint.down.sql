-- 1. Revert discount_tiers monetary fields to NUMERIC(12, 2)
ALTER TABLE discount_tiers
    ALTER COLUMN min_spend TYPE NUMERIC(12, 2) USING min_spend::NUMERIC(12, 2),
    ALTER COLUMN reward_value TYPE NUMERIC(12, 2) USING reward_value::NUMERIC(12, 2),
    ALTER COLUMN max_reward TYPE NUMERIC(12, 2) USING max_reward::NUMERIC(12, 2);

-- 2. Revert discounts monetary fields to NUMERIC(12, 2)
ALTER TABLE discounts
    ALTER COLUMN discount_value TYPE NUMERIC(12, 2) USING discount_value::NUMERIC(12, 2),
    ALTER COLUMN min_purchase TYPE NUMERIC(12, 2) USING min_purchase::NUMERIC(12, 2),
    ALTER COLUMN max_discount_amount TYPE NUMERIC(12, 2) USING max_discount_amount::NUMERIC(12, 2);

-- 3. Revert order_items price & subtotal to NUMERIC(15, 2)
ALTER TABLE order_items
    ALTER COLUMN price TYPE NUMERIC(15, 2) USING price::NUMERIC(15, 2),
    ALTER COLUMN subtotal TYPE NUMERIC(15, 2) USING subtotal::NUMERIC(15, 2);

-- 4. Revert orders monetary fields to NUMERIC(15, 2)
ALTER TABLE orders
    ALTER COLUMN items_subtotal TYPE NUMERIC(15, 2) USING items_subtotal::NUMERIC(15, 2),
    ALTER COLUMN shipping_cost TYPE NUMERIC(15, 2) USING shipping_cost::NUMERIC(15, 2),
    ALTER COLUMN discount_amount TYPE NUMERIC(15, 2) USING discount_amount::NUMERIC(15, 2),
    ALTER COLUMN admin_fee TYPE NUMERIC(15, 2) USING admin_fee::NUMERIC(15, 2),
    ALTER COLUMN total_amount TYPE NUMERIC(15, 2) USING total_amount::NUMERIC(15, 2);

-- 5. Revert books price & original_price to NUMERIC(12, 2)
ALTER TABLE books
    ALTER COLUMN price TYPE NUMERIC(12, 2) USING price::NUMERIC(12, 2);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'books' AND column_name = 'original_price'
    ) THEN
        ALTER TABLE books ALTER COLUMN original_price TYPE NUMERIC(12, 2) USING original_price::NUMERIC(12, 2);
    END IF;
END $$;
