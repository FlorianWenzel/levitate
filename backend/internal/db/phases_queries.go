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

const phaseSelectCols = `id, project_id, name, color, notes, start_date, end_date, budget_total, default_hourly_rate, billable, status, archived_at, float_id, created_at, updated_at`

func scanPhase(scanner interface {
	Scan(dest ...any) error
}) (Phase, error) {
	var p Phase
	err := scanner.Scan(
		&p.ID,
		&p.ProjectID,
		&p.Name,
		&p.Color,
		&p.Notes,
		&p.StartDate,
		&p.EndDate,
		&p.BudgetTotal,
		&p.DefaultHourlyRate,
		&p.Billable,
		&p.Status,
		&p.ArchivedAt,
		&p.FloatID,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	return p, err
}

const listPhasesByProject = `-- name: ListPhasesByProject :many
SELECT ` + phaseSelectCols + `
FROM phases
WHERE project_id = $1
ORDER BY
    CASE WHEN start_date IS NULL THEN 1 ELSE 0 END,
    start_date ASC,
    name ASC,
    id ASC
`

func (q *Queries) ListPhasesByProject(ctx context.Context, projectID pgtype.UUID) ([]Phase, error) {
	rows, err := q.db.Query(ctx, listPhasesByProject, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Phase
	for rows.Next() {
		p, err := scanPhase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const listPhases = `-- name: ListPhases :many
SELECT ` + phaseSelectCols + `
FROM phases
ORDER BY
    CASE WHEN start_date IS NULL THEN 1 ELSE 0 END,
    start_date ASC,
    name ASC,
    id ASC
`

func (q *Queries) ListPhases(ctx context.Context) ([]Phase, error) {
	rows, err := q.db.Query(ctx, listPhases)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Phase
	for rows.Next() {
		p, err := scanPhase(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPhase = `-- name: GetPhase :one
SELECT ` + phaseSelectCols + `
FROM phases WHERE id = $1
`

func (q *Queries) GetPhase(ctx context.Context, id pgtype.UUID) (Phase, error) {
	row := q.db.QueryRow(ctx, getPhase, id)
	return scanPhase(row)
}

const createPhase = `-- name: CreatePhase :one
INSERT INTO phases (project_id, name, color, notes, start_date, end_date, budget_total, default_hourly_rate, billable, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING ` + phaseSelectCols

type CreatePhaseParams struct {
	ProjectID         pgtype.UUID    `json:"project_id"`
	Name              string         `json:"name"`
	Color             string         `json:"color"`
	Notes             string         `json:"notes"`
	StartDate         pgtype.Date    `json:"start_date"`
	EndDate           pgtype.Date    `json:"end_date"`
	BudgetTotal       pgtype.Numeric `json:"budget_total"`
	DefaultHourlyRate pgtype.Numeric `json:"default_hourly_rate"`
	Billable          bool           `json:"billable"`
	Status            int16          `json:"status"`
}

func (q *Queries) CreatePhase(ctx context.Context, arg CreatePhaseParams) (Phase, error) {
	row := q.db.QueryRow(ctx, createPhase,
		arg.ProjectID,
		arg.Name,
		arg.Color,
		arg.Notes,
		arg.StartDate,
		arg.EndDate,
		arg.BudgetTotal,
		arg.DefaultHourlyRate,
		arg.Billable,
		arg.Status,
	)
	return scanPhase(row)
}

const updatePhase = `-- name: UpdatePhase :one
UPDATE phases
SET name                = $2,
    color               = $3,
    notes               = $4,
    start_date          = $5,
    end_date            = $6,
    budget_total        = $7,
    default_hourly_rate = $8,
    billable            = $9,
    status              = $10,
    updated_at          = now()
WHERE id = $1
RETURNING ` + phaseSelectCols

type UpdatePhaseParams struct {
	ID                pgtype.UUID    `json:"id"`
	Name              string         `json:"name"`
	Color             string         `json:"color"`
	Notes             string         `json:"notes"`
	StartDate         pgtype.Date    `json:"start_date"`
	EndDate           pgtype.Date    `json:"end_date"`
	BudgetTotal       pgtype.Numeric `json:"budget_total"`
	DefaultHourlyRate pgtype.Numeric `json:"default_hourly_rate"`
	Billable          bool           `json:"billable"`
	Status            int16          `json:"status"`
}

func (q *Queries) UpdatePhase(ctx context.Context, arg UpdatePhaseParams) (Phase, error) {
	row := q.db.QueryRow(ctx, updatePhase,
		arg.ID,
		arg.Name,
		arg.Color,
		arg.Notes,
		arg.StartDate,
		arg.EndDate,
		arg.BudgetTotal,
		arg.DefaultHourlyRate,
		arg.Billable,
		arg.Status,
	)
	return scanPhase(row)
}

const archivePhase = `-- name: ArchivePhase :one
UPDATE phases
SET archived_at = now(), updated_at = now()
WHERE id = $1 AND archived_at IS NULL
RETURNING ` + phaseSelectCols

func (q *Queries) ArchivePhase(ctx context.Context, id pgtype.UUID) (Phase, error) {
	row := q.db.QueryRow(ctx, archivePhase, id)
	return scanPhase(row)
}

const unarchivePhase = `-- name: UnarchivePhase :one
UPDATE phases
SET archived_at = NULL, updated_at = now()
WHERE id = $1
RETURNING ` + phaseSelectCols

func (q *Queries) UnarchivePhase(ctx context.Context, id pgtype.UUID) (Phase, error) {
	row := q.db.QueryRow(ctx, unarchivePhase, id)
	return scanPhase(row)
}

const deletePhase = `-- name: DeletePhase :exec
DELETE FROM phases WHERE id = $1
`

func (q *Queries) DeletePhase(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deletePhase, id)
	return err
}

const setPhaseFloatID = `-- name: SetPhaseFloatID :exec
UPDATE phases SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetPhaseFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setPhaseFloatID, id, floatID)
	return err
}

const getPhaseByFloatID = `-- name: GetPhaseByFloatID :one
SELECT ` + phaseSelectCols + `
FROM phases WHERE float_id = $1
`

func (q *Queries) GetPhaseByFloatID(ctx context.Context, floatID pgtype.Int8) (Phase, error) {
	row := q.db.QueryRow(ctx, getPhaseByFloatID, floatID)
	return scanPhase(row)
}
