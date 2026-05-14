package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const floatDefaultBaseURL = "https://api.float.com/v3"

type floatImportHandler struct {
	q      *db.Queries
	pool   *pgxpool.Pool
	client *http.Client
}

func newFloatImportHandler(q *db.Queries, pool *pgxpool.Pool) *floatImportHandler {
	return &floatImportHandler{
		q:    q,
		pool: pool,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type floatImportInput struct {
	APIToken  string `json:"api_token"`
	BaseURL   string `json:"base_url"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type floatImportResult struct {
	PeopleCreated      int      `json:"people_created"`
	PeopleSkipped      int      `json:"people_skipped"`
	ProjectsCreated    int      `json:"projects_created"`
	ProjectsSkipped    int      `json:"projects_skipped"`
	AssignmentsCreated int      `json:"assignments_created"`
	AssignmentsSkipped int      `json:"assignments_skipped"`
	TimeOffCreated     int      `json:"time_off_created"`
	TimeOffSkipped     int      `json:"time_off_skipped"`
	Warnings           []string `json:"warnings"`
}

type floatPerson struct {
	ID             int       `json:"people_id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	JobTitle       string    `json:"job_title"`
	Active         *int      `json:"active"`
	WorkDaysHours  []float64 `json:"work_days_hours"`
	PeopleTypeID   int       `json:"people_type_id"`
	EmployeeTypeID int       `json:"employee_type"`
}

type floatClient struct {
	ID   int    `json:"client_id"`
	Name string `json:"name"`
}

type floatProject struct {
	ID          int    `json:"project_id"`
	Name        string `json:"name"`
	ClientID    int    `json:"client_id"`
	Color       string `json:"color"`
	Notes       string `json:"notes"`
	Active      *int   `json:"active"`
	NonBillable *int   `json:"non_billable"`
}

type floatTask struct {
	ID        int     `json:"task_id"`
	ProjectID int     `json:"project_id"`
	PersonID  int     `json:"people_id"`
	PersonIDs []int   `json:"people_ids"`
	StartDate string  `json:"start_date"`
	EndDate   string  `json:"end_date"`
	Hours     float64 `json:"hours"`
	Name      string  `json:"name"`
	Notes     string  `json:"notes"`
}

type floatTimeOff struct {
	ID            int     `json:"timeoff_id"`
	TypeID        int     `json:"timeoff_type_id"`
	StartDate     string  `json:"start_date"`
	EndDate       string  `json:"end_date"`
	Hours         float64 `json:"hours"`
	Notes         string  `json:"timeoff_notes"`
	PeopleIDs     []int   `json:"people_ids"`
	FullDay       int     `json:"full_day"`
	RepeatState   int     `json:"repeat_state"`
	RepeatEndDate string  `json:"repeat_end"`
}

type floatTimeOffType struct {
	ID   int    `json:"timeoff_type_id"`
	Name string `json:"name"`
}

func (h *floatImportHandler) importFloat(w http.ResponseWriter, r *http.Request) {
	var in floatImportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_json", err.Error())
		return
	}
	in.APIToken = strings.TrimSpace(in.APIToken)
	if in.APIToken == "" {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "api_token is required")
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if baseURL == "" {
		baseURL = floatDefaultBaseURL
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "base_url must be a valid URL")
		return
	}
	if in.StartDate == "" {
		in.StartDate = time.Now().AddDate(-1, 0, 0).Format(dateLayout)
	}
	if in.EndDate == "" {
		in.EndDate = time.Now().AddDate(1, 0, 0).Format(dateLayout)
	}
	if in.StartDate > in.EndDate {
		WriteProblem(w, r, http.StatusUnprocessableEntity, "validation", "end_date must be on or after start_date")
		return
	}
	fromDate, err := parseDate(in.StartDate)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_start_date", err.Error())
		return
	}
	toDate, err := parseDate(in.EndDate)
	if err != nil {
		WriteProblem(w, r, http.StatusBadRequest, "bad_end_date", err.Error())
		return
	}

	c := floatClientAPI{baseURL: baseURL, token: in.APIToken, http: h.client}
	people, err := fetchFloatPage[floatPerson](r.Context(), c, "/people", nil)
	if err != nil {
		WriteProblem(w, r, http.StatusBadGateway, "float_people_failed", err.Error())
		return
	}
	projects, err := fetchFloatPage[floatProject](r.Context(), c, "/projects", nil)
	if err != nil {
		WriteProblem(w, r, http.StatusBadGateway, "float_projects_failed", err.Error())
		return
	}
	clients, err := fetchFloatPage[floatClient](r.Context(), c, "/clients", nil)
	if err != nil {
		WriteProblem(w, r, http.StatusBadGateway, "float_clients_failed", err.Error())
		return
	}
	tasks, err := fetchFloatPage[floatTask](r.Context(), c, "/tasks", url.Values{
		"start_date": {in.StartDate},
		"end_date":   {in.EndDate},
	})
	if err != nil {
		WriteProblem(w, r, http.StatusBadGateway, "float_tasks_failed", err.Error())
		return
	}
	timeOffs, err := fetchFloatPage[floatTimeOff](r.Context(), c, "/timeoffs", url.Values{
		"start_date": {in.StartDate},
		"end_date":   {in.EndDate},
	})
	if err != nil {
		WriteProblem(w, r, http.StatusBadGateway, "float_timeoffs_failed", err.Error())
		return
	}
	timeOffTypes, err := fetchFloatPage[floatTimeOffType](r.Context(), c, "/timeoff-types", nil)
	timeOffTypeWarning := ""
	if err != nil {
		timeOffTypes = nil
		timeOffTypeWarning = "Float time-off types could not be loaded; imported time off was categorized as other unless the type name was available."
	}

	result, err := h.importFloatData(r.Context(), people, clients, projects, tasks, timeOffs, timeOffTypes, fromDate, toDate)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "import_failed", err.Error())
		return
	}
	if timeOffTypeWarning != "" {
		result.Warnings = append(result.Warnings, timeOffTypeWarning)
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *floatImportHandler) importFloatData(ctx context.Context, people []floatPerson, clients []floatClient, projects []floatProject, tasks []floatTask, timeOffs []floatTimeOff, timeOffTypes []floatTimeOffType, fromDate pgtype.Date, toDate pgtype.Date) (floatImportResult, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return floatImportResult{}, err
	}
	defer tx.Rollback(ctx)

	q := h.q.WithTx(tx)
	result := floatImportResult{}
	peopleByFloatID := map[int]pgtype.UUID{}
	projectsByFloatID := map[int]pgtype.UUID{}

	existingPeople, err := q.ListPeople(ctx, true)
	if err != nil {
		return result, err
	}
	peopleByEmail := map[string]db.Person{}
	peopleByName := map[string]db.Person{}
	for _, p := range existingPeople {
		if key := normalizeKey(p.Email); key != "" {
			peopleByEmail[key] = p
		}
		if key := normalizeKey(p.Name); key != "" {
			peopleByName[key] = p
		}
	}
	for _, fp := range people {
		if fp.ID == 0 || strings.TrimSpace(fp.Name) == "" {
			result.PeopleSkipped++
			continue
		}
		var person db.Person
		var ok bool
		if key := normalizeKey(fp.Email); key != "" {
			person, ok = peopleByEmail[key]
		}
		if !ok {
			person, ok = peopleByName[normalizeKey(fp.Name)]
		}
		if ok {
			peopleByFloatID[fp.ID] = person.ID
			result.PeopleSkipped++
			continue
		}
		cap, _ := numericFromFloat(floatWeeklyCapacity(fp.WorkDaysHours))
		person, err = q.CreatePerson(ctx, db.CreatePersonParams{
			Name:                strings.TrimSpace(fp.Name),
			Email:               strings.TrimSpace(fp.Email),
			Role:                strings.TrimSpace(fp.JobTitle),
			WeeklyCapacityHours: cap,
		})
		if err != nil {
			return result, err
		}
		if fp.Active != nil && *fp.Active == 0 {
			if archived, err := q.ArchivePerson(ctx, person.ID); err == nil {
				person = archived
			}
		}
		peopleByFloatID[fp.ID] = person.ID
		if key := normalizeKey(person.Email); key != "" {
			peopleByEmail[key] = person
		}
		peopleByName[normalizeKey(person.Name)] = person
		result.PeopleCreated++
	}

	clientNames := map[int]string{}
	for _, c := range clients {
		clientNames[c.ID] = c.Name
	}
	existingProjects, err := q.ListProjects(ctx, true)
	if err != nil {
		return result, err
	}
	projectsByKey := map[string]db.Project{}
	for _, p := range existingProjects {
		projectsByKey[projectKey(p.Name, p.Client)] = p
	}
	for _, fp := range projects {
		if fp.ID == 0 || strings.TrimSpace(fp.Name) == "" {
			result.ProjectsSkipped++
			continue
		}
		client := strings.TrimSpace(clientNames[fp.ClientID])
		if project, ok := projectsByKey[projectKey(fp.Name, client)]; ok {
			projectsByFloatID[fp.ID] = project.ID
			result.ProjectsSkipped++
			continue
		}
		color := normalizeFloatColor(fp.Color)
		billable := true
		if fp.NonBillable != nil && *fp.NonBillable == 1 {
			billable = false
		}
		project, err := q.CreateProject(ctx, db.CreateProjectParams{
			Name:     strings.TrimSpace(fp.Name),
			Client:   client,
			Color:    color,
			Notes:    strings.TrimSpace(fp.Notes),
			Billable: billable,
		})
		if err != nil {
			return result, err
		}
		if fp.Active != nil && *fp.Active == 0 {
			if archived, err := q.ArchiveProject(ctx, project.ID); err == nil {
				project = archived
			}
		}
		projectsByFloatID[fp.ID] = project.ID
		projectsByKey[projectKey(project.Name, project.Client)] = project
		result.ProjectsCreated++
	}

	existingAssignments, err := q.ListAssignmentsInRange(ctx, db.ListAssignmentsInRangeParams{FromDate: fromDate, ToDate: toDate})
	if err != nil {
		return result, err
	}
	assignmentKeys := map[string]struct{}{}
	for _, a := range existingAssignments {
		assignmentKeys[assignmentKey(a.PersonID, a.ProjectID, formatDate(a.StartDate), formatDate(a.EndDate), numericFloat(a.HoursPerDay), a.Notes)] = struct{}{}
	}
	for _, ft := range tasks {
		projectID, ok := projectsByFloatID[ft.ProjectID]
		if !ok || ft.StartDate == "" || ft.EndDate == "" || ft.Hours <= 0 {
			result.AssignmentsSkipped++
			continue
		}
		personIDs := ft.PersonIDs
		if ft.PersonID != 0 {
			personIDs = []int{ft.PersonID}
		}
		if len(personIDs) == 0 {
			result.AssignmentsSkipped++
			continue
		}
		startDate, err := parseDate(ft.StartDate)
		if err != nil {
			result.AssignmentsSkipped++
			continue
		}
		endDate, err := parseDate(ft.EndDate)
		if err != nil {
			result.AssignmentsSkipped++
			continue
		}
		hours, _ := numericFromFloat(ft.Hours)
		notes := joinNotes(ft.Name, ft.Notes)
		for _, floatPersonID := range personIDs {
			personID, ok := peopleByFloatID[floatPersonID]
			if !ok {
				result.AssignmentsSkipped++
				continue
			}
			key := assignmentKey(personID, projectID, ft.StartDate, ft.EndDate, ft.Hours, notes)
			if _, exists := assignmentKeys[key]; exists {
				result.AssignmentsSkipped++
				continue
			}
			_, err := q.CreateAssignment(ctx, db.CreateAssignmentParams{
				PersonID:    personID,
				ProjectID:   projectID,
				StartDate:   startDate,
				EndDate:     endDate,
				HoursPerDay: hours,
				Notes:       notes,
			})
			if err != nil {
				return result, err
			}
			assignmentKeys[key] = struct{}{}
			result.AssignmentsCreated++
		}
	}

	existingTimeOff, err := q.ListTimeOffInRange(ctx, db.ListTimeOffInRangeParams{FromDate: fromDate, ToDate: toDate})
	if err != nil {
		return result, err
	}
	timeOffKeys := map[string]struct{}{}
	for _, t := range existingTimeOff {
		timeOffKeys[timeOffKey(t.PersonID, formatDate(t.StartDate), formatDate(t.EndDate), t.Type, t.Notes)] = struct{}{}
	}
	timeOffTypeNames := map[int]string{}
	for _, t := range timeOffTypes {
		timeOffTypeNames[t.ID] = t.Name
	}
	for _, ft := range timeOffs {
		if ft.StartDate == "" || ft.EndDate == "" || len(ft.PeopleIDs) == 0 {
			result.TimeOffSkipped++
			continue
		}
		startDate, err := parseDate(ft.StartDate)
		if err != nil {
			result.TimeOffSkipped++
			continue
		}
		endDate, err := parseDate(ft.EndDate)
		if err != nil {
			result.TimeOffSkipped++
			continue
		}
		typeName := timeOffTypeNames[ft.TypeID]
		typeValue := mapFloatTimeOffType(typeName)
		notes := strings.TrimSpace(ft.Notes)
		if typeName != "" && !strings.Contains(strings.ToLower(notes), strings.ToLower(typeName)) {
			notes = joinNotes(typeName, notes)
		}
		for _, floatPersonID := range ft.PeopleIDs {
			personID, ok := peopleByFloatID[floatPersonID]
			if !ok {
				result.TimeOffSkipped++
				continue
			}
			key := timeOffKey(personID, ft.StartDate, ft.EndDate, typeValue, notes)
			if _, exists := timeOffKeys[key]; exists {
				result.TimeOffSkipped++
				continue
			}
			_, err := q.CreateTimeOff(ctx, db.CreateTimeOffParams{
				PersonID:  personID,
				StartDate: startDate,
				EndDate:   endDate,
				Type:      typeValue,
				Notes:     notes,
			})
			if err != nil {
				return result, err
			}
			timeOffKeys[key] = struct{}{}
			result.TimeOffCreated++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}

type floatClientAPI struct {
	baseURL string
	token   string
	http    *http.Client
}

func fetchFloatPage[T any](ctx context.Context, c floatClientAPI, path string, params url.Values) ([]T, error) {
	var out []T
	page := 1
	for {
		if params == nil {
			params = url.Values{}
		}
		params.Set("page", strconv.Itoa(page))
		params.Set("per-page", "200")
		u := c.baseURL + path + "?" + params.Encode()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "Levitate/1.0 Float Import")
		res, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		var pageRows []T
		decodeErr := json.NewDecoder(res.Body).Decode(&pageRows)
		res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("Float API returned %s for %s", res.Status, path)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		out = append(out, pageRows...)
		pageCount, _ := strconv.Atoi(res.Header.Get("X-Pagination-Page-Count"))
		if pageCount == 0 || page >= pageCount || len(pageRows) == 0 {
			break
		}
		page++
	}
	return out, nil
}

func floatWeeklyCapacity(days []float64) float64 {
	if len(days) == 0 {
		return 40
	}
	total := 0.0
	for _, h := range days {
		if h > 0 {
			total += h
		}
	}
	return total
}

func normalizeFloatColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#64748B"
	}
	if !strings.HasPrefix(color, "#") {
		color = "#" + color
	}
	if len(color) != 7 {
		return "#64748B"
	}
	return strings.ToUpper(color)
}

func normalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func projectKey(name, client string) string {
	return normalizeKey(client) + "\x00" + normalizeKey(name)
}

func assignmentKey(personID, projectID pgtype.UUID, start, end string, hours float64, notes string) string {
	return strings.Join([]string{uuidString(personID), uuidString(projectID), start, end, strconv.FormatFloat(hours, 'f', 2, 64), strings.TrimSpace(notes)}, "\x00")
}

func timeOffKey(personID pgtype.UUID, start, end, typ, notes string) string {
	return strings.Join([]string{uuidString(personID), start, end, typ, strings.TrimSpace(notes)}, "\x00")
}

func joinNotes(parts ...string) string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		key := normalizeKey(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return strings.Join(out, " — ")
}

func mapFloatTimeOffType(name string) string {
	name = strings.ToLower(name)
	switch {
	case strings.Contains(name, "holiday"):
		return "holiday"
	case strings.Contains(name, "sick"):
		return "sick"
	case strings.Contains(name, "vacation") || strings.Contains(name, "leave") || strings.Contains(name, "pto"):
		return "vacation"
	default:
		return "other"
	}
}
