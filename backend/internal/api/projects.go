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
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation against the named index/constraint.
func isUniqueViolation(err error, constraint string) bool {
	var pg *pgconn.PgError
	if !errors.As(err, &pg) {
		return false
	}
	if pg.Code != "23505" {
		return false
	}
	return pg.ConstraintName == constraint
}

// Float Project budget enums. See https://developer.float.com/swagger-api-v3.yaml
//   budget_type: 1=Total hours, 2=Total fee, 3=Hourly fee
//   budget_priority: 0=Project, 1=Phase, 2=Task
const (
	projectBudgetTypeMin     = 1
	projectBudgetTypeMax     = 3
	projectBudgetPriorityMin = 0
	projectBudgetPriorityMax = 2
)

type projectDTO struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Client         string     `json:"client"`
	Color          string     `json:"color"`
	Notes          string     `json:"notes"`
	Billable       bool       `json:"billable"`
	Status         string     `json:"status"`
	BudgetType     *int       `json:"budget_type"`
	BudgetTotal    *float64   `json:"budget_total"`
	BudgetPriority *int       `json:"budget_priority"`
	Tags           []string   `json:"tags"`
	ProjectCode    *string    `json:"project_code"`
	ArchivedAt     *time.Time `json:"archived_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func toProjectDTO(p db.Project) projectDTO {
	status := "active"
	if p.ArchivedAt.Valid {
		status = "archived"
	}
	var bt *int
	if p.BudgetType.Valid {
		v := int(p.BudgetType.Int16)
		bt = &v
	}
	var bp *int
	if p.BudgetPriority.Valid {
		v := int(p.BudgetPriority.Int16)
		bp = &v
	}
	var btot *float64
	if p.BudgetTotal.Valid {
		v := numericFloat(p.BudgetTotal)
		btot = &v
	}
	tags := p.Tags
	if tags == nil {
		tags = []string{}
	}
	var pc *string
	if p.ProjectCode.Valid {
		v := p.ProjectCode.String
		pc = &v
	}
	return projectDTO{
		ID:             uuidString(p.ID),
		Name:           p.Name,
		Client:         p.Client,
		Color:          p.Color,
		Notes:          p.Notes,
		Billable:       p.Billable,
		Status:         status,
		BudgetType:     bt,
		BudgetTotal:    btot,
		BudgetPriority: bp,
		Tags:           tags,
		ProjectCode:    pc,
		ArchivedAt:     tsPtr(p.ArchivedAt),
		CreatedAt:      ts(p.CreatedAt),
		UpdatedAt:      ts(p.UpdatedAt),
	}
}

type projectInput struct {
	Name           string   `json:"name"`
	Client         string   `json:"client"`
	Color          string   `json:"color"`
	Notes          string   `json:"notes"`
	Billable       *bool    `json:"billable"`
	BudgetType     *int     `json:"budget_type"`
	BudgetTotal    *float64 `json:"budget_total"`
	BudgetPriority *int     `json:"budget_priority"`
	Tags           []string `json:"tags"`
	ProjectCode    *string  `json:"project_code"`
}

// normalizedTags trims whitespace, drops empty entries, and de-duplicates
// (case-insensitively) while preserving the first occurrence's casing. Float's
// own UI treats tags as case-insensitive labels.
func (in projectInput) normalizedTags() []string {
	out := make([]string, 0, len(in.Tags))
	seen := map[string]struct{}{}
	for _, t := range in.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func (in projectInput) validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.BudgetType != nil && (*in.BudgetType < projectBudgetTypeMin || *in.BudgetType > projectBudgetTypeMax) {
		return "budget_type must be 1 (Total hours), 2 (Total fee), or 3 (Hourly fee)"
	}
	if in.BudgetPriority != nil && (*in.BudgetPriority < projectBudgetPriorityMin || *in.BudgetPriority > projectBudgetPriorityMax) {
		return "budget_priority must be 0 (Project), 1 (Phase), or 2 (Task)"
	}
	if in.BudgetTotal != nil && *in.BudgetTotal < 0 {
		return "budget_total must be >= 0"
	}
	return ""
}

func (in projectInput) budgetParams() (pgtype.Int2, pgtype.Numeric, pgtype.Int2) {
	var bt pgtype.Int2
	if in.BudgetType != nil {
		bt = pgtype.Int2{Int16: int16(*in.BudgetType), Valid: true}
	}
	var bp pgtype.Int2
	if in.BudgetPriority != nil {
		bp = pgtype.Int2{Int16: int16(*in.BudgetPriority), Valid: true}
	}
	var btot pgtype.Numeric
	if in.BudgetTotal != nil {
		if n, err := numericFromFloat(*in.BudgetTotal); err == nil {
			btot = n
		}
	}
	return bt, btot, bp
}

// projectCodeParam normalizes the request's project_code into a pgtype.Text.
// A nil pointer or an empty/whitespace string clears the code (stored NULL)
// so the unique index treats absent values as distinct rather than colliding.
func (in projectInput) projectCodeParam() pgtype.Text {
	if in.ProjectCode == nil {
		return pgtype.Text{}
	}
	trimmed := strings.TrimSpace(*in.ProjectCode)
	if trimmed == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: trimmed, Valid: true}
}

type projectsHandler struct {
	q *db.Queries
}

func newProjectsHandler(q *db.Queries) *projectsHandler { return &projectsHandler{q: q} }

func (h *projectsHandler) routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Post("/{id}/archive", h.archive)
		r.Post("/{id}/unarchive", h.unarchive)
	})
}

func (h *projectsHandler) list(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	rows, err := h.q.ListProjects(r.Context(), includeArchived)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]projectDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, toProjectDTO(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *projectsHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.GetProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}

func (h *projectsHandler) create(w http.ResponseWriter, r *http.Request) {
	var in projectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	if in.Color == "" {
		in.Color = "#64748B"
	}
	billable := true
	if in.Billable != nil {
		billable = *in.Billable
	}
	bt, btot, bp := in.budgetParams()
	p, err := h.q.CreateProject(r.Context(), db.CreateProjectParams{
		Name:           in.Name,
		Client:         in.Client,
		Color:          in.Color,
		Notes:          in.Notes,
		Billable:       billable,
		BudgetType:     bt,
		BudgetTotal:    btot,
		BudgetPriority: bp,
		Tags:           in.normalizedTags(),
		ProjectCode:    in.projectCodeParam(),
	})
	if err != nil {
		if isUniqueViolation(err, "projects_project_code_key") {
			WriteProblem(w, r, http.StatusConflict, "project_code_conflict", "project_code must be unique")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toProjectDTO(p))
}

func (h *projectsHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var in projectInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	billable := true
	if in.Billable != nil {
		billable = *in.Billable
	}
	bt, btot, bp := in.budgetParams()
	p, err := h.q.UpdateProject(r.Context(), db.UpdateProjectParams{
		ID:             id,
		Name:           in.Name,
		Client:         in.Client,
		Color:          in.Color,
		Notes:          in.Notes,
		Billable:       billable,
		BudgetType:     bt,
		BudgetTotal:    btot,
		BudgetPriority: bp,
		Tags:           in.normalizedTags(),
		ProjectCode:    in.projectCodeParam(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found")
			return
		}
		if isUniqueViolation(err, "projects_project_code_key") {
			WriteProblem(w, r, http.StatusConflict, "project_code_conflict", "project_code must be unique")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}

func (h *projectsHandler) archive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.ArchiveProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found or already archived")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}

func (h *projectsHandler) unarchive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.UnarchiveProject(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "unarchive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toProjectDTO(p))
}
