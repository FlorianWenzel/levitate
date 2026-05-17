// Hand-written companion to the sqlc-generated queries.sql.go.
//
// sqlc was not used to regenerate the file because the host environment for
// this change set did not include the sqlc binary; the queries here follow
// the same shape sqlc would emit so they can be replaced verbatim by a future
// `sqlc generate` run.

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type LoggedTime struct {
	ID         pgtype.UUID        `json:"id"`
	PersonID   pgtype.UUID        `json:"person_id"`
	Date       pgtype.Date        `json:"date"`
	Hours      pgtype.Numeric     `json:"hours"`
	Billable   bool               `json:"billable"`
	Notes      string             `json:"notes"`
	ProjectID  pgtype.UUID        `json:"project_id"`
	FloatID    pgtype.Int8        `json:"float_id"`
	Locked     bool               `json:"locked"`
	LockedDate pgtype.Timestamptz `json:"locked_date"`
	CreatedBy  pgtype.UUID        `json:"created_by"`
	ModifiedBy pgtype.UUID        `json:"modified_by"`
	CreatedAt  pgtype.Timestamptz `json:"created_at"`
	UpdatedAt  pgtype.Timestamptz `json:"updated_at"`
}

const loggedTimeSelectCols = `id, person_id, date, hours, billable, notes, project_id, float_id, locked, locked_date, created_by, modified_by, created_at, updated_at`

func scanLoggedTime(scanner interface {
	Scan(dest ...any) error
}) (LoggedTime, error) {
	var i LoggedTime
	err := scanner.Scan(
		&i.ID,
		&i.PersonID,
		&i.Date,
		&i.Hours,
		&i.Billable,
		&i.Notes,
		&i.ProjectID,
		&i.FloatID,
		&i.Locked,
		&i.LockedDate,
		&i.CreatedBy,
		&i.ModifiedBy,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const listLoggedTime = `-- name: ListLoggedTime :many
SELECT ` + loggedTimeSelectCols + `
FROM logged_time
WHERE
    ($1::uuid IS NULL OR person_id = $1::uuid)
    AND ($2::uuid IS NULL OR project_id = $2::uuid)
    AND ($3::date IS NULL OR date >= $3::date)
    AND ($4::date IS NULL OR date <= $4::date)
ORDER BY date ASC, id ASC
`

type ListLoggedTimeParams struct {
	PersonID  pgtype.UUID `json:"person_id"`
	ProjectID pgtype.UUID `json:"project_id"`
	DateFrom  pgtype.Date `json:"date_from"`
	DateTo    pgtype.Date `json:"date_to"`
}

func (q *Queries) ListLoggedTime(ctx context.Context, arg ListLoggedTimeParams) ([]LoggedTime, error) {
	rows, err := q.db.Query(ctx, listLoggedTime, arg.PersonID, arg.ProjectID, arg.DateFrom, arg.DateTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []LoggedTime
	for rows.Next() {
		i, err := scanLoggedTime(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getLoggedTime = `-- name: GetLoggedTime :one
SELECT ` + loggedTimeSelectCols + `
FROM logged_time WHERE id = $1
`

func (q *Queries) GetLoggedTime(ctx context.Context, id pgtype.UUID) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, getLoggedTime, id)
	return scanLoggedTime(row)
}

const createLoggedTime = `-- name: CreateLoggedTime :one
INSERT INTO logged_time (person_id, date, hours, billable, notes, project_id, created_by, modified_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
RETURNING ` + loggedTimeSelectCols + `
`

type CreateLoggedTimeParams struct {
	PersonID  pgtype.UUID    `json:"person_id"`
	Date      pgtype.Date    `json:"date"`
	Hours     pgtype.Numeric `json:"hours"`
	Billable  bool           `json:"billable"`
	Notes     string         `json:"notes"`
	ProjectID pgtype.UUID    `json:"project_id"`
	ActorID   pgtype.UUID    `json:"actor_id"`
}

func (q *Queries) CreateLoggedTime(ctx context.Context, arg CreateLoggedTimeParams) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, createLoggedTime,
		arg.PersonID,
		arg.Date,
		arg.Hours,
		arg.Billable,
		arg.Notes,
		arg.ProjectID,
		arg.ActorID,
	)
	return scanLoggedTime(row)
}

const updateLoggedTime = `-- name: UpdateLoggedTime :one
UPDATE logged_time
SET date        = $2,
    hours       = $3,
    billable    = $4,
    notes       = $5,
    project_id  = $6,
    modified_by = $7,
    updated_at  = now()
WHERE id = $1
RETURNING ` + loggedTimeSelectCols + `
`

type UpdateLoggedTimeParams struct {
	ID        pgtype.UUID    `json:"id"`
	Date      pgtype.Date    `json:"date"`
	Hours     pgtype.Numeric `json:"hours"`
	Billable  bool           `json:"billable"`
	Notes     string         `json:"notes"`
	ProjectID pgtype.UUID    `json:"project_id"`
	ActorID   pgtype.UUID    `json:"actor_id"`
}

func (q *Queries) UpdateLoggedTime(ctx context.Context, arg UpdateLoggedTimeParams) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, updateLoggedTime,
		arg.ID,
		arg.Date,
		arg.Hours,
		arg.Billable,
		arg.Notes,
		arg.ProjectID,
		arg.ActorID,
	)
	return scanLoggedTime(row)
}

const deleteLoggedTime = `-- name: DeleteLoggedTime :exec
DELETE FROM logged_time WHERE id = $1
`

func (q *Queries) DeleteLoggedTime(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteLoggedTime, id)
	return err
}

const deleteLoggedTimeByFloatID = `-- name: DeleteLoggedTimeByFloatID :exec
DELETE FROM logged_time WHERE float_id = ANY($1::bigint[])
`

func (q *Queries) DeleteLoggedTimeByFloatID(ctx context.Context, floatIDs []int64) error {
	_, err := q.db.Exec(ctx, deleteLoggedTimeByFloatID, floatIDs)
	return err
}

const setLoggedTimeFloatID = `-- name: SetLoggedTimeFloatID :exec
UPDATE logged_time SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetLoggedTimeFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setLoggedTimeFloatID, id, floatID)
	return err
}

const getLoggedTimeByFloatID = `-- name: GetLoggedTimeByFloatID :one
SELECT ` + loggedTimeSelectCols + `
FROM logged_time WHERE float_id = $1
`

func (q *Queries) GetLoggedTimeByFloatID(ctx context.Context, floatID pgtype.Int8) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, getLoggedTimeByFloatID, floatID)
	return scanLoggedTime(row)
}

// LockLoggedTime flips an entry to locked=true and stamps locked_date with the
// current timestamp (idempotent: re-locking does not refresh locked_date once
// set, mirroring Float's "set automatically when locked transitions to true"
// contract).
const lockLoggedTime = `-- name: LockLoggedTime :one
UPDATE logged_time
SET locked      = true,
    locked_date = COALESCE(locked_date, now()),
    modified_by = $2,
    updated_at  = now()
WHERE id = $1
RETURNING ` + loggedTimeSelectCols + `
`

func (q *Queries) LockLoggedTime(ctx context.Context, id pgtype.UUID, actorID pgtype.UUID) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, lockLoggedTime, id, actorID)
	return scanLoggedTime(row)
}

// UnlockLoggedTime flips an entry back to locked=false and clears locked_date.
const unlockLoggedTime = `-- name: UnlockLoggedTime :one
UPDATE logged_time
SET locked      = false,
    locked_date = NULL,
    modified_by = $2,
    updated_at  = now()
WHERE id = $1
RETURNING ` + loggedTimeSelectCols + `
`

func (q *Queries) UnlockLoggedTime(ctx context.Context, id pgtype.UUID, actorID pgtype.UUID) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, unlockLoggedTime, id, actorID)
	return scanLoggedTime(row)
}
