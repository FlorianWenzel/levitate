DROP INDEX IF EXISTS projects_project_code_key;

ALTER TABLE projects
    DROP COLUMN IF EXISTS project_code;
