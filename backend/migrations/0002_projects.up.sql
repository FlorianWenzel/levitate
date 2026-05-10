CREATE TABLE projects (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    client      text NOT NULL DEFAULT '',
    color       text NOT NULL DEFAULT '#64748B',
    notes       text NOT NULL DEFAULT '',
    archived_at timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX projects_active_idx ON projects (archived_at) WHERE archived_at IS NULL;
