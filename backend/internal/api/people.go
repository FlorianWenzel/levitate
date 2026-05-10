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

type personDTO struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Email               string     `json:"email"`
	Role                string     `json:"role"`
	WeeklyCapacityHours float64    `json:"weekly_capacity_hours"`
	ArchivedAt          *time.Time `json:"archived_at"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func toPersonDTO(p db.Person) personDTO {
	return personDTO{
		ID:                  uuidString(p.ID),
		Name:                p.Name,
		Email:               p.Email,
		Role:                p.Role,
		WeeklyCapacityHours: numericFloat(p.WeeklyCapacityHours),
		ArchivedAt:          tsPtr(p.ArchivedAt),
		CreatedAt:           ts(p.CreatedAt),
		UpdatedAt:           ts(p.UpdatedAt),
	}
}

type personInput struct {
	Name                string  `json:"name"`
	Email               string  `json:"email"`
	Role                string  `json:"role"`
	WeeklyCapacityHours float64 `json:"weekly_capacity_hours"`
}

func (in personInput) validate() string {
	if in.Name == "" {
		return "name is required"
	}
	if in.WeeklyCapacityHours < 0 || in.WeeklyCapacityHours > 168 {
		return "weekly_capacity_hours must be between 0 and 168"
	}
	return ""
}

type peopleHandler struct {
	q *db.Queries
}

func newPeopleHandler(q *db.Queries) *peopleHandler { return &peopleHandler{q: q} }

func (h *peopleHandler) routes(r chi.Router) {
	// Read access for all authenticated users.
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	// Mutations require admin.
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Post("/{id}/archive", h.archive)
		r.Post("/{id}/unarchive", h.unarchive)
	})
}

func (h *peopleHandler) list(w http.ResponseWriter, r *http.Request) {
	includeArchived := r.URL.Query().Get("include_archived") == "true"
	rows, err := h.q.ListPeople(r.Context(), includeArchived)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]personDTO, 0, len(rows))
	for _, p := range rows {
		out = append(out, toPersonDTO(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *peopleHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.GetPerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "person not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPersonDTO(p))
}

func (h *peopleHandler) create(w http.ResponseWriter, r *http.Request) {
	var in personInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	cap, _ := numericFromFloat(in.WeeklyCapacityHours)
	p, err := h.q.CreatePerson(r.Context(), db.CreatePersonParams{
		Name:                in.Name,
		Email:               in.Email,
		Role:                in.Role,
		WeeklyCapacityHours: cap,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toPersonDTO(p))
}

func (h *peopleHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	var in personInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	cap, _ := numericFromFloat(in.WeeklyCapacityHours)
	p, err := h.q.UpdatePerson(r.Context(), db.UpdatePersonParams{
		ID:                  id,
		Name:                in.Name,
		Email:               in.Email,
		Role:                in.Role,
		WeeklyCapacityHours: cap,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "person not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPersonDTO(p))
}

func (h *peopleHandler) archive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.ArchivePerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "person not found or already archived")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "archive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPersonDTO(p))
}

func (h *peopleHandler) unarchive(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	p, err := h.q.UnarchivePerson(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "person not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "unarchive_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toPersonDTO(p))
}
