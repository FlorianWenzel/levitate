-- ===== users =====

-- name: UpsertUser :one
INSERT INTO users (sub, email, name, role)
VALUES ($1, $2, $3, $4)
ON CONFLICT (sub) DO UPDATE SET
    email = EXCLUDED.email,
    name = EXCLUDED.name,
    role = EXCLUDED.role,
    updated_at = now()
RETURNING *;

-- name: GetUserBySub :one
SELECT * FROM users WHERE sub = $1;

-- name: EnsurePersonForUser :exec
INSERT INTO people (user_id, name, email, weekly_capacity_hours)
VALUES ($1, $2, $3, 40)
ON CONFLICT (user_id) DO NOTHING;

-- ===== people =====

-- name: ListPeople :many
SELECT * FROM people
WHERE
    (sqlc.arg(include_archived)::boolean OR archived_at IS NULL)
ORDER BY name ASC;

-- name: GetPerson :one
SELECT * FROM people WHERE id = $1;

-- name: CreatePerson :one
INSERT INTO people (name, email, role, weekly_capacity_hours)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdatePerson :one
UPDATE people
SET name = $2,
    email = $3,
    role = $4,
    weekly_capacity_hours = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ArchivePerson :one
UPDATE people
SET archived_at = now(), updated_at = now()
WHERE id = $1 AND archived_at IS NULL
RETURNING *;

-- name: UnarchivePerson :one
UPDATE people
SET archived_at = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- ===== projects =====

-- name: ListProjects :many
SELECT * FROM projects
WHERE
    (sqlc.arg(include_archived)::boolean OR archived_at IS NULL)
ORDER BY name ASC;

-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: CreateProject :one
INSERT INTO projects (name, client, color, notes, billable)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = $2,
    client = $3,
    color = $4,
    notes = $5,
    billable = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ArchiveProject :one
UPDATE projects
SET archived_at = now(), updated_at = now()
WHERE id = $1 AND archived_at IS NULL
RETURNING *;

-- name: UnarchiveProject :one
UPDATE projects
SET archived_at = NULL, updated_at = now()
WHERE id = $1
RETURNING *;

-- ===== assignments =====

-- name: ListAssignmentsInRange :many
SELECT a.*
FROM assignments a
WHERE a.start_date <= sqlc.arg(to_date)::date
  AND a.end_date >= sqlc.arg(from_date)::date
ORDER BY a.start_date ASC, a.id ASC;

-- name: GetAssignment :one
SELECT * FROM assignments WHERE id = $1;

-- name: CreateAssignment :one
INSERT INTO assignments (person_id, project_id, start_date, end_date, hours_per_day, notes)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateAssignment :one
UPDATE assignments
SET person_id     = $2,
    project_id    = $3,
    start_date    = $4,
    end_date      = $5,
    hours_per_day = $6,
    notes         = $7,
    updated_at    = now()
WHERE id = $1
RETURNING *;

-- name: DeleteAssignment :exec
DELETE FROM assignments WHERE id = $1;

-- ===== time_off =====

-- name: ListTimeOffInRange :many
SELECT t.*
FROM time_off t
WHERE t.start_date <= sqlc.arg(to_date)::date
  AND t.end_date   >= sqlc.arg(from_date)::date
ORDER BY t.start_date ASC, t.id ASC;

-- name: GetTimeOff :one
SELECT * FROM time_off WHERE id = $1;

-- name: CreateTimeOff :one
INSERT INTO time_off (person_id, start_date, end_date, type, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateTimeOff :one
UPDATE time_off
SET person_id = $2,
    start_date = $3,
    end_date = $4,
    type = $5,
    notes = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTimeOff :exec
DELETE FROM time_off WHERE id = $1;
