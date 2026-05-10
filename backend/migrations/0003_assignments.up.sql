CREATE TABLE assignments (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    project_id      uuid NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    start_date      date NOT NULL,
    end_date        date NOT NULL,
    hours_per_day   numeric(4,2) NOT NULL DEFAULT 8,
    notes           text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT assignments_date_order CHECK (end_date >= start_date),
    CONSTRAINT assignments_hours_range CHECK (hours_per_day > 0 AND hours_per_day <= 24)
);

CREATE INDEX assignments_person_idx ON assignments (person_id, start_date, end_date);
CREATE INDEX assignments_project_idx ON assignments (project_id);
CREATE INDEX assignments_range_idx ON assignments USING gist (daterange(start_date, end_date, '[]'));
