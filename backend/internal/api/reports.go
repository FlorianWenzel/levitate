package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
)

type utilizationCell struct {
	PersonID            string  `json:"person_id"`
	PersonName          string  `json:"person_name"`
	WeeklyCapacityHours float64 `json:"weekly_capacity_hours"`
	WeekStart           string  `json:"week_start"`
	AssignedHours       float64 `json:"assigned_hours"`
	TimeOffHours        float64 `json:"time_off_hours"`
	AvailableHours      float64 `json:"available_hours"`
	UtilizationPct      float64 `json:"utilization_pct"`
	Overallocated       bool    `json:"overallocated"`
}

// computeUtilization builds per-person, per-week utilization rows for the inclusive range.
// Assigned hours = days_overlapping_week * hours_per_day per assignment.
// Time-off is converted to a workday-equivalent proportion of weekly capacity:
//
//	time_off_hours = (workdays_in_week_off / 5) * weekly_capacity_hours
func computeUtilization(
	people []db.Person,
	assignments []db.Assignment,
	timeOff []db.TimeOff,
	from, to time.Time,
) []utilizationCell {
	weeks := weekStartsBetween(from, to)
	out := make([]utilizationCell, 0, len(people)*len(weeks))

	for _, p := range people {
		if p.ArchivedAt.Valid {
			continue
		}
		capacity := numericFloat(p.WeeklyCapacityHours)
		for _, w := range weeks {
			weekEnd := w.AddDate(0, 0, 6) // Sunday inclusive
			cell := utilizationCell{
				PersonID:            uuidString(p.ID),
				PersonName:          p.Name,
				WeeklyCapacityHours: capacity,
				WeekStart:           w.Format(dateLayout),
			}
			for _, a := range assignments {
				if !uuidEq(a.PersonID, p.ID) {
					continue
				}
				// Only Mon–Fri count toward assigned hours. Weekends are
				// neither shown nor billed on the schedule grid.
				days := overlapWorkdays(a.StartDate.Time, a.EndDate.Time, w, weekEnd)
				if days <= 0 {
					continue
				}
				cell.AssignedHours += float64(days) * numericFloat(a.HoursPerDay)
			}
			workdaysOff := 0
			for _, t := range timeOff {
				if !uuidEq(t.PersonID, p.ID) {
					continue
				}
				workdaysOff += overlapWorkdays(t.StartDate.Time, t.EndDate.Time, w, weekEnd)
			}
			if workdaysOff > 0 && capacity > 0 {
				cell.TimeOffHours = (float64(workdaysOff) / 5.0) * capacity
				if cell.TimeOffHours > capacity {
					cell.TimeOffHours = capacity
				}
			}
			cell.AvailableHours = capacity - cell.TimeOffHours
			if cell.AvailableHours < 0 {
				cell.AvailableHours = 0
			}
			if cell.AvailableHours > 0 {
				cell.UtilizationPct = (cell.AssignedHours / cell.AvailableHours) * 100
			}
			cell.Overallocated = cell.AssignedHours > cell.AvailableHours+0.0001
			out = append(out, cell)
		}
	}
	return out
}

func uuidEq(a, b pgtype.UUID) bool { return a.Bytes == b.Bytes }

func weekStartsBetween(from, to time.Time) []time.Time {
	w := mondayOf(from)
	var out []time.Time
	for !w.After(to) {
		out = append(out, w)
		w = w.AddDate(0, 0, 7)
	}
	return out
}

func mondayOf(d time.Time) time.Time {
	// Sunday=0..Saturday=6 → shift so Monday=0
	off := (int(d.Weekday()) + 6) % 7
	return time.Date(d.Year(), d.Month(), d.Day()-off, 0, 0, 0, 0, time.UTC)
}

// overlapDaysAll returns the number of inclusive days where [aStart..aEnd] overlaps [bStart..bEnd].
func overlapDaysAll(aStart, aEnd, bStart, bEnd time.Time) int {
	start := maxTime(aStart, bStart)
	end := minTime(aEnd, bEnd)
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}

// overlapWorkdays counts Mon–Fri days in the overlap of [aStart..aEnd] and [bStart..bEnd].
func overlapWorkdays(aStart, aEnd, bStart, bEnd time.Time) int {
	start := maxTime(aStart, bStart)
	end := minTime(aEnd, bEnd)
	if end.Before(start) {
		return 0
	}
	d := start
	count := 0
	for !d.After(end) {
		w := d.Weekday()
		if w != time.Saturday && w != time.Sunday {
			count++
		}
		d = d.AddDate(0, 0, 1)
	}
	return count
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

type reportsHandler struct {
	q *db.Queries
}

func newReportsHandler(q *db.Queries) *reportsHandler { return &reportsHandler{q: q} }

func (h *reportsHandler) utilizationJSON(w http.ResponseWriter, r *http.Request) {
	cells, err := h.loadUtilization(r)
	if err != nil {
		writeRangeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cells)
}

type rangeError struct {
	status int
	title  string
	detail string
}

func (e rangeError) Error() string { return e.detail }

func writeRangeError(w http.ResponseWriter, r *http.Request, err error) {
	if re, ok := err.(rangeError); ok {
		WriteProblem(w, r, re.status, re.title, re.detail)
		return
	}
	WriteProblem(w, r, http.StatusInternalServerError, "report_failed", err.Error())
}

func (h *reportsHandler) utilizationCSV(w http.ResponseWriter, r *http.Request) {
	cells, err := h.loadUtilization(r)
	if err != nil {
		writeRangeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="utilization.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"person_name", "person_id", "weekly_capacity_hours", "week_start",
		"assigned_hours", "time_off_hours", "available_hours", "utilization_pct", "overallocated",
	})
	for _, c := range cells {
		_ = cw.Write([]string{
			c.PersonName,
			c.PersonID,
			fmtFloat(c.WeeklyCapacityHours),
			c.WeekStart,
			fmtFloat(c.AssignedHours),
			fmtFloat(c.TimeOffHours),
			fmtFloat(c.AvailableHours),
			fmtFloat(c.UtilizationPct),
			strconv.FormatBool(c.Overallocated),
		})
	}
}

func (h *reportsHandler) assignmentsCSV(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		WriteProblem(w, r, http.StatusBadRequest, "missing_range", "from and to query params are required (YYYY-MM-DD)")
		return
	}
	fromDate, err := parseDate(fromStr)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_from", err.Error())
		return
	}
	toDate, err := parseDate(toStr)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_to", err.Error())
		return
	}
	people, err := h.q.ListPeople(r.Context(), true)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	projects, err := h.q.ListProjects(r.Context(), true)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	rows, err := h.q.ListAssignmentsInRange(r.Context(), db.ListAssignmentsInRangeParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "load_failed", err.Error())
		return
	}
	pIdx := map[[16]byte]string{}
	for _, p := range people {
		pIdx[p.ID.Bytes] = p.Name
	}
	prIdx := map[[16]byte]db.Project{}
	for _, p := range projects {
		prIdx[p.ID.Bytes] = p
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="assignments.csv"`)
	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{
		"assignment_id", "person_name", "person_id", "project_name", "client",
		"start_date", "end_date", "days", "hours_per_day", "total_hours", "notes",
	})
	for _, a := range rows {
		// Assignments are billed in workdays only.
		days := overlapWorkdays(a.StartDate.Time, a.EndDate.Time, a.StartDate.Time, a.EndDate.Time)
		hrs := numericFloat(a.HoursPerDay)
		project := prIdx[a.ProjectID.Bytes]
		_ = cw.Write([]string{
			uuidString(a.ID),
			pIdx[a.PersonID.Bytes],
			uuidString(a.PersonID),
			project.Name,
			project.Client,
			formatDate(a.StartDate),
			formatDate(a.EndDate),
			strconv.Itoa(days),
			fmtFloat(hrs),
			fmtFloat(float64(days) * hrs),
			a.Notes,
		})
	}
}

func fmtFloat(f float64) string {
	return fmt.Sprintf("%g", f)
}

func (h *reportsHandler) loadUtilization(r *http.Request) ([]utilizationCell, error) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	if fromStr == "" || toStr == "" {
		return nil, rangeError{http.StatusBadRequest, "missing_range", "from and to query params are required (YYYY-MM-DD)"}
	}
	fromDate, err := parseDate(fromStr)
	if err != nil {
		return nil, rangeError{http.StatusBadRequest, "bad_from", err.Error()}
	}
	toDate, err := parseDate(toStr)
	if err != nil {
		return nil, rangeError{http.StatusBadRequest, "bad_to", err.Error()}
	}
	if toDate.Time.Before(fromDate.Time) {
		return nil, rangeError{http.StatusBadRequest, "bad_range", "to must be on or after from"}
	}
	people, err := h.q.ListPeople(r.Context(), false)
	if err != nil {
		return nil, err
	}
	assignments, err := h.q.ListAssignmentsInRange(r.Context(), db.ListAssignmentsInRangeParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		return nil, err
	}
	timeOff, err := h.q.ListTimeOffInRange(r.Context(), db.ListTimeOffInRangeParams{
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		return nil, err
	}
	return computeUtilization(people, assignments, timeOff, fromDate.Time, toDate.Time), nil
}
