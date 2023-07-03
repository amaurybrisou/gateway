CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- file: 20220621120000_create_tables.up.sql

-- Create the "user" table
CREATE TABLE "user" (
    "id" UUID PRIMARY KEY,
    "external_id" TEXT NOT NULL,
    "email" TEXT NOT NULL UNIQUE,
    "avatar" TEXT NOT NULL,
    "firstname" TEXT,
    "lastname" TEXT,
    "password" TEXT NOT NULL,
    "role" TEXT NOT NULL DEFAULT 'USER',
    "stripe_key" TEXT,
    "is_new" BOOLEAN default true,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);

-- Create the "service" table
CREATE TABLE "service" (
    "id" UUID PRIMARY KEY,
    "name" TEXT NOT NULL UNIQUE,
    "description" TEXT NOT NULL,
    "status" TEXT,
    "domain" TEXT UNIQUE,
    "prefix" TEXT NOT NULL UNIQUE,
    "host" TEXT NOT NULL,
    "image_url" TEXT,
    "required_roles" JSONB NOT NULL,
    "pricing_table_key" TEXT NOT NULL,
    "pricing_table_publishable_key" TEXT NOT NULL,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP DEFAULT NOW(),
    "deleted_at" TIMESTAMP
);

CREATE TABLE "user_role" (
    "user_id" UUID,
    "subscription_id" TEXT NOT NULL,
    "role" TEXT NOT NULL,
    "expires_at" TIMESTAMP,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP DEFAULT NOW(),
    "deleted_at" TIMESTAMP,
    PRIMARY KEY ("user_id", "role"),
    FOREIGN KEY ("user_id") REFERENCES "user"("id")
);

-- temporary password: w9oHDCAlPxT12WbH
INSERT INTO "user" ("id", "external_id", "email", "avatar", "firstname", "lastname", "password", "role", "stripe_key", "created_at", "updated_at", "deleted_at") VALUES
('d179fd63-0b0f-4f35-9f15-f903a394c035',	'',	'gateway@gateway.com',	'',	'Seigneur',	'',	'$2a$10$gS.0eGpOQMZ5sJn0X2m4teBgA6zD9OL2bTY/Y2D/Ui8fZN3.65bVO',	'ADMIN',	NULL,	'2023-06-30 06:53:45.886516',	NULL,	NULL);