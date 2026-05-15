-- 0008_deleted_log: integration sync delete log (Float parity).
-- Mirrors Float's GET /deleted/* endpoints so external consumers can
-- reconcile deletions without a full re-sync. Records expire after 72 hours.

CREATE TABLE deleted_log (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type text NOT NULL CHECK (entity_type IN ('assignment','time_off','logged_time')),
    entity_id   uuid NOT NULL,
    deleted_at  timestamptz NOT NULL DEFAULT now()
);

-- Lookup by (entity_type, deleted_at) for the cursor-paginated public API,
-- and by entity_id for de-dup checks.
CREATE INDEX deleted_log_type_time_idx ON deleted_log (entity_type, deleted_at, id);
CREATE INDEX deleted_log_entity_idx ON deleted_log (entity_id);

-- Capture deletions of the local entities Float mirrors. These triggers write
-- the deleted row's UUID into deleted_log with the current timestamp; the
-- public API and the Float import sync read from this table to reconcile.
CREATE OR REPLACE FUNCTION deleted_log_log_assignment() RETURNS trigger AS $$
BEGIN
    INSERT INTO deleted_log (entity_type, entity_id) VALUES ('assignment', OLD.id);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION deleted_log_log_time_off() RETURNS trigger AS $$
BEGIN
    INSERT INTO deleted_log (entity_type, entity_id) VALUES ('time_off', OLD.id);
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER assignments_deleted_log
    AFTER DELETE ON assignments
    FOR EACH ROW EXECUTE FUNCTION deleted_log_log_assignment();

CREATE TRIGGER time_off_deleted_log
    AFTER DELETE ON time_off
    FOR EACH ROW EXECUTE FUNCTION deleted_log_log_time_off();

-- Track the upstream Float entity ID on imported rows so the next Float
-- import can reconcile remote deletions (Float returns int IDs in its
-- /deleted/* feed; we need that mapping to find the local UUIDs to delete).
ALTER TABLE assignments ADD COLUMN float_id bigint;
ALTER TABLE time_off    ADD COLUMN float_id bigint;
CREATE INDEX assignments_float_id_idx ON assignments (float_id) WHERE float_id IS NOT NULL;
CREATE INDEX time_off_float_id_idx    ON time_off    (float_id) WHERE float_id IS NOT NULL;
