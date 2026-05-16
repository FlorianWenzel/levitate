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

// Float status_type_id constants: 1=Home, 2=Travel, 3=Custom, 4=Office.
const (
	statusTypeHome   = 1
	statusTypeTravel = 2
	statusTypeCustom = 3
	statusTypeOffice = 4
)

// statusDTO mirrors Float's Statuses response shape: it preserves Float's
// field names (people_id, status_type_id, repeat_state, ...) so a Float
// integration can be a direct field-by-field passthrough.
type statusDTO struct {
	ID            string    `json:"id"`
	StatusTypeID  int       `json:"status_type_id"`
	PeopleID      string    `json:"people_id"`
	StatusName    string    `json:"status_name"`
	StartDate     string    `json:"start_date"`
	EndDate       string    `json:"end_date"`
	RepeatState   int       `json:"repeat_state"`
	RepeatEndDate *string   `json:"repeat_end_date"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toStatusDTO(s db.UserStatus) statusDTO {
	var repeatEnd *string
	if s.RepeatEndDate.Valid {
		v := formatDate(s.RepeatEndDate)
		repeatEnd = &v
	}
	return statusDTO{
		ID:            uuidString(s.ID),
		StatusTypeID:  int(s.StatusTypeID),
		PeopleID:      uuidString(s.PersonID),
		StatusName:    s.StatusName,
		StartDate:     formatDate(s.StartDate),
		EndDate:       formatDate(s.EndDate),
		RepeatState:   int(s.RepeatState),
		RepeatEndDate: repeatEnd,
		CreatedAt:     ts(s.CreatedAt),
		UpdatedAt:     ts(s.UpdatedAt),
	}
}

type statusInput struct {
	StatusTypeID  *int    `json:"status_type_id"`
	PeopleID      string  `json:"people_id"`
	StatusName    string  `json:"status_name"`
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	RepeatState   *int    `json:"repeat_state"`
	RepeatEndDate *string `json:"repeat_end_date"`
}

func (in statusInput) validate() string {
	if in.StatusTypeID == nil {
		return "status_type_id is required"
	}
	if *in.StatusTypeID < statusTypeHome || *in.StatusTypeID > statusTypeOffice {
		return "status_type_id must be 1 (Home), 2 (Travel), 3 (Custom), or 4 (Office)"
	}
	if *in.StatusTypeID == statusTypeCustom && strings.TrimSpace(in.StatusName) == "" {
		return "status_name is required when status_type_id is 3 (Custom)"
	}
	if strings.TrimSpace(in.PeopleID) == "" {
		return "people_id is required"
	}
	if in.StartDate == "" || in.EndDate == "" {
		return "start_date and end_date are required"
	}
	start, err := parseDate(in.StartDate)
	if err != nil {
		return "start_date must be YYYY-MM-DD"
	}
	end, err := parseDate(in.EndDate)
	if err != nil {
		return "end_date must be YYYY-MM-DD"
	}
	if start.Time.After(end.Time) {
		return "end_date must be on or after start_date"
	}
	if in.RepeatState != nil && (*in.RepeatState < 0 || *in.RepeatState > 4) {
		return "repeat_state must be between 0 and 4"
	}
	if in.RepeatState != nil && *in.RepeatState > 0 {
		if in.RepeatEndDate == nil || *in.RepeatEndDate == "" {
			return "repeat_end_date is required when repeat_state > 0"
		}
	}
	if in.RepeatEndDate != nil && *in.RepeatEndDate != "" {
		rEnd, err := parseDate(*in.RepeatEndDate)
		if err != nil {
			return "repeat_end_date must be YYYY-MM-DD"
		}
		if rEnd.Time.Before(end.Time) {
			return "repeat_end_date must be on or after end_date"
		}
	}
	return ""
}

func (in statusInput) repeatEndPG() pgtype.Date {
	var d pgtype.Date
	if in.RepeatEndDate == nil || *in.RepeatEndDate == "" {
		return d
	}
	parsed, err := parseDate(*in.RepeatEndDate)
	if err != nil {
		return d
	}
	return parsed
}

type statusesHandler struct {
	q *db.Queries
}

func newStatusesHandler(q *db.Queries) *statusesHandler { return &statusesHandler{q: q} }

func (h *statusesHandler) routes(r chi.Router) {
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

func (h *statusesHandler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var params db.ListUserStatusesParams
	if v := q.Get("people_id"); v != "" {
		id, err := pgUUID(v)
		if err != nil {
			WriteProblem(w, r, http.StatusBadRequest, "bad_people_id", err.Error())
			return
		}
		params.PersonID = id
	}
	if v := q.Get("status_type_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < statusTypeHome || n > statusTypeOffice {
			WriteProblem(w, r, http.StatusBadRequest, "bad_status_type_id", "status_type_id must be 1..4")
			return
		}
		params.StatusTypeID = pgtype.Int2{Int16: int16(n), Valid: true}
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

	rows, err := h.q.ListUserStatuses(r.Context(), params)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "list_failed", err.Error())
		return
	}
	out := make([]statusDTO, 0, len(rows))
	for _, s := range rows {
		out = append(out, toStatusDTO(s))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *statusesHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	s, err := h.q.GetUserStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "status not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toStatusDTO(s))
}

func (h *statusesHandler) create(w http.ResponseWriter, r *http.Request) {
	var in statusInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if msg := in.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}
	personID, err := pgUUID(in.PeopleID)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_people_id", err.Error())
		return
	}
	if _, err := h.q.GetPerson(r.Context(), personID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "people_id not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "lookup_failed", err.Error())
		return
	}
	startDate, _ := parseDate(in.StartDate)
	endDate, _ := parseDate(in.EndDate)
	statusName := ""
	if *in.StatusTypeID == statusTypeCustom {
		statusName = strings.TrimSpace(in.StatusName)
	}
	repeat := 0
	if in.RepeatState != nil {
		repeat = *in.RepeatState
	}
	s, err := h.q.CreateUserStatus(r.Context(), db.CreateUserStatusParams{
		PersonID:      personID,
		StatusTypeID:  int16(*in.StatusTypeID),
		StatusName:    statusName,
		StartDate:     startDate,
		EndDate:       endDate,
		RepeatState:   int16(repeat),
		RepeatEndDate: in.repeatEndPG(),
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "create_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toStatusDTO(s))
}

func (h *statusesHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	existing, err := h.q.GetUserStatus(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "status not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "get_failed", err.Error())
		return
	}

	// PATCH semantics: fall back to existing values for omitted fields, then
	// run the same validation as create on the merged document so partial
	// updates can't sneak in an invalid combination (e.g. repeat_state without
	// a repeat_end_date).
	merged := statusInput{
		StatusTypeID:  intPtr(int(existing.StatusTypeID)),
		PeopleID:      uuidString(existing.PersonID),
		StatusName:    existing.StatusName,
		StartDate:     formatDate(existing.StartDate),
		EndDate:       formatDate(existing.EndDate),
		RepeatState:   intPtr(int(existing.RepeatState)),
		RepeatEndDate: nil,
	}
	if existing.RepeatEndDate.Valid {
		v := formatDate(existing.RepeatEndDate)
		merged.RepeatEndDate = &v
	}

	var patch statusInput
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	if patch.StatusTypeID != nil {
		merged.StatusTypeID = patch.StatusTypeID
		if *patch.StatusTypeID != statusTypeCustom {
			merged.StatusName = ""
		}
	}
	if strings.TrimSpace(patch.PeopleID) != "" {
		merged.PeopleID = patch.PeopleID
	}
	if patch.StatusName != "" {
		merged.StatusName = patch.StatusName
	}
	if patch.StartDate != "" {
		merged.StartDate = patch.StartDate
	}
	if patch.EndDate != "" {
		merged.EndDate = patch.EndDate
	}
	if patch.RepeatState != nil {
		merged.RepeatState = patch.RepeatState
		if *patch.RepeatState == 0 {
			merged.RepeatEndDate = nil
		}
	}
	if patch.RepeatEndDate != nil {
		if *patch.RepeatEndDate == "" {
			merged.RepeatEndDate = nil
		} else {
			merged.RepeatEndDate = patch.RepeatEndDate
		}
	}

	if msg := merged.validate(); msg != "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", msg)
		return
	}

	personID, err := pgUUID(merged.PeopleID)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_people_id", err.Error())
		return
	}
	if !uuidEq(personID, existing.PersonID) {
		if _, err := h.q.GetPerson(r.Context(), personID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "people_id not found")
				return
			}
			WriteProblem(w, r, http.StatusInternalServerError, "lookup_failed", err.Error())
			return
		}
	}

	startDate, _ := parseDate(merged.StartDate)
	endDate, _ := parseDate(merged.EndDate)
	statusName := ""
	if *merged.StatusTypeID == statusTypeCustom {
		statusName = strings.TrimSpace(merged.StatusName)
	}
	repeat := 0
	if merged.RepeatState != nil {
		repeat = *merged.RepeatState
	}
	s, err := h.q.UpdateUserStatus(r.Context(), db.UpdateUserStatusParams{
		ID:            id,
		PersonID:      personID,
		StatusTypeID:  int16(*merged.StatusTypeID),
		StatusName:    statusName,
		StartDate:     startDate,
		EndDate:       endDate,
		RepeatState:   int16(repeat),
		RepeatEndDate: merged.repeatEndPG(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteProblem(w, r, http.StatusNotFound, "not_found", "status not found")
			return
		}
		WriteProblem(w, r, http.StatusInternalServerError, "update_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toStatusDTO(s))
}

func (h *statusesHandler) del(w http.ResponseWriter, r *http.Request) {
	id, err := pgUUID(chi.URLParam(r, "id"))
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_id", err.Error())
		return
	}
	if err := h.q.DeleteUserStatus(r.Context(), id); err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "delete_failed", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func intPtr(v int) *int { return &v }
