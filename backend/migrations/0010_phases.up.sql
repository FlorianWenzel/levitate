CREATE TABLE phases (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id          uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name                text NOT NULL,
    color               text NOT NULL DEFAULT '',
    notes               text NOT NULL DEFAULT '',
    start_date          date,
    end_date            date,
    budget_total        numeric(12, 2) NOT NULL DEFAULT 0,
    default_hourly_rate numeric(12, 2) NOT NULL DEFAULT 0,
    billable            boolean NOT NULL DEFAULT true,
    status              smallint NOT NULL DEFAULT 2,
    archived_at         timestamptz,
    float_id            bigint UNIQUE,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX phases_project_idx ON phases (project_id);

ALTER TABLE milestones
    ADD CONSTRAINT milestones_phase_id_fkey
    FOREIGN KEY (phase_id) REFERENCES phases(id) ON DELETE SET NULL;
