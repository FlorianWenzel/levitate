-- 0001_init: users (OIDC-synced), people (schedulable humans), audit_log.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sub         text NOT NULL UNIQUE,
    email       text NOT NULL,
    name        text NOT NULL DEFAULT '',
    role        text NOT NULL DEFAULT 'member',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE people (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  text NOT NULL,
    email                 text NOT NULL DEFAULT '',
    role                  text NOT NULL DEFAULT '',
    weekly_capacity_hours numeric(5,2) NOT NULL DEFAULT 40,
    archived_at           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now(),
    updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX people_active_idx ON people (archived_at) WHERE archived_at IS NULL;

CREATE TABLE audit_log (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
    action        text NOT NULL,
    entity_type   text NOT NULL,
    entity_id     uuid,
    diff_json     jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_log_entity_idx ON audit_log (entity_type, entity_id);
