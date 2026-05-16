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

type UserStatus struct {
	ID            pgtype.UUID        `json:"id"`
	PersonID      pgtype.UUID        `json:"person_id"`
	StatusTypeID  int16              `json:"status_type_id"`
	StatusName    string             `json:"status_name"`
	StartDate     pgtype.Date        `json:"start_date"`
	EndDate       pgtype.Date        `json:"end_date"`
	RepeatState   int16              `json:"repeat_state"`
	RepeatEndDate pgtype.Date        `json:"repeat_end_date"`
	FloatID       pgtype.Int8        `json:"float_id"`
	CreatedAt     pgtype.Timestamptz `json:"created_at"`
	UpdatedAt     pgtype.Timestamptz `json:"updated_at"`
}

const userStatusSelectCols = `id, person_id, status_type_id, status_name, start_date, end_date, repeat_state, repeat_end_date, float_id, created_at, updated_at`

func scanUserStatus(scanner interface {
	Scan(dest ...any) error
}) (UserStatus, error) {
	var s UserStatus
	err := scanner.Scan(
		&s.ID,
		&s.PersonID,
		&s.StatusTypeID,
		&s.StatusName,
		&s.StartDate,
		&s.EndDate,
		&s.RepeatState,
		&s.RepeatEndDate,
		&s.FloatID,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	return s, err
}

const listUserStatuses = `-- name: ListUserStatuses :many
SELECT ` + userStatusSelectCols + `
FROM user_statuses
WHERE
    ($1::uuid IS NULL OR person_id = $1::uuid)
    AND ($2::smallint IS NULL OR status_type_id = $2::smallint)
    AND ($3::date IS NULL OR end_date >= $3::date)
    AND ($4::date IS NULL OR start_date <= $4::date)
ORDER BY start_date ASC, id ASC
`

type ListUserStatusesParams struct {
	PersonID     pgtype.UUID `json:"person_id"`
	StatusTypeID pgtype.Int2 `json:"status_type_id"`
	DateFrom     pgtype.Date `json:"date_from"`
	DateTo       pgtype.Date `json:"date_to"`
}

func (q *Queries) ListUserStatuses(ctx context.Context, arg ListUserStatusesParams) ([]UserStatus, error) {
	rows, err := q.db.Query(ctx, listUserStatuses, arg.PersonID, arg.StatusTypeID, arg.DateFrom, arg.DateTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []UserStatus
	for rows.Next() {
		s, err := scanUserStatus(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getUserStatus = `-- name: GetUserStatus :one
SELECT ` + userStatusSelectCols + `
FROM user_statuses WHERE id = $1
`

func (q *Queries) GetUserStatus(ctx context.Context, id pgtype.UUID) (UserStatus, error) {
	row := q.db.QueryRow(ctx, getUserStatus, id)
	return scanUserStatus(row)
}

const createUserStatus = `-- name: CreateUserStatus :one
INSERT INTO user_statuses (person_id, status_type_id, status_name, start_date, end_date, repeat_state, repeat_end_date)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + userStatusSelectCols

type CreateUserStatusParams struct {
	PersonID      pgtype.UUID `json:"person_id"`
	StatusTypeID  int16       `json:"status_type_id"`
	StatusName    string      `json:"status_name"`
	StartDate     pgtype.Date `json:"start_date"`
	EndDate       pgtype.Date `json:"end_date"`
	RepeatState   int16       `json:"repeat_state"`
	RepeatEndDate pgtype.Date `json:"repeat_end_date"`
}

func (q *Queries) CreateUserStatus(ctx context.Context, arg CreateUserStatusParams) (UserStatus, error) {
	row := q.db.QueryRow(ctx, createUserStatus,
		arg.PersonID,
		arg.StatusTypeID,
		arg.StatusName,
		arg.StartDate,
		arg.EndDate,
		arg.RepeatState,
		arg.RepeatEndDate,
	)
	return scanUserStatus(row)
}

const updateUserStatus = `-- name: UpdateUserStatus :one
UPDATE user_statuses
SET person_id       = $2,
    status_type_id  = $3,
    status_name     = $4,
    start_date      = $5,
    end_date        = $6,
    repeat_state    = $7,
    repeat_end_date = $8,
    updated_at      = now()
WHERE id = $1
RETURNING ` + userStatusSelectCols

type UpdateUserStatusParams struct {
	ID            pgtype.UUID `json:"id"`
	PersonID      pgtype.UUID `json:"person_id"`
	StatusTypeID  int16       `json:"status_type_id"`
	StatusName    string      `json:"status_name"`
	StartDate     pgtype.Date `json:"start_date"`
	EndDate       pgtype.Date `json:"end_date"`
	RepeatState   int16       `json:"repeat_state"`
	RepeatEndDate pgtype.Date `json:"repeat_end_date"`
}

func (q *Queries) UpdateUserStatus(ctx context.Context, arg UpdateUserStatusParams) (UserStatus, error) {
	row := q.db.QueryRow(ctx, updateUserStatus,
		arg.ID,
		arg.PersonID,
		arg.StatusTypeID,
		arg.StatusName,
		arg.StartDate,
		arg.EndDate,
		arg.RepeatState,
		arg.RepeatEndDate,
	)
	return scanUserStatus(row)
}

const deleteUserStatus = `-- name: DeleteUserStatus :exec
DELETE FROM user_statuses WHERE id = $1
`

func (q *Queries) DeleteUserStatus(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteUserStatus, id)
	return err
}

const setUserStatusFloatID = `-- name: SetUserStatusFloatID :exec
UPDATE user_statuses SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetUserStatusFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setUserStatusFloatID, id, floatID)
	return err
}

const getUserStatusByFloatID = `-- name: GetUserStatusByFloatID :one
SELECT ` + userStatusSelectCols + `
FROM user_statuses WHERE float_id = $1
`

func (q *Queries) GetUserStatusByFloatID(ctx context.Context, floatID pgtype.Int8) (UserStatus, error) {
	row := q.db.QueryRow(ctx, getUserStatusByFloatID, floatID)
	return scanUserStatus(row)
}

const deleteUserStatusesByFloatID = `-- name: DeleteUserStatusesByFloatID :exec
DELETE FROM user_statuses WHERE float_id = ANY($1::bigint[])
`

func (q *Queries) DeleteUserStatusesByFloatID(ctx context.Context, floatIDs []int64) error {
	_, err := q.db.Exec(ctx, deleteUserStatusesByFloatID, floatIDs)
	return err
}
