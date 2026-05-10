-- Link a person row to a Levitate (OIDC) user so signing in auto-creates a
-- schedulable Person. user_id remains nullable for contractors who don't log in.
-- Postgres treats NULLs as distinct under UNIQUE, so multiple unlinked people
-- (NULL user_id) are still allowed.
ALTER TABLE people
    ADD COLUMN user_id uuid REFERENCES users(id) ON DELETE SET NULL UNIQUE;
