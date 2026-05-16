package api

import (
	"encoding/json"
	"errors"
	"fmt"
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

// roleCostRateEntry mirrors Float's `cost_rate_history` array element shape:
// `{ rate: string, effective_date: string }`. The rate is a string-formatted
// decimal (e.g. "180.000") so a Float integration can pass it through.
type roleCostRateEntry struct {
	Rate          string `json:"rate"`
	EffectiveDate string `json:"effective_date"`
}

// roleDTO mirrors Float's Roles response. Float emits `default_hourly_rate`
// as a string (e.g. "260.000"); we follow suit so the field is a direct
// passthrough on a Float import/export round-trip.
type roleDTO struct {
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	DefaultHourlyRate string              `json:"default_hourly_rate"`
	CostRateHistory   []roleCostRateEntry `json:"cost_rate_history"`
	PeopleIDs         []string            `json:"people_ids"`
	PeopleCount       int                 `json:"people_count"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

func formatRateString(n pgtype.Numeric) string {
	return strconv.FormatFloat(numericFloat(n), 'f', 3, 64)
}

func decodeCostRateHistory(raw []byte) []roleCostRateEntry {
	if len(raw) == 0 {
		return []roleCostRateEntry{}
	}
	var out []roleCostRateEntry
	if err := json.Unmarshal(raw, &out); err != nil {
		return []roleCostRateEntry{}
	}
	if out == nil {
		return []roleCostRateEntry{}
	}
	return out
}

func toRoleDTO(r db.Role, peopleIDs []string) roleDTO {
	if peopleIDs == nil {
		peopleIDs = []string{}
	}
	return roleDTO{
		ID:                uuidString(r.ID),
		Name:              r.Name,
		DefaultHourlyRate: formatRateString(r.DefaultHourlyRate),
		CostRateHistory:   decodeCostRateHistory(r.CostRateHistory),
		PeopleIDs:         peopleIDs,
		PeopleCount:       len(peopleIDs),
		CreatedAt:         ts(r.CreatedAt),
		UpdatedAt:         ts(r.UpdatedAt),
	}
}

// roleInput accepts both a numeric or string `default_hourly_rate` so callers
// can post Float's native string form (e.g. "260.000") or a JSON number.
type roleInput struct {
	Name              string              `json:"name"`
	DefaultHourlyRate *json.RawMessage    `json:"default_hourly_rate"`
	CostRateHistory   []roleCostRateEntry `json:"cost_rate_history"`
}

func parseRateInput(raw *json.RawMessage) (pgtype.Numeric, error) {
	if raw == nil {
		return numericFromFloat(0)
	}
	trimmed := strings.TrimSpace(string(*raw))
	if trimmed == "" || trimmed == "null" {
		return numericFromFloat(0)
	}
	if strings.HasPrefix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(*raw, &s); err != nil {
			return pgtype.Numeric{}, fmt.Errorf("default_hourly_rate must be a string or number")
		}
		if s == "" {
			return numericFromFloat(0)
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return pgtype.Numeric{}, fmt.Errorf("default_hourly_rate must be a numeric string")
		}
		if f < 0 {
			return pgtype.Numeric{}, fmt.Errorf("default_hourly_rate must be >= 0")
		}
		return numericFromFloat(f)
	}
	f, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return pgtype.Numeric{}, fmt.Errorf("default_hourly_rate must be a string or number")
	}
	if f < 0 {
		return pgtype.Numeric{}, fmt.Errorf("default_hourly_rate must be >= 0")
	}
	return numericFromFloat(f)
}

func validateCostRateHistory(entries []roleCostRateEntry) string {
	for i, e := range entries {
		if strings.TrimSpace(e.Rate) == "" {
			return fmt.Sprintf("cost_rate_history[%d].rate is required", i)
		}
		if _, err := strconv.ParseFloat(e.Rate, 64); err != nil {
			return fmt.Sprintf("cost_rate_history[%d].rate must be a numeric string", i)
		}
		if strings.TrimSpace(e.EffectiveDate) == "" {
			return fmt.Sprintf("cost_rate_history[%d].effective_date is required", i)
		}
		if _, err := parseDate(e.EffectiveDate); err != nil {
			return fmt.Sprintf("cost_rate_history[%d].effective_date must be YYYY-MM-DD", i)
		}
	}
	return ""
}

func encodeCostRateHistory(entries []roleCostRateEntry) ([]byte, error) {
	if entries == nil {
		entries = []roleCostRateEntry{}
	}
	return json.Marshal(entries)
}

type rolesHandler struct {
	q *db.Queries
}

func newRolesHandler(q *db.Queries) *rolesHandler { return &rolesHandler{q: q} }

func (h *rolesHandler) routes(r chi.Router) {
	r.Get("/", h.list)
	r.Get("/{id}", h.get)
	r.Group(func(r chi.Router) {
		r.Use(auth.RequireRole(auth.RoleAdmin))
		r.Post("/", h.create)
		r.Patch("/{id}", h.update)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.del)
	})
}

// peopleIDsByRoleName builds a name -> []personID map by matching the
// people.role text column case-insensitively. Float's Roles API derives
// `people_ids[]` and `people_count` the same way (people whose role matches
// the role's name), and we mirror that without introducing a join table so
// CRUD on roles doesn't need to also rewrite people rows.
func (h *rolesHandler) peopleIDsByRoleName(r *http.Request) (map[string][]string, error) {
	people, err := h.q.ListPeople(r.Context(), false)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, p := range people {
		key := strings.ToLower(strings.TrimSpace(p.Role))
		if key == "" {
			continue
		}
		out[key] = append(out[key], uuidString(p.ID))
	}
	return out, nil
}

func (h *rolesHandler) list(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.ListRoles(r.Context())
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	byName, err := h.peopleIDsByRoleName(r)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]roleDTO, 0, len(rows))
	for _, role := range rows {
		out = append(out, toRoleDTO(role, byName[strings.ToLower(strings.TrimSpace(role.Name))]))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *rolesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	role, err := h.q.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "role not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	byName, err := h.peopleIDsByRoleName(r)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toRoleDTO(role, byName[strings.ToLower(strings.TrimSpace(role.Name))]))
}

func (h *rolesHandler) create(w http.ResponseWriter, r *http.Request) {
	var in roleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "name is required")
		return
	}
	if msg := validateCostRateHistory(in.CostRateHistory); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	rate, err := parseRateInput(in.DefaultHourlyRate)
	if err != nil {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", err.Error())
		return
	}
	if existing, err := h.q.GetRoleByName(r.Context(), name); err == nil {
		WriteProblem(w, r, http.StatusConflict, "duplicate_name", fmt.Sprintf("role %q already exists with id %s", existing.Name, uuidString(existing.ID)))
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		WriteProblem(w, r, http.StatusInternalServerError, "lookup_failed", err.Error())
		return
	}
	history, err := encodeCostRateHistory(in.CostRateHistory)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "encode_failed", err.Error())
		return
	}
	role, err := h.q.CreateRole(r.Context(), db.CreateRoleParams{
		Name:              name,
		DefaultHourlyRate: rate,
		CostRateHistory:   history,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	byName, err := h.peopleIDsByRoleName(r)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toRoleDTO(role, byName[strings.ToLower(role.Name)]))
}

func (h *rolesHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	existing, err := h.q.GetRole(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "role not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	var in roleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}

	name := existing.Name
	if strings.TrimSpace(in.Name) != "" {
		name = strings.TrimSpace(in.Name)
	}
	rate := existing.DefaultHourlyRate
	if in.DefaultHourlyRate != nil {
		parsed, err := parseRateInput(in.DefaultHourlyRate)
		if err != nil {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", err.Error())
			return
		}
		rate = parsed
	}
	history := existing.CostRateHistory
	if in.CostRateHistory != nil {
		if msg := validateCostRateHistory(in.CostRateHistory); msg != "" {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
			return
		}
		encoded, err := encodeCostRateHistory(in.CostRateHistory)
		if err != nil {
			WriteProblem(w, r, http.StatusInternalServerError, "encode_failed", err.Error())
			return
		}
		history = encoded
	}

	if !strings.EqualFold(name, existing.Name) {
		if dup, err := h.q.GetRoleByName(r.Context(), name); err == nil && !uuidEq(dup.ID, existing.ID) {
			WriteProblem(w, r, http.StatusConflict, "duplicate_name", fmt.Sprintf("role %q already exists with id %s", dup.Name, uuidString(dup.ID)))
			return
		} else if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusInternalServerError, "lookup_failed", err.Error())
			return
		}
	}

	updated, err := h.q.UpdateRole(r.Context(), db.UpdateRoleParams{
		ID:                id,
		Name:              name,
		DefaultHourlyRate: rate,
		CostRateHistory:   history,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "role not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	byName, err := h.peopleIDsByRoleName(r)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toRoleDTO(updated, byName[strings.ToLower(updated.Name)]))
}

func (h *rolesHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteRole(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
