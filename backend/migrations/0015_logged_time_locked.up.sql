-- 0015_logged_time_locked: Float-parity `locked` and `locked_date` fields.
-- Mirrors Float's LoggedTime schema (https://developer.float.com/reference/logged-time):
-- `locked` is a server-managed boolean indicating the entry is no longer
-- editable; `locked_date` records when the lock was applied. Float derives
-- locked from project/phase/task lock settings; clients of /logged-time may
-- not toggle these fields directly — the API layer ignores client-supplied
-- values and uses dedicated admin endpoints to flip them server-side.

ALTER TABLE logged_time
    ADD COLUMN locked      boolean     NOT NULL DEFAULT false,
    ADD COLUMN locked_date timestamptz;
