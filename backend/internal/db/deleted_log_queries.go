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

type DeletedLog struct {
	ID         pgtype.UUID        `json:"id"`
	EntityType string             `json:"entity_type"`
	EntityID   pgtype.UUID        `json:"entity_id"`
	DeletedAt  pgtype.Timestamptz `json:"deleted_at"`
}

const insertDeletedLog = `-- name: InsertDeletedLog :one
INSERT INTO deleted_log (entity_type, entity_id)
VALUES ($1, $2)
RETURNING id, entity_type, entity_id, deleted_at
`

type InsertDeletedLogParams struct {
	EntityType string      `json:"entity_type"`
	EntityID   pgtype.UUID `json:"entity_id"`
}

func (q *Queries) InsertDeletedLog(ctx context.Context, arg InsertDeletedLogParams) (DeletedLog, error) {
	row := q.db.QueryRow(ctx, insertDeletedLog, arg.EntityType, arg.EntityID)
	var i DeletedLog
	err := row.Scan(&i.ID, &i.EntityType, &i.EntityID, &i.DeletedAt)
	return i, err
}

const listDeletedLog = `-- name: ListDeletedLog :many
SELECT id, entity_type, entity_id, deleted_at FROM deleted_log
WHERE entity_type = $1::text
  AND deleted_at > now() - INTERVAL '72 hours'
  AND (
        $2::timestamptz IS NULL
     OR deleted_at > $2::timestamptz
     OR (deleted_at = $2::timestamptz AND id > $3::uuid)
  )
ORDER BY deleted_at ASC, id ASC
LIMIT $4::int
`

type ListDeletedLogParams struct {
	EntityType string             `json:"entity_type"`
	AfterTs    pgtype.Timestamptz `json:"after_ts"`
	AfterID    pgtype.UUID        `json:"after_id"`
	RowLimit   int32              `json:"row_limit"`
}

func (q *Queries) ListDeletedLog(ctx context.Context, arg ListDeletedLogParams) ([]DeletedLog, error) {
	rows, err := q.db.Query(ctx, listDeletedLog, arg.EntityType, arg.AfterTs, arg.AfterID, arg.RowLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []DeletedLog
	for rows.Next() {
		var i DeletedLog
		if err := rows.Scan(&i.ID, &i.EntityType, &i.EntityID, &i.DeletedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const purgeDeletedLog = `-- name: PurgeDeletedLog :exec
DELETE FROM deleted_log
WHERE deleted_at <= now() - INTERVAL '72 hours'
`

func (q *Queries) PurgeDeletedLog(ctx context.Context) error {
	_, err := q.db.Exec(ctx, purgeDeletedLog)
	return err
}

const deleteAssignmentsByID = `-- name: DeleteAssignmentsByID :exec
DELETE FROM assignments WHERE id = ANY($1::uuid[])
`

func (q *Queries) DeleteAssignmentsByID(ctx context.Context, ids []pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteAssignmentsByID, ids)
	return err
}

const deleteTimeOffByID = `-- name: DeleteTimeOffByID :exec
DELETE FROM time_off WHERE id = ANY($1::uuid[])
`

func (q *Queries) DeleteTimeOffByID(ctx context.Context, ids []pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteTimeOffByID, ids)
	return err
}

const deleteAssignmentsByFloatID = `-- name: DeleteAssignmentsByFloatID :exec
DELETE FROM assignments WHERE float_id = ANY($1::bigint[])
`

func (q *Queries) DeleteAssignmentsByFloatID(ctx context.Context, floatIDs []int64) error {
	_, err := q.db.Exec(ctx, deleteAssignmentsByFloatID, floatIDs)
	return err
}

const deleteTimeOffByFloatID = `-- name: DeleteTimeOffByFloatID :exec
DELETE FROM time_off WHERE float_id = ANY($1::bigint[])
`

func (q *Queries) DeleteTimeOffByFloatID(ctx context.Context, floatIDs []int64) error {
	_, err := q.db.Exec(ctx, deleteTimeOffByFloatID, floatIDs)
	return err
}

const setAssignmentFloatID = `-- name: SetAssignmentFloatID :exec
UPDATE assignments SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetAssignmentFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setAssignmentFloatID, id, floatID)
	return err
}

const setTimeOffFloatID = `-- name: SetTimeOffFloatID :exec
UPDATE time_off SET float_id = $2 WHERE id = $1
`

func (q *Queries) SetTimeOffFloatID(ctx context.Context, id pgtype.UUID, floatID int64) error {
	_, err := q.db.Exec(ctx, setTimeOffFloatID, id, floatID)
	return err
}
