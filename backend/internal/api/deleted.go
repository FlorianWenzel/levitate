package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	deletedLogDefaultLimit = 50
	deletedLogMaxLimit     = 500
)

// deletedEntryDTO mirrors Float's GET /deleted/* response shape.
type deletedEntryDTO struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

type deletedListResponse struct {
	Data []deletedEntryDTO `json:"data"`
}

type deletedHandler struct {
	q *db.Queries
}

func newDeletedHandler(q *db.Queries) *deletedHandler { return &deletedHandler{q: q} }

func (h *deletedHandler) routes(r chi.Router) {
	r.Get("/assignments", h.listFor("assignment"))
	r.Get("/time-off", h.listFor("time_off"))
	r.Get("/logged-time", h.listFor("logged_time"))
}

func (h *deletedHandler) listFor(entityType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit, err := parseDeletedLimit(r.URL.Query().Get("limit"))
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_limit", err.Error())
			return
		}
		afterTs, afterID, err := decodeDeletedCursor(r.URL.Query().Get("cursor"))
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_cursor", err.Error())
			return
		}

		// Fetch one extra row to determine has_more cheaply.
		rows, err := h.q.ListDeletedLog(r.Context(), db.ListDeletedLogParams{
			EntityType: entityType,
			AfterTs:    afterTs,
			AfterID:    afterID,
			RowLimit:   int32(limit) + 1,
		})
		if err != nil {
			WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
			return
		}

		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}

		out := make([]deletedEntryDTO, 0, len(rows))
		for _, row := range rows {
			out = append(out, deletedEntryDTO{
				ID:        uuidString(row.EntityID),
				Timestamp: ts(row.DeletedAt),
			})
		}

		nextCursor := ""
		if hasMore && len(rows) > 0 {
			last := rows[len(rows)-1]
			nextCursor = encodeDeletedCursor(last.DeletedAt, last.ID)
		}

		w.Header().Set("X-Pagination-Has-More", strconv.FormatBool(hasMore))
		if nextCursor != "" {
			w.Header().Set("X-Pagination-Next-Cursor", nextCursor)
		}
		writeJSON(w, http.StatusOK, deletedListResponse{Data: out})
	}
}

func parseDeletedLimit(raw string) (int, error) {
	if raw == "" {
		return deletedLogDefaultLimit, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		n = 1
	}
	if n > deletedLogMaxLimit {
		n = deletedLogMaxLimit
	}
	return n, nil
}

// deletedCursorPayload is the encoded form of the (deleted_at, id) pair we
// page after. Using a structured JSON body inside an opaque base64 token keeps
// the wire shape stable if we ever extend the cursor.
type deletedCursorPayload struct {
	Ts string `json:"ts"`
	ID string `json:"id"`
}

func encodeDeletedCursor(t pgtype.Timestamptz, id pgtype.UUID) string {
	body := deletedCursorPayload{
		Ts: t.Time.UTC().Format(time.RFC3339Nano),
		ID: uuidString(id),
	}
	raw, _ := json.Marshal(body)
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeDeletedCursor(raw string) (pgtype.Timestamptz, pgtype.UUID, error) {
	var ts pgtype.Timestamptz
	var id pgtype.UUID
	if strings.TrimSpace(raw) == "" {
		return ts, id, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ts, id, err
	}
	var body deletedCursorPayload
	if err := json.Unmarshal(decoded, &body); err != nil {
		return ts, id, err
	}
	t, err := time.Parse(time.RFC3339Nano, body.Ts)
	if err != nil {
		return ts, id, err
	}
	ts.Time = t
	ts.Valid = true
	parsed, err := uuid.Parse(body.ID)
	if err != nil {
		return ts, id, err
	}
	id.Bytes = parsed
	id.Valid = true
	return ts, id, nil
}
