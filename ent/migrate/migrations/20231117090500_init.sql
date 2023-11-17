-- Drop index "last_read_notifications_user_id_key" from table: "last_read_notifications"
DROP INDEX "last_read_notifications_user_id_key";
-- Modify "last_read_notifications" table
ALTER TABLE "last_read_notifications" ADD COLUMN "type" character varying NOT NULL DEFAULT 'COMMON';
-- Create index "lastreadnotification_user_id_type" to table: "last_read_notifications"
CREATE UNIQUE INDEX "lastreadnotification_user_id_type" ON "last_read_notifications" ("user_id", "type");
