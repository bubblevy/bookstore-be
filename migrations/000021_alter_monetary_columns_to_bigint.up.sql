-- 1. Alter books price & original_price to BIGINT
ALTER TABLE books
    ALTER COLUMN price TYPE BIGINT USING ROUND(price)::BIGINT;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM information_schema.columns 
        WHERE table_name = 'books' AND column_name = 'original_price'
    ) THEN
        ALTER TABLE books ALTER COLUMN original_price TYPE BIGINT USING ROUND(original_price)::BIGINT;
    END IF;
END $$;

-- 2. Alter orders monetary fields to BIGINT
ALTER TABLE orders
    ALTER COLUMN items_subtotal TYPE BIGINT USING ROUND(items_subtotal)::BIGINT,
    ALTER COLUMN shipping_cost TYPE BIGINT USING ROUND(shipping_cost)::BIGINT,
    ALTER COLUMN discount_amount TYPE BIGINT USING ROUND(discount_amount)::BIGINT,
    ALTER COLUMN admin_fee TYPE BIGINT USING ROUND(admin_fee)::BIGINT,
    ALTER COLUMN total_amount TYPE BIGINT USING ROUND(total_amount)::BIGINT;

-- 3. Alter order_items price & subtotal to BIGINT
ALTER TABLE order_items
    ALTER COLUMN price TYPE BIGINT USING ROUND(price)::BIGINT,
    ALTER COLUMN subtotal TYPE BIGINT USING ROUND(subtotal)::BIGINT;

-- 4. Alter discounts monetary fields to BIGINT
ALTER TABLE discounts
    ALTER COLUMN discount_value TYPE BIGINT USING ROUND(discount_value)::BIGINT,
    ALTER COLUMN min_purchase TYPE BIGINT USING ROUND(min_purchase)::BIGINT,
    ALTER COLUMN max_discount_amount TYPE BIGINT USING ROUND(max_discount_amount)::BIGINT;

-- 5. Alter discount_tiers monetary fields to BIGINT
ALTER TABLE discount_tiers
    ALTER COLUMN min_spend TYPE BIGINT USING ROUND(min_spend)::BIGINT,
    ALTER COLUMN reward_value TYPE BIGINT USING ROUND(reward_value)::BIGINT,
    ALTER COLUMN max_reward TYPE BIGINT USING ROUND(max_reward)::BIGINT;
