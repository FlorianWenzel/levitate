ALTER TABLE projects
    ADD COLUMN project_code text;

CREATE UNIQUE INDEX projects_project_code_key
    ON projects (project_code)
    WHERE project_code IS NOT NULL AND project_code <> '';
