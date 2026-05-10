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

const dateLayout = "2006-01-02"

type assignmentDTO struct {
	ID          string    `json:"id"`
	PersonID    string    `json:"person_id"`
	ProjectID   string    `json:"project_id"`
	StartDate   string    `json:"start_date"` // YYYY-MM-DD
	EndDate     string    `json:"end_date"`   // YYYY-MM-DD
	HoursPerDay float64   `json:"hours_per_day"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toAssignmentDTO(a db.Assignment) assignmentDTO {
	return assignmentDTO{
		ID:          uuidString(a.ID),
		PersonID:    uuidString(a.PersonID),
		ProjectID:   uuidString(a.ProjectID),
		StartDate:   formatDate(a.StartDate),
		EndDate:     formatDate(a.EndDate),
		HoursPerDay: numericFloat(a.HoursPerDay),
		Notes:       a.Notes,
		CreatedAt:   ts(a.CreatedAt),
		UpdatedAt:   ts(a.UpdatedAt),
	}
}

func formatDate(d pgtype.Date) string {
	if !d.Valid {
		return ""
	}
	return d.Time.Format(dateLayout)
}

func parseDate(s string) (pgtype.Date, error) {
	var d pgtype.Date
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return d, err
	}
	d.Time = t
	d.Valid = true
	return d, nil
}

type assignmentInput struct {
	PersonID    string  `json:"person_id"`
	ProjectID   string  `json:"project_id"`
	StartDate   string  `json:"start_date"`
	EndDate     string  `json:"end_date"`
	HoursPerDay float64 `json:"hours_per_day"`
	Notes       string  `json:"notes"`
}

func (in assignmentInput) validate() string {
	if in.PersonID == "" || in.ProjectID == "" {
		return "person_id and project_id are required"
	}
	if in.StartDate == "" || in.EndDate == "" {
		return "start_date and end_date are required"
	}
	if in.HoursPerDay <= 0 || in.HoursPerDay > 24 {
		return "hours_per_day must be > 0 and <= 24"
	}
	if in.StartDate > in.EndDate {
		return "end_date must be on or after start_date"
	}
	return ""
}

type assignmentsHandler struct {
	q *db.Queries
}

func newAssignmentsHandler(q *db.Queries) *assignmentsHandler { return &assignmentsHandler{q: q} }

func (h *assignmentsHandler) routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.del)
	})
}

func (h *assignmentsHandler) list(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		WriteProblem(w, r, http.StatusBadRequest, "missing_range", "from and to query params are required (YYYY-MM-DD)")
		return
	}
	fromDate, err := parseDate(from)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_from", err.Error())
		return
	}
	toDate, err := parseDate(to)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_to", err.Error())
		return
	}
	rows, err := h.q.ListAssignmentsInRange(r.Context(), db.ListAssignmentsInRangeParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]assignmentDTO, 0, len(rows))
	for _, a := range rows {
		out = append(out, toAssignmentDTO(a))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *assignmentsHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	a, err := h.q.GetAssignment(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "assignment not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toAssignmentDTO(a))
}

func (h *assignmentsHandler) create(w http.ResponseWriter, r *http.Request) {
	var in assignmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, err := pgUUID(in.PersonID)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_person_id", err.Error())
		return
	}
	projectID, err := pgUUID(in.ProjectID)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_project_id", err.Error())
		return
	}
	startDate, _ := parseDate(in.StartDate)
	endDate, _ := parseDate(in.EndDate)
	hours, _ := numericFromFloat(in.HoursPerDay)

	a, err := h.q.CreateAssignment(r.Context(), db.CreateAssignmentParams{
		PersonID:    personID,
		ProjectID:   projectID,
		StartDate:   startDate,
		EndDate:     endDate,
		HoursPerDay: hours,
		Notes:       in.Notes,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toAssignmentDTO(a))
}

func (h *assignmentsHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var in assignmentInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, _ := pgUUID(in.PersonID)
	projectID, _ := pgUUID(in.ProjectID)
	startDate, _ := parseDate(in.StartDate)
	endDate, _ := parseDate(in.EndDate)
	hours, _ := numericFromFloat(in.HoursPerDay)

	a, err := h.q.UpdateAssignment(r.Context(), db.UpdateAssignmentParams{
		ID:          id,
		PersonID:    personID,
		ProjectID:   projectID,
		StartDate:   startDate,
		EndDate:     endDate,
		HoursPerDay: hours,
		Notes:       in.Notes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "assignment not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toAssignmentDTO(a))
}

func (h *assignmentsHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteAssignment(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
