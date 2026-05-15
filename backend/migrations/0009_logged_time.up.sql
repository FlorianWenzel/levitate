-- 0009_logged_time: timesheet entries (Float parity).
-- Mirrors Float's /logged-time API: per-person, per-day actual hours against
-- a project. Float derives billability from the project/phase/task; we follow
-- the same rule and snapshot the project's billable flag at write time.

CREATE TABLE logged_time (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id   uuid NOT NULL REFERENCES people(id) ON DELETE CASCADE,
    date        date NOT NULL,
    hours       numeric(5,2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    billable    boolean NOT NULL DEFAULT true,
    notes       text NOT NULL DEFAULT '',
    project_id  uuid REFERENCES projects(id) ON DELETE SET NULL,
    float_id    bigint,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX logged_time_person_date_idx ON logged_time (person_id, date);
CREATE INDEX logged_time_project_idx ON logged_time (project_id);
CREATE INDEX logged_time_float_id_idx ON logged_time (float_id) WHERE float_id IS NOT NULL;

-- Mirror the deletion trigger used for assignments / time_off so the public
-- /api/deleted/logged-time feed (registered in 0008) picks up local deletions.
CREATE OR REPLACE FUNCTION deleted_log_log_logged_time() RETURNS trigger AS $$
BEGIN
    INSERT INTO deleted_log (entity_type, entity_id) VALUES ('logged_time', OLD.id);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER logged_time_deleted_log
    AFTER DELETE ON logged_time
    FOR EACH ROW EXECUTE FUNCTION deleted_log_log_logged_time();
