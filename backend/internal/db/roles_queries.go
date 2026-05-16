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

type Role struct {
	ID                pgtype.UUID        `json:"id"`
	Name              string             `json:"name"`
	DefaultHourlyRate pgtype.Numeric     `json:"default_hourly_rate"`
	CostRateHistory   []byte             `json:"cost_rate_history"`
	FloatID           pgtype.Int8        `json:"float_id"`
	CreatedAt         pgtype.Timestamptz `json:"created_at"`
	UpdatedAt         pgtype.Timestamptz `json:"updated_at"`
}

const roleSelectCols = `id, name, default_hourly_rate, cost_rate_history, float_id, created_at, updated_at`

func scanRole(scanner interface {
	Scan(dest ...any) error
}) (Role, error) {
	var r Role
	err := scanner.Scan(
		&r.ID,
		&r.Name,
		&r.DefaultHourlyRate,
		&r.CostRateHistory,
		&r.FloatID,
		&r.CreatedAt,
		&r.UpdatedAt,
	)
	return r, err
}

const listRoles = `-- name: ListRoles :many
SELECT ` + roleSelectCols + `
FROM roles
ORDER BY name ASC, id ASC
`

func (q *Queries) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := q.db.Query(ctx, listRoles)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Role
	for rows.Next() {
		r, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getRole = `-- name: GetRole :one
SELECT ` + roleSelectCols + `
FROM roles WHERE id = $1
`

func (q *Queries) GetRole(ctx context.Context, id pgtype.UUID) (Role, error) {
	row := q.db.QueryRow(ctx, getRole, id)
	return scanRole(row)
}

const getRoleByName = `-- name: GetRoleByName :one
SELECT ` + roleSelectCols + `
FROM roles WHERE lower(name) = lower($1)
`

func (q *Queries) GetRoleByName(ctx context.Context, name string) (Role, error) {
	row := q.db.QueryRow(ctx, getRoleByName, name)
	return scanRole(row)
}

const createRole = `-- name: CreateRole :one
INSERT INTO roles (name, default_hourly_rate, cost_rate_history)
VALUES ($1, $2, $3)
RETURNING ` + roleSelectCols

type CreateRoleParams struct {
	Name              string         `json:"name"`
	DefaultHourlyRate pgtype.Numeric `json:"default_hourly_rate"`
	CostRateHistory   []byte         `json:"cost_rate_history"`
}

func (q *Queries) CreateRole(ctx context.Context, arg CreateRoleParams) (Role, error) {
	row := q.db.QueryRow(ctx, createRole, arg.Name, arg.DefaultHourlyRate, arg.CostRateHistory)
	return scanRole(row)
}

const updateRole = `-- name: UpdateRole :one
UPDATE roles
SET name                = $2,
    default_hourly_rate = $3,
    cost_rate_history   = $4,
    updated_at          = now()
WHERE id = $1
RETURNING ` + roleSelectCols

type UpdateRoleParams struct {
	ID                pgtype.UUID    `json:"id"`
	Name              string         `json:"name"`
	DefaultHourlyRate pgtype.Numeric `json:"default_hourly_rate"`
	CostRateHistory   []byte         `json:"cost_rate_history"`
}

func (q *Queries) UpdateRole(ctx context.Context, arg UpdateRoleParams) (Role, error) {
	row := q.db.QueryRow(ctx, updateRole, arg.ID, arg.Name, arg.DefaultHourlyRate, arg.CostRateHistory)
	return scanRole(row)
}

const deleteRole = `-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1
`

func (q *Queries) DeleteRole(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteRole, id)
	return err
}

const setRoleFloatID = `-- name: SetRoleFloatID :exec
UPDATE roles SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetRoleFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setRoleFloatID, id, floatID)
	return err
}

const getRoleByFloatID = `-- name: GetRoleByFloatID :one
SELECT ` + roleSelectCols + `
FROM roles WHERE float_id = $1
`

func (q *Queries) GetRoleByFloatID(ctx context.Context, floatID pgtype.Int8) (Role, error) {
	row := q.db.QueryRow(ctx, getRoleByFloatID, floatID)
	return scanRole(row)
}
