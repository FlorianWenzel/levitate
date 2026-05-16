package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type loggedTimeDTO struct {
	ID         string     `json:"id"`
	PersonID   string     `json:"person_id"`
	Date       string     `json:"date"`
	Hours      float64    `json:"hours"`
	Billable   bool       `json:"billable"`
	Notes      string     `json:"notes"`
	ProjectID  *string    `json:"project_id"`
	Locked     bool       `json:"locked"`
	LockedDate *time.Time `json:"locked_date"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toLoggedTimeDTO(l db.LoggedTime) loggedTimeDTO {
	var projectID *string
	if l.ProjectID.Valid {
		s := uuidString(l.ProjectID)
		projectID = &s
	}
	return loggedTimeDTO{
		ID:         uuidString(l.ID),
		PersonID:   uuidString(l.PersonID),
		Date:       formatDate(l.Date),
		Hours:      numericFloat(l.Hours),
		Billable:   l.Billable,
		Notes:      l.Notes,
		ProjectID:  projectID,
		Locked:     l.Locked,
		LockedDate: tsPtr(l.LockedDate),
		CreatedAt:  ts(l.CreatedAt),
		UpdatedAt:  ts(l.UpdatedAt),
	}
}

// loggedTimeInput is the POST body for /api/logged-time. `locked` /
// `locked_date` are intentionally absent: Float treats them as read-only
// projections of project/phase/task lock settings, so we accept lock
// transitions only via the dedicated /lock and /unlock admin endpoints.
type loggedTimeInput struct {
	PersonID  string  `json:"person_id"`
	Date      string  `json:"date"`
	Hours     float64 `json:"hours"`
	Notes     string  `json:"notes"`
	ProjectID *string `json:"project_id"`
}

func (in loggedTimeInput) validate() string {
	if in.PersonID == "" {
		return "person_id is required"
	}
	if in.Date == "" {
		return "date is required"
	}
	if _, err := parseDate(in.Date); err != nil {
		return "date must be YYYY-MM-DD"
	}
	if in.Hours <= 0 || in.Hours > 24 {
		return "hours must be > 0 and <= 24"
	}
	return ""
}

// loggedTimePatch is the PATCH body for /api/logged-time/{id}; every field is
// optional so callers can update a subset (e.g. hours + notes only). billable
// is intentionally absent: it's derived from the referenced project at write
// time, matching Float's contract. locked / locked_date are likewise omitted
// — flip them via the /lock and /unlock admin endpoints, not via PATCH.
type loggedTimePatch struct {
	Date      *string  `json:"date"`
	Hours     *float64 `json:"hours"`
	Notes     *string  `json:"notes"`
	ProjectID *string  `json:"project_id"`
}

type loggedTimeHandler struct {
	q *db.Queries
}

func newLoggedTimeHandler(q *db.Queries) *loggedTimeHandler { return &loggedTimeHandler{q: q} }

func (h *loggedTimeHandler) routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.del)
		r.Post("/{id}/lock", h.lock)
		r.Post("/{id}/unlock", h.unlock)
	})
}

func (h *loggedTimeHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var params db.ListLoggedTimeParams
	if v := q.Get("person_id"); v != "" {
		id, err := pgUUID(v)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_person_id", err.Error())
			return
		}
		params.PersonID = id
	}
	if v := q.Get("project_id"); v != "" {
		id, err := pgUUID(v)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_project_id", err.Error())
			return
		}
		params.ProjectID = id
	}
	if v := q.Get("date_from"); v != "" {
		d, err := parseDate(v)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_date_from", err.Error())
			return
		}
		params.DateFrom = d
	}
	if v := q.Get("date_to"); v != "" {
		d, err := parseDate(v)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_date_to", err.Error())
			return
		}
		params.DateTo = d
	}

	rows, err := h.q.ListLoggedTime(r.Context(), params)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]loggedTimeDTO, 0, len(rows))
	for _, l := range rows {
		out = append(out, toLoggedTimeDTO(l))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *loggedTimeHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	l, err := h.q.GetLoggedTime(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "logged-time not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toLoggedTimeDTO(l))
}

func (h *loggedTimeHandler) create(w http.ResponseWriter, r *http.Request) {
	var in loggedTimeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, _ := pgUUID(in.PersonID)
	date, _ := parseDate(in.Date)
	hours, _ := numericFromFloat(in.Hours)

	projectID, billable, problem := h.resolveProjectBillable(w, r, in.ProjectID)
	if problem {
		return
	}

	l, err := h.q.CreateLoggedTime(r.Context(), db.CreateLoggedTimeParams{
		PersonID:  personID,
		Date:      date,
		Hours:     hours,
		Billable:  billable,
		Notes:     in.Notes,
		ProjectID: projectID,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toLoggedTimeDTO(l))
}

func (h *loggedTimeHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var patch loggedTimePatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	existing, err := h.q.GetLoggedTime(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "logged-time not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	// Float treats a locked entry as immutable; reject edits with 409 to give
	// callers an unambiguous signal (vs. silently dropping changes).
	if existing.Locked {
		WriteProblem(w, r, http.StatusConflict, "locked", "logged-time entry is locked")
		return
	}

	date := existing.Date
	if patch.Date != nil {
		d, err := parseDate(*patch.Date)
		if err != nil {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "date must be YYYY-MM-DD")
			return
		}
		date = d
	}
	hours := existing.Hours
	if patch.Hours != nil {
		if *patch.Hours <= 0 || *patch.Hours > 24 {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "hours must be > 0 and <= 24")
			return
		}
		h2, _ := numericFromFloat(*patch.Hours)
		hours = h2
	}
	notes := existing.Notes
	if patch.Notes != nil {
		notes = *patch.Notes
	}

	projectID := existing.ProjectID
	billable := existing.Billable
	if patch.ProjectID != nil {
		// Treat "" as clearing the project link (and falling back to non-billable).
		pid, b, problem := h.resolveProjectBillable(w, r, patch.ProjectID)
		if problem {
			return
		}
		projectID = pid
		billable = b
	}

	l, err := h.q.UpdateLoggedTime(r.Context(), db.UpdateLoggedTimeParams{
		ID:        id,
		Date:      date,
		Hours:     hours,
		Billable:  billable,
		Notes:     notes,
		ProjectID: projectID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "logged-time not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toLoggedTimeDTO(l))
}

func (h *loggedTimeHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteLoggedTime(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// lock flips the entry's locked flag to true and stamps locked_date with the
// current timestamp. Mirrors Float's locking contract where lock state is set
// by the system (project/phase/task settings) and is not modifiable through
// the regular /logged-time payload.
func (h *loggedTimeHandler) lock(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	l, err := h.q.LockLoggedTime(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "logged-time not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "lock_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toLoggedTimeDTO(l))
}

func (h *loggedTimeHandler) unlock(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	l, err := h.q.UnlockLoggedTime(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "logged-time not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "unlock_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toLoggedTimeDTO(l))
}

// resolveProjectBillable looks up the referenced project (if any) and derives
// the billable flag from it. Float's API treats billability as a read-only
// projection of the project/phase/task; we follow the same contract and
// ignore any client-supplied `billable` field. A nil or empty project_id
// leaves the row unlinked and non-billable.
//
// Writes a problem+json response and returns `problem=true` on validation
// failure so callers can simply return without further error handling.
func (h *loggedTimeHandler) resolveProjectBillable(w http.ResponseWriter, r *http.Request, raw *string) (pgtype.UUID, bool, bool) {
	var empty pgtype.UUID
	if raw == nil || *raw == "" {
		return empty, false, false
	}
	id, err := pgUUID(*raw)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_project_id", err.Error())
		return empty, false, true
	}
	p, err := h.q.GetProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "project not found")
			return empty, false, true
		}
		WriteProblem(w, r, http.StatusInternalServerError, "lookup_failed", err.Error())
		return empty, false, true
	}
	return id, p.Billable, false
}
