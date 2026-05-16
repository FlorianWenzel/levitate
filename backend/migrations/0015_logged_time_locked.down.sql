ALTER TABLE logged_time
    DROP COLUMN IF EXISTS locked_date,
    DROP COLUMN IF EXISTS locked;
