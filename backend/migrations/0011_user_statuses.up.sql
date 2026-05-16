CREATE TABLE user_statuses (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id       uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    status_type_id  smallint NOT NULL,
    status_name     text NOT NULL DEFAULT '',
    start_date      date NOT NULL,
    end_date        date NOT NULL,
    repeat_state    smallint NOT NULL DEFAULT 0,
    repeat_end_date date,
    float_id        bigint UNIQUE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_statuses_type_check CHECK (status_type_id BETWEEN 1 AND 4),
    CONSTRAINT user_statuses_date_check CHECK (end_date >= start_date),
    CONSTRAINT user_statuses_repeat_check CHECK (repeat_state BETWEEN 0 AND 4)
);

CREATE INDEX user_statuses_person_idx ON user_statuses (person_id);
CREATE INDEX user_statuses_date_idx ON user_statuses (start_date, end_date);
