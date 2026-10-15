ALTER TABLE correction_proposals
    DROP COLUMN IF EXISTS rejection_reason;

ALTER TABLE recipe_versions
    DROP COLUMN IF EXISTS title;
