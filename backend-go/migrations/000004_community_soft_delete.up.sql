ALTER TABLE cooking_records
    ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE';

CREATE INDEX idx_comments_active_created
    ON comments(recipe_version_id, created_at DESC)
    WHERE status = 'ACTIVE';

CREATE INDEX idx_cooking_records_active_created
    ON cooking_records(recipe_version_id, created_at DESC)
    WHERE status = 'ACTIVE';
