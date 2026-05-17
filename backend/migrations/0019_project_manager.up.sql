-- 0019_project_manager: Float-parity `project_manager` and `all_pms_schedule`
-- fields on Project (https://developer.float.com/swagger-api-v3.yaml).
-- `project_manager` is the assigned project manager's name (Float exposes it
-- as a string); `all_pms_schedule` is the flag controlling whether all PMs may
-- schedule on the project.

ALTER TABLE projects
    ADD COLUMN project_manager   text,
    ADD COLUMN all_pms_schedule  boolean NOT NULL DEFAULT false;
