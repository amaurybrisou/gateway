BEGIN;

-- Add a new temporary column
ALTER TABLE "service"
ADD COLUMN "required_roles_temp" JSONB;

-- Copy values from the original column to the temporary column
UPDATE "service"
SET "required_roles_temp" = ARRAY_TO_JSON("required_roles"::TEXT[]);

-- Drop the original column
ALTER TABLE "service"
DROP COLUMN "required_roles";

-- Rename the temporary column to the original column name
ALTER TABLE "service"
RENAME COLUMN "required_roles_temp" TO "required_roles";

COMMIT;
