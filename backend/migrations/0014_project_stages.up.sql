-- Float-parity project stages. Float exposes project stages via /stages and
-- references them per-project via project.stage_id. The stage's `active` flag
-- determines whether a stage maps to an "active" or "archived" effective
-- project status; when a project has stage_id set, it takes precedence over
-- the legacy archived_at-derived status.
CREATE TABLE project_stages (
    id         bigint PRIMARY KEY,
    name       text NOT NULL,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO project_stages (id, name, active) VALUES
    (1, 'In Progress', true),
    (2, 'Tentative',   true),
    (3, 'On Hold',     true),
    (4, 'Completed',   false),
    (5, 'Cancelled',   false);

ALTER TABLE projects
    ADD COLUMN stage_id bigint;

CREATE INDEX projects_stage_id_idx ON projects (stage_id) WHERE stage_id IS NOT NULL;
