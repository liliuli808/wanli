DROP INDEX IF EXISTS idx_cooking_records_active_created;
DROP INDEX IF EXISTS idx_comments_active_created;

ALTER TABLE cooking_records
    DROP COLUMN IF EXISTS status;
