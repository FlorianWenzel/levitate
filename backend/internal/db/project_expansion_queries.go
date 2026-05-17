// Hand-written companion to the sqlc-generated queries.sql.go. These queries
// power the Float `expand=project_tasks,project_team` parameter on the
// projects API; see `ListProjectTasksByProjects` / `ListProjectTeamByProjects`
// declarations in queries.sql. The Go shapes mirror what sqlc would emit so a
// future `sqlc generate` run replaces them verbatim.

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type ListProjectTasksByProjectsRow struct {
	ID          pgtype.UUID    `json:"id"`
	ProjectID   pgtype.UUID    `json:"project_id"`
	PersonID    pgtype.UUID    `json:"person_id"`
	HoursPerDay pgtype.Numeric `json:"hours_per_day"`
	Notes       string         `json:"notes"`
}

const listProjectTasksByProjects = `-- name: ListProjectTasksByProjects :many
SELECT id, project_id, person_id, hours_per_day, notes
FROM assignments
WHERE project_id = ANY($1::uuid[])
ORDER BY project_id, id`

func (q *Queries) ListProjectTasksByProjects(ctx context.Context, projectIDs []pgtype.UUID) ([]ListProjectTasksByProjectsRow, error) {
	rows, err := q.db.Query(ctx, listProjectTasksByProjects, projectIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListProjectTasksByProjectsRow
	for rows.Next() {
		var i ListProjectTasksByProjectsRow
		if err := rows.Scan(&i.ID, &i.ProjectID, &i.PersonID, &i.HoursPerDay, &i.Notes); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type ListProjectTeamByProjectsRow struct {
	ProjectID  pgtype.UUID    `json:"project_id"`
	PersonID   pgtype.UUID    `json:"person_id"`
	HourlyRate pgtype.Numeric `json:"hourly_rate"`
}

const listProjectTeamByProjects = `-- name: ListProjectTeamByProjects :many
SELECT DISTINCT
    a.project_id,
    a.person_id,
    COALESCE(r.default_hourly_rate, 0::numeric(12,3)) AS hourly_rate
FROM assignments a
JOIN people p ON p.id = a.person_id
LEFT JOIN roles r ON lower(r.name) = lower(p.role)
WHERE a.project_id = ANY($1::uuid[])
ORDER BY a.project_id, a.person_id`

func (q *Queries) ListProjectTeamByProjects(ctx context.Context, projectIDs []pgtype.UUID) ([]ListProjectTeamByProjectsRow, error) {
	rows, err := q.db.Query(ctx, listProjectTeamByProjects, projectIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []ListProjectTeamByProjectsRow
	for rows.Next() {
		var i ListProjectTeamByProjectsRow
		if err := rows.Scan(&i.ProjectID, &i.PersonID, &i.HourlyRate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
