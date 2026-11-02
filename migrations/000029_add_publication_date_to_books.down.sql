ALTER TABLE books ADD COLUMN IF NOT EXISTS publication_year SMALLINT CHECK (publication_year >= 1000 AND publication_year <= 2100);

-- Populate publication_year from publication_date
UPDATE books 
SET publication_year = EXTRACT(YEAR FROM publication_date)::SMALLINT 
WHERE publication_date IS NOT NULL;

-- Drop publication_date
ALTER TABLE books DROP COLUMN IF EXISTS publication_date;
