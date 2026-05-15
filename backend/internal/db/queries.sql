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

-- ===== milestones =====

-- name: ListMilestonesByProject :many
SELECT * FROM milestones
WHERE project_id = $1
ORDER BY date ASC, id ASC;

-- name: GetMilestone :one
SELECT * FROM milestones WHERE id = $1;

-- name: CreateMilestone :one
INSERT INTO milestones (project_id, phase_id, name, date, end_date)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateMilestone :one
UPDATE milestones
SET phase_id = $2,
    name = $3,
    date = $4,
    end_date = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteMilestone :exec
DELETE FROM milestones WHERE id = $1;

-- ===== logged_time =====

-- name: ListLoggedTime :many
SELECT * FROM logged_time
WHERE
    (sqlc.narg(person_id)::uuid IS NULL OR person_id = sqlc.narg(person_id)::uuid)
    AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id)::uuid)
    AND (sqlc.narg(date_from)::date IS NULL OR date >= sqlc.narg(date_from)::date)
    AND (sqlc.narg(date_to)::date IS NULL OR date <= sqlc.narg(date_to)::date)
ORDER BY date ASC, id ASC;

-- name: GetLoggedTime :one
SELECT * FROM logged_time WHERE id = $1;

-- name: CreateLoggedTime :one
INSERT INTO logged_time (person_id, date, hours, billable, notes, project_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateLoggedTime :one
UPDATE logged_time
SET date       = $2,
    hours      = $3,
    billable   = $4,
    notes      = $5,
    project_id = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteLoggedTime :exec
DELETE FROM logged_time WHERE id = $1;

-- name: DeleteLoggedTimeByFloatID :exec
DELETE FROM logged_time WHERE float_id = ANY(sqlc.arg(float_ids)::bigint[]);

-- name: SetLoggedTimeFloatID :exec
UPDATE logged_time SET float_id = $2 WHERE id = $1;

-- name: GetLoggedTimeByFloatID :one
SELECT * FROM logged_time WHERE float_id = $1;

-- ===== deleted_log =====

-- name: InsertDeletedLog :one
INSERT INTO deleted_log (entity_type, entity_id)
VALUES ($1, $2)
RETURNING *;

-- name: ListDeletedLog :many
SELECT * FROM deleted_log
WHERE entity_type = sqlc.arg(entity_type)::text
  AND deleted_at > now() - INTERVAL '72 hours'
  AND (
        sqlc.arg(after_ts)::timestamptz IS NULL
     OR deleted_at > sqlc.arg(after_ts)::timestamptz
     OR (deleted_at = sqlc.arg(after_ts)::timestamptz AND id > sqlc.arg(after_id)::uuid)
  )
ORDER BY deleted_at ASC, id ASC
LIMIT sqlc.arg(row_limit)::int;

-- name: PurgeDeletedLog :exec
DELETE FROM deleted_log
WHERE deleted_at <= now() - INTERVAL '72 hours';

-- name: DeleteAssignmentsByID :exec
DELETE FROM assignments WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteTimeOffByID :exec
DELETE FROM time_off WHERE id = ANY(sqlc.arg(ids)::uuid[]);

-- name: DeleteAssignmentsByFloatID :exec
DELETE FROM assignments WHERE float_id = ANY(sqlc.arg(float_ids)::bigint[]);

-- name: DeleteTimeOffByFloatID :exec
DELETE FROM time_off WHERE float_id = ANY(sqlc.arg(float_ids)::bigint[]);

-- name: SetAssignmentFloatID :exec
UPDATE assignments SET float_id = $2 WHERE id = $1;

-- name: SetTimeOffFloatID :exec
UPDATE time_off SET float_id = $2 WHERE id = $1;
