-- Create "greeters" table
CREATE TABLE "greeters" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(255) NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_greeters_deleted_at" to table: "greeters"
CREATE INDEX "idx_greeters_deleted_at" ON "greeters" ("deleted_at");
-- Set comment to column: "name" on table: "greeters"
COMMENT ON COLUMN "greeters"."name" IS '名称';
