package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type milestoneDTO struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	PhaseID   *string   `json:"phase_id"`
	Name      string    `json:"name"`
	Date      string    `json:"date"`
	EndDate   *string   `json:"end_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toMilestoneDTO(m db.Milestone) milestoneDTO {
	var phaseID *string
	if m.PhaseID.Valid {
		s := uuidString(m.PhaseID)
		phaseID = &s
	}
	var endDate *string
	if m.EndDate.Valid {
		s := formatDate(m.EndDate)
		endDate = &s
	}
	return milestoneDTO{
		ID:        uuidString(m.ID),
		ProjectID: uuidString(m.ProjectID),
		PhaseID:   phaseID,
		Name:      m.Name,
		Date:      formatDate(m.Date),
		EndDate:   endDate,
		CreatedAt: ts(m.CreatedAt),
		UpdatedAt: ts(m.UpdatedAt),
	}
}

type milestoneInput struct {
	Name    string  `json:"name"`
	Date    string  `json:"date"`
	EndDate *string `json:"end_date"`
	PhaseID *string `json:"phase_id"`
}

func (in milestoneInput) validate() string {
	if strings.TrimSpace(in.Name) == "" {
		return "name is required"
	}
	if in.Date == "" {
		return "date is required"
	}
	if _, err := parseDate(in.Date); err != nil {
		return "date must be YYYY-MM-DD"
	}
	if in.EndDate != nil && *in.EndDate != "" {
		end, err := parseDate(*in.EndDate)
		if err != nil {
			return "end_date must be YYYY-MM-DD"
		}
		start, _ := parseDate(in.Date)
		if start.Time.After(end.Time) {
			return "end_date must be on or after date"
		}
	}
	return ""
}

func (in milestoneInput) phasePG() (pgtype.UUID, string) {
	var u pgtype.UUID
	if in.PhaseID == nil || *in.PhaseID == "" {
		return u, ""
	}
	parsed, err := pgUUID(*in.PhaseID)
	if err != nil {
		return u, "phase_id must be a valid UUID"
	}
	return parsed, ""
}

func (in milestoneInput) endDatePG() pgtype.Date {
	var d pgtype.Date
	if in.EndDate == nil || *in.EndDate == "" {
		return d
	}
	parsed, err := parseDate(*in.EndDate)
	if err != nil {
		return d
	}
	return parsed
}

type milestonesHandler struct {
	q *db.Queries
}

func newMilestonesHandler(q *db.Queries) *milestonesHandler { return &milestonesHandler{q: q} }

// projectRoutes wires the routes under /api/projects/{id}/milestones.
func (h *milestonesHandler) projectRoutes(r chi.Router) {
	r.Get("/", h.listForProject)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.createForProject)
	})
}

// itemRoutes wires the routes under /api/milestones/{id}.
func (h *milestonesHandler) itemRoutes(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Patch("/{id}", h.update)
		r.Delete("/{id}", h.del)
	})
}

func (h *milestonesHandler) listForProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	rows, err := h.q.ListMilestonesByProject(r.Context(), projectID)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]milestoneDTO, 0, len(rows))
	for _, m := range rows {
		out = append(out, toMilestoneDTO(m))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *milestonesHandler) createForProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if _, err := h.q.GetProject(r.Context(), projectID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	var in milestoneInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	phaseID, phaseErr := in.phasePG()
	if phaseErr != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", phaseErr)
		return
	}
	date, _ := parseDate(in.Date)
	m, err := h.q.CreateMilestone(r.Context(), db.CreateMilestoneParams{
		ProjectID: projectID,
		PhaseID:   phaseID,
		Name:      strings.TrimSpace(in.Name),
		Date:      date,
		EndDate:   in.endDatePG(),
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toMilestoneDTO(m))
}

func (h *milestonesHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var in milestoneInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	phaseID, phaseErr := in.phasePG()
	if phaseErr != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", phaseErr)
		return
	}
	date, _ := parseDate(in.Date)
	m, err := h.q.UpdateMilestone(r.Context(), db.UpdateMilestoneParams{
		ID:      id,
		PhaseID: phaseID,
		Name:    strings.TrimSpace(in.Name),
		Date:    date,
		EndDate: in.endDatePG(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "milestone not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toMilestoneDTO(m))
}

func (h *milestonesHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteMilestone(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
