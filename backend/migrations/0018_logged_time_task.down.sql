ALTER TABLE logged_time
    DROP COLUMN IF EXISTS task_meta_id,
    DROP COLUMN IF EXISTS task_name;
