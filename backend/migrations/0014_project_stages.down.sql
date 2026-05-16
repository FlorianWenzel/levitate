DROP INDEX IF EXISTS projects_stage_id_idx;
ALTER TABLE projects DROP COLUMN IF EXISTS stage_id;
DROP TABLE IF EXISTS project_stages;
