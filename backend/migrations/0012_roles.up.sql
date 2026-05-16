CREATE TABLE roles (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 text NOT NULL,
    default_hourly_rate  numeric(12, 3) NOT NULL DEFAULT 0,
    cost_rate_history    jsonb NOT NULL DEFAULT '[]'::jsonb,
    float_id             bigint UNIQUE,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX roles_name_idx ON roles (lower(name));
