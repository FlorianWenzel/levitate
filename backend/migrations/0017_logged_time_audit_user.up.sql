-- 0017_logged_time_audit_user: Float-parity `created_by` and `modified_by`.
-- Float's LoggedTime schema (https://developer.float.com/reference/logged-time)
-- exposes `created_by` (user ID of creator) and `modified_by` (user ID of last
-- modifier) alongside the `created` / `modified` timestamps. We mirror that
-- here as nullable FKs to the levitate users table: nullable because rows
-- created before this migration (or imported from Float) have no local user
-- to attribute them to. ON DELETE SET NULL preserves the row's audit history
-- when the user is later removed.

ALTER TABLE logged_time
    ADD COLUMN created_by  uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN modified_by uuid REFERENCES users(id) ON DELETE SET NULL;
