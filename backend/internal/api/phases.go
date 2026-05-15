package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	phasesDefaultLimit = 50
	phasesMaxLimit     = 200
)

type phaseDTO struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	Name              string     `json:"name"`
	Color             string     `json:"color"`
	Notes             string     `json:"notes"`
	StartDate         *string    `json:"start_date"`
	EndDate           *string    `json:"end_date"`
	BudgetTotal       float64    `json:"budget_total"`
	DefaultHourlyRate float64    `json:"default_hourly_rate"`
	NonBillable       bool       `json:"non_billable"`
	Status            int        `json:"status"`
	Active            int        `json:"active"`
	ArchivedAt        *time.Time `json:"archived_at"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func toPhaseDTO(p db.Phase) phaseDTO {
	var start, end *string
	if p.StartDate.Valid {
		s := formatDate(p.StartDate)
		start = &s
	}
	if p.EndDate.Valid {
		s := formatDate(p.EndDate)
		end = &s
	}
	active := 1
	if p.ArchivedAt.Valid {
		active = 0
	}
	return phaseDTO{
		ID:                uuidString(p.ID),
		ProjectID:         uuidString(p.ProjectID),
		Name:              p.Name,
		Color:             p.Color,
		Notes:             p.Notes,
		StartDate:         start,
		EndDate:           end,
		BudgetTotal:       numericFloat(p.BudgetTotal),
		DefaultHourlyRate: numericFloat(p.DefaultHourlyRate),
		NonBillable:       !p.Billable,
		Status:            int(p.Status),
		Active:            active,
		ArchivedAt:        tsPtr(p.ArchivedAt),
		CreatedAt:         ts(p.CreatedAt),
		UpdatedAt:         ts(p.UpdatedAt),
	}
}

// phaseInput matches the Float Phases API write contract closely while staying
// idiomatic for levitate consumers. Optional fields default sensibly when
// omitted (status=2 "Confirmed", non_billable=false, budgets=0).
type phaseInput struct {
	Name              string   `json:"name"`
	Color             string   `json:"color"`
	Notes             string   `json:"notes"`
	StartDate         *string  `json:"start_date"`
	EndDate           *string  `json:"end_date"`
	BudgetTotal       *float64 `json:"budget_total"`
	DefaultHourlyRate *float64 `json:"default_hourly_rate"`
	NonBillable       *bool    `json:"non_billable"`
	Status            *int     `json:"status"`
}

func (in phaseInput) validate() string {
	if strings.TrimSpace(in.Name) == "" {
		return "name is required"
	}
	if in.StartDate != nil && *in.StartDate != "" {
		if _, err := parseDate(*in.StartDate); err != nil {
			return "start_date must be YYYY-MM-DD"
		}
	}
	if in.EndDate != nil && *in.EndDate != "" {
		if _, err := parseDate(*in.EndDate); err != nil {
			return "end_date must be YYYY-MM-DD"
		}
	}
	if in.StartDate != nil && *in.StartDate != "" && in.EndDate != nil && *in.EndDate != "" {
		start, _ := parseDate(*in.StartDate)
		end, _ := parseDate(*in.EndDate)
		if start.Time.After(end.Time) {
			return "end_date must be on or after start_date"
		}
	}
	if in.Status != nil && (*in.Status < 0 || *in.Status > 2) {
		return "status must be 0, 1, or 2"
	}
	if in.BudgetTotal != nil && *in.BudgetTotal < 0 {
		return "budget_total must be >= 0"
	}
	if in.DefaultHourlyRate != nil && *in.DefaultHourlyRate < 0 {
		return "default_hourly_rate must be >= 0"
	}
	return ""
}

func (in phaseInput) datePG(s *string) pgtype.Date {
	var d pgtype.Date
	if s == nil || *s == "" {
		return d
	}
	parsed, err := parseDate(*s)
	if err != nil {
		return d
	}
	return parsed
}

type phasesHandler struct {
	q *db.Queries
}

func newPhasesHandler(q *db.Queries) *phasesHandler { return &phasesHandler{q: q} }

// projectRoutes wires the routes under /api/projects/{id}/phases.
func (h *phasesHandler) projectRoutes(r chi.Router) {
	r.Get("/", h.listForProject)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.createForProject)
	})
}

// itemRoutes wires the routes under /api/phases.
func (h *phasesHandler) itemRoutes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Patch("/{id}", h.update)
		r.Put("/{id}", h.update)
		r.Post("/{id}/archive", h.archive)
		r.Post("/{id}/unarchive", h.unarchive)
		r.Delete("/{id}", h.del)
	})
}

func writePhaseList(w http.ResponseWriter, rows []db.Phase, limit, offset int) {
	total := len(rows)
	end := offset + limit
	if offset > total {
		offset = total
	}
	if end > total {
		end = total
	}
	page := rows[offset:end]
	out := make([]phaseDTO, 0, len(page))
	for _, p := range page {
		out = append(out, toPhaseDTO(p))
	}
	hasMore := end < total
	totalPages := 1
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
		if totalPages == 0 {
			totalPages = 1
		}
	}
	currentPage := 1
	if limit > 0 {
		currentPage = offset/limit + 1
	}
	w.Header().Set("X-Pagination-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Pagination-Page-Count", strconv.Itoa(totalPages))
	w.Header().Set("X-Pagination-Current-Page", strconv.Itoa(currentPage))
	w.Header().Set("X-Pagination-Per-Page", strconv.Itoa(limit))
	w.Header().Set("X-Pagination-Has-More", strconv.FormatBool(hasMore))
	writeJSON(w, http.StatusOK, out)
}

func parsePhasesPagination(r *http.Request) (limit, offset int, err error) {
	limit = phasesDefaultLimit
	if raw := r.URL.Query().Get("per-page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
		if n < 1 {
			n = 1
		}
		if n > phasesMaxLimit {
			n = phasesMaxLimit
		}
		limit = n
	}
	page := 1
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, err
		}
		if n < 1 {
			n = 1
		}
		page = n
	}
	offset = (page - 1) * limit
	return limit, offset, nil
}

func (h *phasesHandler) listForProject(w http.ResponseWriter, r *http.Request) {
	projectID, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	limit, offset, err := parsePhasesPagination(r)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_pagination", err.Error())
		return
	}
	rows, err := h.q.ListPhasesByProject(r.Context(), projectID)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	writePhaseList(w, rows, limit, offset)
}

func (h *phasesHandler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePhasesPagination(r)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_pagination", err.Error())
		return
	}
	if pid := r.URL.Query().Get("project_id"); pid != "" {
		projectID, err := pgUUID(pid)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_project_id", err.Error())
			return
		}
		rows, err := h.q.ListPhasesByProject(r.Context(), projectID)
		if err != nil {
			WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
			return
		}
		writePhaseList(w, rows, limit, offset)
		return
	}
	rows, err := h.q.ListPhases(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	writePhaseList(w, rows, limit, offset)
}

func (h *phasesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.GetPhase(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "phase not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPhaseDTO(p))
}

func (h *phasesHandler) createForProject(w http.ResponseWriter, r *http.Request) {
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
	var in phaseInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	params := db.CreatePhaseParams{
		ProjectID: projectID,
		Name:      strings.TrimSpace(in.Name),
		Color:     strings.TrimSpace(in.Color),
		Notes:     in.Notes,
		StartDate: in.datePG(in.StartDate),
		EndDate:   in.datePG(in.EndDate),
		Billable:  true,
		Status:    2,
	}
	if in.NonBillable != nil {
		params.Billable = !*in.NonBillable
	}
	if in.Status != nil {
		params.Status = int16(*in.Status)
	}
	budget := 0.0
	if in.BudgetTotal != nil {
		budget = *in.BudgetTotal
	}
	if n, err := numericFromFloat(budget); err == nil {
		params.BudgetTotal = n
	}
	rate := 0.0
	if in.DefaultHourlyRate != nil {
		rate = *in.DefaultHourlyRate
	}
	if n, err := numericFromFloat(rate); err == nil {
		params.DefaultHourlyRate = n
	}
	p, err := h.q.CreatePhase(r.Context(), params)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toPhaseDTO(p))
}

func (h *phasesHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	existing, err := h.q.GetPhase(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "phase not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	var in phaseInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if strings.TrimSpace(in.Name) == "" {
		in.Name = existing.Name
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	params := db.UpdatePhaseParams{
		ID:                id,
		Name:              strings.TrimSpace(in.Name),
		Color:             strings.TrimSpace(in.Color),
		Notes:             in.Notes,
		StartDate:         in.datePG(in.StartDate),
		EndDate:           in.datePG(in.EndDate),
		BudgetTotal:       existing.BudgetTotal,
		DefaultHourlyRate: existing.DefaultHourlyRate,
		Billable:          existing.Billable,
		Status:            existing.Status,
	}
	if in.BudgetTotal != nil {
		if n, err := numericFromFloat(*in.BudgetTotal); err == nil {
			params.BudgetTotal = n
		}
	}
	if in.DefaultHourlyRate != nil {
		if n, err := numericFromFloat(*in.DefaultHourlyRate); err == nil {
			params.DefaultHourlyRate = n
		}
	}
	if in.NonBillable != nil {
		params.Billable = !*in.NonBillable
	}
	if in.Status != nil {
		params.Status = int16(*in.Status)
	}
	p, err := h.q.UpdatePhase(r.Context(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "phase not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPhaseDTO(p))
}

func (h *phasesHandler) archive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.ArchivePhase(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "phase not found or already archived")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPhaseDTO(p))
}

func (h *phasesHandler) unarchive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.UnarchivePhase(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "phase not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "unarchive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPhaseDTO(p))
}

func (h *phasesHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeletePhase(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
