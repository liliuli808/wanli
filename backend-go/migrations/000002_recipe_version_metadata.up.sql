ALTER TABLE recipe_versions
    ADD COLUMN title VARCHAR(100);

UPDATE recipe_versions AS version
SET title = recipe.title
FROM recipes AS recipe
WHERE recipe.id = version.recipe_id;

ALTER TABLE recipe_versions
    ALTER COLUMN title SET NOT NULL;

ALTER TABLE correction_proposals
    ADD COLUMN rejection_reason VARCHAR(255);
