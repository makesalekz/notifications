-- Modify "notification_data" table
ALTER TABLE "notification_data" ADD COLUMN "project" character varying NULL;
-- Modify "notifications" table
ALTER TABLE "notifications" ADD COLUMN "project_id" bigint NULL;
