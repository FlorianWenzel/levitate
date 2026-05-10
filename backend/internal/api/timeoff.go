package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

var validTimeOffTypes = []string{"vacation", "sick", "holiday", "other"}

type timeOffDTO struct {
	ID        string    `json:"id"`
	PersonID  string    `json:"person_id"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Type      string    `json:"type"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toTimeOffDTO(t db.TimeOff) timeOffDTO {
	return timeOffDTO{
		ID:        uuidString(t.ID),
		PersonID:  uuidString(t.PersonID),
		StartDate: formatDate(t.StartDate),
		EndDate:   formatDate(t.EndDate),
		Type:      t.Type,
		Notes:     t.Notes,
		CreatedAt: ts(t.CreatedAt),
		UpdatedAt: ts(t.UpdatedAt),
	}
}

type timeOffInput struct {
	PersonID  string `json:"person_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Type      string `json:"type"`
	Notes     string `json:"notes"`
}

func (in timeOffInput) validate() string {
	if in.PersonID == "" {
		return "person_id is required"
	}
	if in.StartDate == "" || in.EndDate == "" {
		return "start_date and end_date are required"
	}
	if in.StartDate > in.EndDate {
		return "end_date must be on or after start_date"
	}
	if !slices.Contains(validTimeOffTypes, in.Type) {
		return "type must be one of: vacation, sick, holiday, other"
	}
	return ""
}

type timeOffHandler struct {
	q *db.Queries
}

func newTimeOffHandler(q *db.Queries) *timeOffHandler { return &timeOffHandler{q: q} }

func (h *timeOffHandler) routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.del)
	})
}

func (h *timeOffHandler) list(w http.ResponseWriter, r *http.Request) {
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
	rows, err := h.q.ListTimeOffInRange(r.Context(), db.ListTimeOffInRangeParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]timeOffDTO, 0, len(rows))
	for _, t := range rows {
		out = append(out, toTimeOffDTO(t))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *timeOffHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	t, err := h.q.GetTimeOff(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "time-off not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTimeOffDTO(t))
}

func (h *timeOffHandler) create(w http.ResponseWriter, r *http.Request) {
	var in timeOffInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, _ := pgUUID(in.PersonID)
	startDate, _ := parseDate(in.StartDate)
	endDate, _ := parseDate(in.EndDate)

	t, err := h.q.CreateTimeOff(r.Context(), db.CreateTimeOffParams{
		PersonID:  personID,
		StartDate: startDate,
		EndDate:   endDate,
		Type:      in.Type,
		Notes:     in.Notes,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toTimeOffDTO(t))
}

func (h *timeOffHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var in timeOffInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, _ := pgUUID(in.PersonID)
	startDate, _ := parseDate(in.StartDate)
	endDate, _ := parseDate(in.EndDate)

	t, err := h.q.UpdateTimeOff(r.Context(), db.UpdateTimeOffParams{
		ID:        id,
		PersonID:  personID,
		StartDate: startDate,
		EndDate:   endDate,
		Type:      in.Type,
		Notes:     in.Notes,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "time-off not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTimeOffDTO(t))
}

func (h *timeOffHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteTimeOff(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
