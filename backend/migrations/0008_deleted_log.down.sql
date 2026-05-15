DROP INDEX IF EXISTS time_off_float_id_idx;
DROP INDEX IF EXISTS assignments_float_id_idx;
ALTER TABLE time_off    DROP COLUMN IF EXISTS float_id;
ALTER TABLE assignments DROP COLUMN IF EXISTS float_id;
DROP TRIGGER IF EXISTS time_off_deleted_log ON time_off;
DROP TRIGGER IF EXISTS assignments_deleted_log ON assignments;
DROP FUNCTION IF EXISTS deleted_log_log_time_off();
DROP FUNCTION IF EXISTS deleted_log_log_assignment();
DROP TABLE IF EXISTS deleted_log;
