BEGIN;

-- Add a new temporary column
ALTER TABLE "service"
ADD COLUMN "required_roles_temp" TEXT[];

-- Copy values from the original column to the temporary column
UPDATE "service"
SET "required_roles_temp" = ARRAY(SELECT jsonb_array_elements_text("required_roles"));

-- Drop the original column
ALTER TABLE "service"
DROP COLUMN "required_roles";

-- Rename the temporary column to the original column name
ALTER TABLE "service"
RENAME COLUMN "required_roles_temp" TO "required_roles";

COMMIT;