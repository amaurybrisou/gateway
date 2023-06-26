CREATE EXTENSION IF NOT EXISTS "pgcrypto";
-- file: 20220621120000_create_tables.up.sql

-- Create the "user" table
CREATE TABLE "user" (
    "id" UUID PRIMARY KEY,
    "email" TEXT NOT NULL UNIQUE,
    "avatar" TEXT NOT NULL,
    "firstname" TEXT NOT NULL,
    "lastname" TEXT NOT NULL,
    "role" TEXT NOT NULL DEFAULT 'USER',
    "stripe_key" TEXT,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP,
    "deleted_at" TIMESTAMP
);

CREATE TABLE "access_token" (
    "user_id" UUID REFERENCES "user"("id") UNIQUE,
    "external_id" TEXT NOT NULL UNIQUE,
    "token" TEXT NOT NULL,
    "expires_at" TIMESTAMP NOT NULL
);

-- Create the "service" table
CREATE TABLE "service" (
    "id" UUID PRIMARY KEY,
    "name" TEXT NOT NULL UNIQUE,
    "domain" TEXT UNIQUE,
    "prefix" TEXT NOT NULL UNIQUE,
    "host" TEXT NOT NULL,
    "required_roles" JSONB NOT NULL,
    "costs" JSONB NOT NULL,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP,
    "deleted_at" TIMESTAMP
);

-- Create the "user_role" table
CREATE TABLE "user_role" (
    "user_id" UUID REFERENCES "user"("id"),
    "role" TEXT NOT NULL,
    "expiration_time" TIMESTAMP NOT NULL,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "updated_at" TIMESTAMP,
    "deleted_at" TIMESTAMP
);

-- Create the "user_payment" table
CREATE TABLE "user_payment" (
    "id" UUID PRIMARY KEY,
    "user_id" UUID REFERENCES "user"("id"),
    "service_id" UUID REFERENCES "service"("id"),
    "amount" NUMERIC NOT NULL,
    "duration" int NOT NULL,
    "created_at" TIMESTAMP DEFAULT NOW() NOT NULL,
    "status" TEXT NOT NULL,
    "updated_at" TIMESTAMP,
    "deleted_at" TIMESTAMP
);
