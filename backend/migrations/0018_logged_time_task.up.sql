-- 0018_logged_time_task: Float-parity `task_name` and `task_meta_id` fields.
-- Float's LoggedTime schema (https://developer.float.com/reference/logged-time)
-- exposes `task_name` (display name of the associated task) and
-- `task_meta_id` (unique identifier for the linked task metadata). Both are
-- free-form strings on Float's side and nullable when no task is associated;
-- we mirror that here as plain nullable text columns.

ALTER TABLE logged_time
    ADD COLUMN task_name    text,
    ADD COLUMN task_meta_id text;
