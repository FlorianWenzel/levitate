package api

import (
	"reflect"
	"testing"

	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestParseExpand(t *testing.T) {
	cases := []struct {
		in   string
		want projectExpansions
	}{
		{"", projectExpansions{}},
		{"expenses", projectExpansions{expenses: true}},
		{"project_tasks,project_team", projectExpansions{projectTasks: true, projectTeam: true}},
		{"  expenses , project_tasks ,  project_team ", projectExpansions{expenses: true, projectTasks: true, projectTeam: true}},
		{"unknown,expenses,bogus", projectExpansions{expenses: true}},
		{",,", projectExpansions{}},
	}
	for _, tc := range cases {
		got := parseExpand(tc.in)
		if got != tc.want {
			t.Errorf("parseExpand(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func mustPgUUID(t *testing.T, s string) pgtype.UUID {
	t.Helper()
	id, err := pgUUID(s)
	if err != nil {
		t.Fatalf("pgUUID(%q): %v", s, err)
	}
	return id
}

func numericFromInt(t *testing.T, v int) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(rune('0')); err != nil {
		// fall through; we'll set via float64
	}
	got, err := numericFromFloat(float64(v))
	if err != nil {
		t.Fatalf("numericFromFloat(%d): %v", v, err)
	}
	return got
}

// TestMergeExpansions confirms the projection logic groups expansion rows by
// project_id and exposes them as empty arrays when no rows match (rather than
// omitting the key entirely, which would break Float clients that expect the
// array to exist).
func TestMergeExpansions(t *testing.T) {
	projectA := uuid.New().String()
	projectB := uuid.New().String()
	personA := uuid.New().String()
	personB := uuid.New().String()
	taskA1 := uuid.New().String()
	taskA2 := uuid.New().String()
	taskB1 := uuid.New().String()

	out := []projectDTO{
		{ID: projectA, Name: "A"},
		{ID: projectB, Name: "B"},
	}

	tasks := []db.ListProjectTasksByProjectsRow{
		{
			ID:          mustPgUUID(t, taskA1),
			ProjectID:   mustPgUUID(t, projectA),
			PersonID:    mustPgUUID(t, personA),
			HoursPerDay: numericFromInt(t, 4),
			Notes:       "Design sprint",
		},
		{
			ID:          mustPgUUID(t, taskA2),
			ProjectID:   mustPgUUID(t, projectA),
			PersonID:    mustPgUUID(t, personB),
			HoursPerDay: numericFromInt(t, 2),
			Notes:       "Code review",
		},
		{
			ID:          mustPgUUID(t, taskB1),
			ProjectID:   mustPgUUID(t, projectB),
			PersonID:    mustPgUUID(t, personA),
			HoursPerDay: numericFromInt(t, 6),
			Notes:       "",
		},
	}
	team := []db.ListProjectTeamByProjectsRow{
		{
			ProjectID:  mustPgUUID(t, projectA),
			PersonID:   mustPgUUID(t, personA),
			HourlyRate: numericFromInt(t, 260),
		},
		{
			ProjectID:  mustPgUUID(t, projectA),
			PersonID:   mustPgUUID(t, personB),
			HourlyRate: numericFromInt(t, 180),
		},
		// projectB has no team rows
	}

	mergeExpansions(out, projectExpansions{expenses: true, projectTasks: true, projectTeam: true}, tasks, team)

	if got := len(out[0].ProjectTasks); got != 2 {
		t.Fatalf("project A tasks: got %d, want 2", got)
	}
	if got := len(out[1].ProjectTasks); got != 1 {
		t.Fatalf("project B tasks: got %d, want 1", got)
	}
	wantA0 := projectTaskDTO{TaskID: taskA1, Name: "Design sprint", Hours: 4, PeopleID: personA}
	if out[0].ProjectTasks[0] != wantA0 {
		t.Errorf("project A task[0]: got %+v, want %+v", out[0].ProjectTasks[0], wantA0)
	}

	if got := len(out[0].ProjectTeam); got != 2 {
		t.Fatalf("project A team: got %d, want 2", got)
	}
	if out[0].ProjectTeam[0] != (projectTeamDTO{PeopleID: personA, HourlyRate: 260}) {
		t.Errorf("project A team[0]: got %+v", out[0].ProjectTeam[0])
	}
	if got := out[1].ProjectTeam; !reflect.DeepEqual(got, []projectTeamDTO{}) {
		t.Errorf("project B team: got %+v, want []", got)
	}

	if got := out[0].Expenses; !reflect.DeepEqual(got, []projectExpenseDTO{}) {
		t.Errorf("project A expenses: got %+v, want []", got)
	}
}

// TestMergeExpansionsSkipsUnrequested verifies that fields the caller did not
// ask for stay nil (and therefore get omitted from JSON output via omitempty),
// so non-expanded responses keep their existing shape.
func TestMergeExpansionsSkipsUnrequested(t *testing.T) {
	pid := uuid.New().String()
	out := []projectDTO{{ID: pid}}
	mergeExpansions(out, projectExpansions{}, nil, nil)
	if out[0].Expenses != nil || out[0].ProjectTasks != nil || out[0].ProjectTeam != nil {
		t.Fatalf("unrequested expansions were populated: %+v", out[0])
	}
}

// TestMergeExpansionsOrphanRowsIgnored covers the defensive case where a query
// returns a row whose project_id isn't in `out` (e.g. a race between list and
// expansion queries). The orphan row must not panic and must not mutate any
// project's expansion arrays beyond the empty initial state.
func TestMergeExpansionsOrphanRowsIgnored(t *testing.T) {
	pid := uuid.New().String()
	orphan := uuid.New().String()
	out := []projectDTO{{ID: pid}}
	tasks := []db.ListProjectTasksByProjectsRow{
		{
			ID:          mustPgUUID(t, uuid.New().String()),
			ProjectID:   mustPgUUID(t, orphan),
			PersonID:    mustPgUUID(t, uuid.New().String()),
			HoursPerDay: numericFromInt(t, 1),
		},
	}
	mergeExpansions(out, projectExpansions{projectTasks: true}, tasks, nil)
	if got := out[0].ProjectTasks; !reflect.DeepEqual(got, []projectTaskDTO{}) {
		t.Fatalf("orphan row leaked into project: %+v", got)
	}
}
