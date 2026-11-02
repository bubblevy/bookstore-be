ALTER TABLE books ADD COLUMN IF NOT EXISTS publication_date DATE;

-- Populate existing rows where publication_year is set
UPDATE books 
SET publication_date = make_date(publication_year, 1, 1) 
WHERE publication_year IS NOT NULL AND publication_date IS NULL;

-- Drop publication_year
ALTER TABLE books DROP COLUMN IF EXISTS publication_year;
