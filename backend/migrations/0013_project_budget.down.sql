ALTER TABLE projects
    DROP COLUMN IF EXISTS budget_type,
    DROP COLUMN IF EXISTS budget_total,
    DROP COLUMN IF EXISTS budget_priority;
