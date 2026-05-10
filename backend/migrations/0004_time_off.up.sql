CREATE TABLE time_off (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    start_date  date NOT NULL,
    end_date    date NOT NULL,
    type        text NOT NULL,
    notes       text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT time_off_date_order CHECK (end_date >= start_date),
    CONSTRAINT time_off_type CHECK (type IN ('vacation','sick','holiday','other'))
);

CREATE INDEX time_off_person_idx ON time_off (person_id, start_date, end_date);
