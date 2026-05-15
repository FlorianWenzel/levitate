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
	ID        pgtype.UUID        `json:"id"`
	PersonID  pgtype.UUID        `json:"person_id"`
	Date      pgtype.Date        `json:"date"`
	Hours     pgtype.Numeric     `json:"hours"`
	Billable  bool               `json:"billable"`
	Notes     string             `json:"notes"`
	ProjectID pgtype.UUID        `json:"project_id"`
	FloatID   pgtype.Int8        `json:"float_id"`
	CreatedAt pgtype.Timestamptz `json:"created_at"`
	UpdatedAt pgtype.Timestamptz `json:"updated_at"`
}

const listLoggedTime = `-- name: ListLoggedTime :many
SELECT id, person_id, date, hours, billable, notes, project_id, float_id, created_at, updated_at
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
		var i LoggedTime
		if err := rows.Scan(
			&i.ID,
			&i.PersonID,
			&i.Date,
			&i.Hours,
			&i.Billable,
			&i.Notes,
			&i.ProjectID,
			&i.FloatID,
			&i.CreatedAt,
			&i.UpdatedAt,
		); err != nil {
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
SELECT id, person_id, date, hours, billable, notes, project_id, float_id, created_at, updated_at
FROM logged_time WHERE id = $1
`

func (q *Queries) GetLoggedTime(ctx context.Context, id pgtype.UUID) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, getLoggedTime, id)
	var i LoggedTime
	err := row.Scan(
		&i.ID,
		&i.PersonID,
		&i.Date,
		&i.Hours,
		&i.Billable,
		&i.Notes,
		&i.ProjectID,
		&i.FloatID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createLoggedTime = `-- name: CreateLoggedTime :one
INSERT INTO logged_time (person_id, date, hours, billable, notes, project_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, person_id, date, hours, billable, notes, project_id, float_id, created_at, updated_at
`

type CreateLoggedTimeParams struct {
	PersonID  pgtype.UUID    `json:"person_id"`
	Date      pgtype.Date    `json:"date"`
	Hours     pgtype.Numeric `json:"hours"`
	Billable  bool           `json:"billable"`
	Notes     string         `json:"notes"`
	ProjectID pgtype.UUID    `json:"project_id"`
}

func (q *Queries) CreateLoggedTime(ctx context.Context, arg CreateLoggedTimeParams) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, createLoggedTime,
		arg.PersonID,
		arg.Date,
		arg.Hours,
		arg.Billable,
		arg.Notes,
		arg.ProjectID,
	)
	var i LoggedTime
	err := row.Scan(
		&i.ID,
		&i.PersonID,
		&i.Date,
		&i.Hours,
		&i.Billable,
		&i.Notes,
		&i.ProjectID,
		&i.FloatID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const updateLoggedTime = `-- name: UpdateLoggedTime :one
UPDATE logged_time
SET date       = $2,
    hours      = $3,
    billable   = $4,
    notes      = $5,
    project_id = $6,
    updated_at = now()
WHERE id = $1
RETURNING id, person_id, date, hours, billable, notes, project_id, float_id, created_at, updated_at
`

type UpdateLoggedTimeParams struct {
	ID        pgtype.UUID    `json:"id"`
	Date      pgtype.Date    `json:"date"`
	Hours     pgtype.Numeric `json:"hours"`
	Billable  bool           `json:"billable"`
	Notes     string         `json:"notes"`
	ProjectID pgtype.UUID    `json:"project_id"`
}

func (q *Queries) UpdateLoggedTime(ctx context.Context, arg UpdateLoggedTimeParams) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, updateLoggedTime,
		arg.ID,
		arg.Date,
		arg.Hours,
		arg.Billable,
		arg.Notes,
		arg.ProjectID,
	)
	var i LoggedTime
	err := row.Scan(
		&i.ID,
		&i.PersonID,
		&i.Date,
		&i.Hours,
		&i.Billable,
		&i.Notes,
		&i.ProjectID,
		&i.FloatID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
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
SELECT id, person_id, date, hours, billable, notes, project_id, float_id, created_at, updated_at
FROM logged_time WHERE float_id = $1
`

func (q *Queries) GetLoggedTimeByFloatID(ctx context.Context, floatID pgtype.Int8) (LoggedTime, error) {
	row := q.db.QueryRow(ctx, getLoggedTimeByFloatID, floatID)
	var i LoggedTime
	err := row.Scan(
		&i.ID,
		&i.PersonID,
		&i.Date,
		&i.Hours,
		&i.Billable,
		&i.Notes,
		&i.ProjectID,
		&i.FloatID,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
