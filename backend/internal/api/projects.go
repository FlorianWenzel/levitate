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
)

type projectDTO struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Client     string     `json:"client"`
	Color      string     `json:"color"`
	Notes      string     `json:"notes"`
	Billable   bool       `json:"billable"`
	Status     string     `json:"status"`
	ArchivedAt *time.Time `json:"archived_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

func toProjectDTO(p db.Project) projectDTO {
	status := "active"
	if p.ArchivedAt.Valid {
		status = "archived"
	}
	return projectDTO{
		ID:         uuidString(p.ID),
		Name:       p.Name,
		Client:     p.Client,
		Color:      p.Color,
		Notes:      p.Notes,
		Billable:   p.Billable,
		Status:     status,
		ArchivedAt: tsPtr(p.ArchivedAt),
		CreatedAt:  ts(p.CreatedAt),
		UpdatedAt:  ts(p.UpdatedAt),
	}
}

type projectInput struct {
	Name     string `json:"name"`
	Client   string `json:"client"`
	Color    string `json:"color"`
	Notes    string `json:"notes"`
	Billable *bool  `json:"billable"`
}

func (in projectInput) validate() string {
	if in.Name == "" {
		return "name is required"
	}
	return ""
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
	p, err := h.q.CreateProject(r.Context(), db.CreateProjectParams{
		Name:     in.Name,
		Client:   in.Client,
		Color:    in.Color,
		Notes:    in.Notes,
		Billable: billable,
	})
	if err != nil {
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
	p, err := h.q.UpdateProject(r.Context(), db.UpdateProjectParams{
		ID:       id,
		Name:     in.Name,
		Client:   in.Client,
		Color:    in.Color,
		Notes:    in.Notes,
		Billable: billable,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "project not found")
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
