CREATE TABLE milestones (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    phase_id    uuid,
    name        text NOT NULL,
    date        date NOT NULL,
    end_date    date,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX milestones_project_idx ON milestones (project_id);
