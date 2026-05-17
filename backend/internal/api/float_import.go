package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/jackc/pgx/v5"
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
	AssignmentsDeleted int      `json:"assignments_deleted"`
	TimeOffCreated     int      `json:"time_off_created"`
	TimeOffSkipped     int      `json:"time_off_skipped"`
	TimeOffDeleted     int      `json:"time_off_deleted"`
	MilestonesCreated  int      `json:"milestones_created"`
	MilestonesSkipped  int      `json:"milestones_skipped"`
	PhasesCreated      int      `json:"phases_created"`
	PhasesSkipped      int      `json:"phases_skipped"`
	LoggedTimeCreated  int      `json:"logged_time_created"`
	LoggedTimeSkipped  int      `json:"logged_time_skipped"`
	LoggedTimeDeleted  int      `json:"logged_time_deleted"`
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
	ID             int      `json:"project_id"`
	Name           string   `json:"name"`
	ClientID       int      `json:"client_id"`
	Color          string   `json:"color"`
	Notes          string   `json:"notes"`
	Active         *int     `json:"active"`
	NonBillable    *int     `json:"non_billable"`
	BudgetType     *int     `json:"budget_type"`
	BudgetTotal    *float64 `json:"budget_total"`
	BudgetPriority *int     `json:"budget_priority"`
	Tags           []string `json:"tags"`
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

type floatPhase struct {
	ID                int     `json:"phase_id"`
	ProjectID         int     `json:"project_id"`
	Name              string  `json:"name"`
	Color             string  `json:"color"`
	Notes             string  `json:"notes"`
	StartDate         string  `json:"start_date"`
	EndDate           string  `json:"end_date"`
	BudgetTotal       float64 `json:"budget_total"`
	DefaultHourlyRate float64 `json:"default_hourly_rate"`
	NonBillable       *int    `json:"non_billable"`
	Status            *int    `json:"status"`
	Active            *int    `json:"active"`
}

type floatMilestone struct {
	ID        int    `json:"milestone_id"`
	Name      string `json:"name"`
	ProjectID int    `json:"project_id"`
	PhaseID   int    `json:"phase_id"`
	Date      string `json:"date"`
	EndDate   string `json:"end_date"`
}

type floatLoggedTime struct {
	ID         int64   `json:"logged_time_id"`
	PersonID   int     `json:"people_id"`
	ProjectID  int     `json:"project_id"`
	Date       string  `json:"date"`
	Hours      float64 `json:"hours"`
	Billable   *int    `json:"billable"`
	Notes      string  `json:"notes"`
	Locked     *int    `json:"locked"`
	LockedDate string  `json:"locked_date"`
}

// floatDeletedEntry models Float's /deleted/<entity> response rows.
type floatDeletedEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"timestamp"`
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

	milestones, err := fetchFloatPage[floatMilestone](r.Context(), c, "/milestones", nil)
	milestonesWarning := ""
	if err != nil {
		milestones = nil
		milestonesWarning = "Float milestones could not be loaded; imported projects were created without milestones."
	}

	phases, err := fetchFloatPage[floatPhase](r.Context(), c, "/phases", nil)
	phasesWarning := ""
	if err != nil {
		phases = nil
		phasesWarning = "Float phases could not be loaded; imported projects were created without phases."
	}

	loggedTime, err := fetchFloatPage[floatLoggedTime](r.Context(), c, "/logged-time", url.Values{
		"start_date": {in.StartDate},
		"end_date":   {in.EndDate},
	})
	loggedTimeWarning := ""
	if err != nil {
		loggedTime = nil
		loggedTimeWarning = "Float logged time could not be loaded; timesheet entries were not imported."
	}

	// Float's delete log (72h retention) tells us which remote tasks/timeoffs
	// have been deleted since the last sync. We surface these as warnings on
	// failure rather than aborting the import — a working import that skips
	// reconciliation is more useful than no import at all.
	deletedTasks, deletedTasksWarning := fetchFloatDeletedSafe(r.Context(), c, "/deleted/tasks")
	deletedTimeOffs, deletedTimeOffsWarning := fetchFloatDeletedSafe(r.Context(), c, "/deleted/timeoffs")
	deletedLoggedTime, deletedLoggedTimeWarning := fetchFloatDeletedSafe(r.Context(), c, "/deleted/logged-time")

	// Attribute imported timesheet rows to the admin who triggered the import,
	// matching Float's contract where every LoggedTime entry carries a
	// `created_by` / `modified_by` user id. Lookup failure (e.g. a missing
	// users row on the very first request) leaves the columns NULL.
	var actorID pgtype.UUID
	if p, ok := auth.FromContext(r.Context()); ok && p.Subject != "" {
		if u, err := h.q.GetUserBySub(r.Context(), p.Subject); err == nil {
			actorID = u.ID
		}
	}

	result, err := h.importFloatData(r.Context(), people, clients, projects, tasks, timeOffs, timeOffTypes, milestones, phases, loggedTime, deletedTasks, deletedTimeOffs, deletedLoggedTime, fromDate, toDate, actorID)
	if err != nil {
		WriteProblem(w, r, http.StatusInternalServerError, "import_failed", err.Error())
		return
	}
	if timeOffTypeWarning != "" {
		result.Warnings = append(result.Warnings, timeOffTypeWarning)
	}
	if milestonesWarning != "" {
		result.Warnings = append(result.Warnings, milestonesWarning)
	}
	if phasesWarning != "" {
		result.Warnings = append(result.Warnings, phasesWarning)
	}
	if deletedTasksWarning != "" {
		result.Warnings = append(result.Warnings, deletedTasksWarning)
	}
	if deletedTimeOffsWarning != "" {
		result.Warnings = append(result.Warnings, deletedTimeOffsWarning)
	}
	if loggedTimeWarning != "" {
		result.Warnings = append(result.Warnings, loggedTimeWarning)
	}
	if deletedLoggedTimeWarning != "" {
		result.Warnings = append(result.Warnings, deletedLoggedTimeWarning)
	}
	writeJSON(w, http.StatusOK, result)
}

// fetchFloatDeletedSafe paginates a Float /deleted/* endpoint and returns the
// IDs to reconcile. Float advertises cursor-based pagination here (cursor +
// limit, max 500), but if the endpoint is unavailable we return a soft
// warning instead of aborting the whole import.
func fetchFloatDeletedSafe(ctx context.Context, c floatClientAPI, path string) ([]int64, string) {
	rows, err := fetchFloatDeleted(ctx, c, path)
	if err != nil {
		return nil, fmt.Sprintf("Float %s could not be loaded; remote deletions were not reconciled.", path)
	}
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		if r.ID != 0 {
			ids = append(ids, r.ID)
		}
	}
	return ids, ""
}

func (h *floatImportHandler) importFloatData(ctx context.Context, people []floatPerson, clients []floatClient, projects []floatProject, tasks []floatTask, timeOffs []floatTimeOff, timeOffTypes []floatTimeOffType, milestones []floatMilestone, phases []floatPhase, loggedTime []floatLoggedTime, deletedTaskIDs []int64, deletedTimeOffIDs []int64, deletedLoggedTimeIDs []int64, fromDate pgtype.Date, toDate pgtype.Date, actorID pgtype.UUID) (floatImportResult, error) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return floatImportResult{}, err
	}
	defer tx.Rollback(ctx)

	q := h.q.WithTx(tx)
	result := floatImportResult{}
	peopleByFloatID := map[int]pgtype.UUID{}
	projectsByFloatID := map[int]pgtype.UUID{}

	// Reconcile remote deletions first so a remote delete+recreate (same
	// natural key) doesn't get short-circuited by our dedup logic below.
	if len(deletedTaskIDs) > 0 {
		before, err := countRowsByFloatID(ctx, tx, "assignments", deletedTaskIDs)
		if err != nil {
			return result, err
		}
		if err := q.DeleteAssignmentsByFloatID(ctx, deletedTaskIDs); err != nil {
			return result, err
		}
		result.AssignmentsDeleted = before
	}
	if len(deletedTimeOffIDs) > 0 {
		before, err := countRowsByFloatID(ctx, tx, "time_off", deletedTimeOffIDs)
		if err != nil {
			return result, err
		}
		if err := q.DeleteTimeOffByFloatID(ctx, deletedTimeOffIDs); err != nil {
			return result, err
		}
		result.TimeOffDeleted = before
	}
	if len(deletedLoggedTimeIDs) > 0 {
		before, err := countRowsByFloatID(ctx, tx, "logged_time", deletedLoggedTimeIDs)
		if err != nil {
			return result, err
		}
		if err := q.DeleteLoggedTimeByFloatID(ctx, deletedLoggedTimeIDs); err != nil {
			return result, err
		}
		result.LoggedTimeDeleted = before
	}

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
		params := db.CreateProjectParams{
			Name:     strings.TrimSpace(fp.Name),
			Client:   client,
			Color:    color,
			Notes:    strings.TrimSpace(fp.Notes),
			Billable: billable,
			Tags:     normalizeFloatTags(fp.Tags),
		}
		if fp.BudgetType != nil && *fp.BudgetType >= projectBudgetTypeMin && *fp.BudgetType <= projectBudgetTypeMax {
			params.BudgetType = pgtype.Int2{Int16: int16(*fp.BudgetType), Valid: true}
		}
		if fp.BudgetPriority != nil && *fp.BudgetPriority >= projectBudgetPriorityMin && *fp.BudgetPriority <= projectBudgetPriorityMax {
			params.BudgetPriority = pgtype.Int2{Int16: int16(*fp.BudgetPriority), Valid: true}
		}
		if fp.BudgetTotal != nil && *fp.BudgetTotal >= 0 {
			if n, err := numericFromFloat(*fp.BudgetTotal); err == nil {
				params.BudgetTotal = n
			}
		}
		project, err := q.CreateProject(ctx, params)
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
			created, err := q.CreateAssignment(ctx, db.CreateAssignmentParams{
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
			if ft.ID != 0 {
				if err := q.SetAssignmentFloatID(ctx, created.ID, int64(ft.ID)); err != nil {
					return result, err
				}
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
			created, err := q.CreateTimeOff(ctx, db.CreateTimeOffParams{
				PersonID:  personID,
				StartDate: startDate,
				EndDate:   endDate,
				Type:      typeValue,
				Notes:     notes,
			})
			if err != nil {
				return result, err
			}
			if ft.ID != 0 {
				if err := q.SetTimeOffFloatID(ctx, created.ID, int64(ft.ID)); err != nil {
					return result, err
				}
			}
			timeOffKeys[key] = struct{}{}
			result.TimeOffCreated++
		}
	}

	phasesByFloatID := map[int]pgtype.UUID{}
	for _, fp := range phases {
		projectID, ok := projectsByFloatID[fp.ProjectID]
		if !ok || fp.ID == 0 || strings.TrimSpace(fp.Name) == "" {
			result.PhasesSkipped++
			continue
		}
		// Skip if we've already imported this phase (matched by Float ID).
		existing, err := q.GetPhaseByFloatID(ctx, pgtype.Int8{Int64: int64(fp.ID), Valid: true})
		if err == nil {
			phasesByFloatID[fp.ID] = existing.ID
			result.PhasesSkipped++
			continue
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return result, err
		}
		// Also dedupe by (project, name) so a re-import without Float ID
		// linkage doesn't create duplicates.
		existingPhases, err := q.ListPhasesByProject(ctx, projectID)
		if err != nil {
			return result, err
		}
		dupID := pgtype.UUID{}
		for _, ep := range existingPhases {
			if normalizeKey(ep.Name) == normalizeKey(fp.Name) {
				dupID = ep.ID
				break
			}
		}
		if dupID.Valid {
			phasesByFloatID[fp.ID] = dupID
			if err := q.SetPhaseFloatID(ctx, dupID, int64(fp.ID)); err != nil {
				return result, err
			}
			result.PhasesSkipped++
			continue
		}
		params := db.CreatePhaseParams{
			ProjectID: projectID,
			Name:      strings.TrimSpace(fp.Name),
			Color:     normalizeFloatColor(fp.Color),
			Notes:     strings.TrimSpace(fp.Notes),
			Billable:  true,
			Status:    2,
		}
		if fp.StartDate != "" {
			if parsed, err := parseDate(fp.StartDate); err == nil {
				params.StartDate = parsed
			}
		}
		if fp.EndDate != "" {
			if parsed, err := parseDate(fp.EndDate); err == nil {
				params.EndDate = parsed
			}
		}
		if fp.NonBillable != nil && *fp.NonBillable == 1 {
			params.Billable = false
		}
		if fp.Status != nil {
			params.Status = int16(*fp.Status)
		}
		if n, err := numericFromFloat(fp.BudgetTotal); err == nil {
			params.BudgetTotal = n
		}
		if n, err := numericFromFloat(fp.DefaultHourlyRate); err == nil {
			params.DefaultHourlyRate = n
		}
		created, err := q.CreatePhase(ctx, params)
		if err != nil {
			return result, err
		}
		if err := q.SetPhaseFloatID(ctx, created.ID, int64(fp.ID)); err != nil {
			return result, err
		}
		if fp.Active != nil && *fp.Active == 0 {
			if archived, err := q.ArchivePhase(ctx, created.ID); err == nil {
				created = archived
			}
		}
		phasesByFloatID[fp.ID] = created.ID
		result.PhasesCreated++
	}

	for _, fm := range milestones {
		projectID, ok := projectsByFloatID[fm.ProjectID]
		if !ok {
			result.MilestonesSkipped++
			continue
		}
		name := strings.TrimSpace(fm.Name)
		if name == "" || fm.Date == "" {
			result.MilestonesSkipped++
			continue
		}
		date, err := parseDate(fm.Date)
		if err != nil {
			result.MilestonesSkipped++
			continue
		}
		existing, err := q.ListMilestonesByProject(ctx, projectID)
		if err != nil {
			return result, err
		}
		alreadyExists := false
		for _, em := range existing {
			if normalizeKey(em.Name) == normalizeKey(name) && formatDate(em.Date) == fm.Date {
				alreadyExists = true
				break
			}
		}
		if alreadyExists {
			result.MilestonesSkipped++
			continue
		}
		var endDate pgtype.Date
		if fm.EndDate != "" {
			parsed, err := parseDate(fm.EndDate)
			if err == nil {
				endDate = parsed
			}
		}
		var phaseID pgtype.UUID
		if fm.PhaseID != 0 {
			if pid, ok := phasesByFloatID[fm.PhaseID]; ok {
				phaseID = pid
			}
		}
		if _, err := q.CreateMilestone(ctx, db.CreateMilestoneParams{
			ProjectID: projectID,
			PhaseID:   phaseID,
			Name:      name,
			Date:      date,
			EndDate:   endDate,
		}); err != nil {
			return result, err
		}
		result.MilestonesCreated++
	}

	// Track billable inheritance per Float project so we can stamp the value
	// on each imported logged-time row (Float's API treats billability as a
	// read-only projection of the project/phase/task).
	projectBillableByFloatID := map[int]bool{}
	for _, fp := range projects {
		billable := true
		if fp.NonBillable != nil && *fp.NonBillable == 1 {
			billable = false
		}
		projectBillableByFloatID[fp.ID] = billable
	}

	for _, fl := range loggedTime {
		if fl.PersonID == 0 || fl.Date == "" || fl.Hours <= 0 {
			result.LoggedTimeSkipped++
			continue
		}
		personID, ok := peopleByFloatID[fl.PersonID]
		if !ok {
			result.LoggedTimeSkipped++
			continue
		}
		date, err := parseDate(fl.Date)
		if err != nil {
			result.LoggedTimeSkipped++
			continue
		}
		hours, _ := numericFromFloat(fl.Hours)

		var projectID pgtype.UUID
		billable := false
		if fl.ProjectID != 0 {
			if pid, ok := projectsByFloatID[fl.ProjectID]; ok {
				projectID = pid
				if b, ok := projectBillableByFloatID[fl.ProjectID]; ok {
					billable = b
				}
			}
		}
		// Float reports billability on the logged-time row too; trust it if
		// present (e.g. the project link wasn't resolvable), otherwise we
		// inherit from the project we just looked up.
		if fl.Billable != nil {
			billable = *fl.Billable == 1
		}
		notes := strings.TrimSpace(fl.Notes)

		locked := fl.Locked != nil && *fl.Locked == 1

		// Upsert by Float ID: if we've already imported this row, update in
		// place so a re-sync picks up edits made in Float.
		if fl.ID != 0 {
			existing, err := q.GetLoggedTimeByFloatID(ctx, pgtype.Int8{Int64: fl.ID, Valid: true})
			if err == nil {
				// Float's locked flag may change between syncs (an entry can
				// be locked once a project's timesheet window closes, and
				// re-opened). Reflect both transitions locally.
				if !existing.Locked && locked {
					if _, err := q.LockLoggedTime(ctx, existing.ID, actorID); err != nil {
						return result, err
					}
				} else if existing.Locked && !locked {
					if _, err := q.UnlockLoggedTime(ctx, existing.ID, actorID); err != nil {
						return result, err
					}
				}
				if !locked {
					// Locked entries are immutable on our side too; skip the
					// content update to avoid clobbering an already-locked
					// snapshot.
					if _, err := q.UpdateLoggedTime(ctx, db.UpdateLoggedTimeParams{
						ID:        existing.ID,
						Date:      date,
						Hours:     hours,
						Billable:  billable,
						Notes:     notes,
						ProjectID: projectID,
						ActorID:   actorID,
					}); err != nil {
						return result, err
					}
				}
				result.LoggedTimeSkipped++
				continue
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return result, err
			}
		}

		created, err := q.CreateLoggedTime(ctx, db.CreateLoggedTimeParams{
			PersonID:  personID,
			Date:      date,
			Hours:     hours,
			Billable:  billable,
			Notes:     notes,
			ProjectID: projectID,
			ActorID:   actorID,
		})
		if err != nil {
			return result, err
		}
		if fl.ID != 0 {
			if err := q.SetLoggedTimeFloatID(ctx, created.ID, fl.ID); err != nil {
				return result, err
			}
		}
		if locked {
			if _, err := q.LockLoggedTime(ctx, created.ID, actorID); err != nil {
				return result, err
			}
		}
		result.LoggedTimeCreated++
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

// countRowsByFloatID returns how many rows in the given table currently match
// the supplied Float IDs. We use this to report how many local rows the
// remote-deletion reconciliation actually removed (vs. how many IDs Float
// listed — Float keeps deletions for 72h and re-delivers, so we'll often see
// IDs that no longer exist locally).
func countRowsByFloatID(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, table string, floatIDs []int64) (int, error) {
	if len(floatIDs) == 0 {
		return 0, nil
	}
	var n int
	q := "SELECT count(*) FROM " + table + " WHERE float_id = ANY($1::bigint[])"
	if err := tx.QueryRow(ctx, q, floatIDs).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// fetchFloatDeleted paginates Float's cursor-based /deleted/<entity>
// endpoint, accumulating all rows up to a hard cap on iterations.
func fetchFloatDeleted(ctx context.Context, c floatClientAPI, path string) ([]floatDeletedEntry, error) {
	var out []floatDeletedEntry
	cursor := ""
	for i := 0; i < 200; i++ {
		params := url.Values{}
		params.Set("limit", "500")
		if cursor != "" {
			params.Set("cursor", cursor)
		}
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
		var page struct {
			Data []floatDeletedEntry `json:"data"`
		}
		decodeErr := json.NewDecoder(res.Body).Decode(&page)
		nextCursor := res.Header.Get("X-Pagination-Next-Cursor")
		hasMore := strings.EqualFold(res.Header.Get("X-Pagination-Has-More"), "true")
		status := res.StatusCode
		res.Body.Close()
		if status < 200 || status >= 300 {
			return nil, fmt.Errorf("Float API returned %d for %s", status, path)
		}
		if decodeErr != nil {
			return nil, decodeErr
		}
		out = append(out, page.Data...)
		if !hasMore || nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}
	return out, nil
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

// normalizeFloatTags mirrors the API-side projectInput.normalizedTags: trims
// whitespace, drops empties, and de-duplicates case-insensitively. Returns an
// empty (non-nil) slice when no usable tags remain so the persisted value is
// always a concrete empty array rather than NULL.
func normalizeFloatTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, t := range tags {
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
