package api

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// resetHandler truncates all schedule fixtures so e2e tests can start from a clean slate.
// It is gated by LEVITATE_ALLOW_TEST_RESET=true; never expose this in production.
//
// `users` rows are kept (they're synced from OIDC). Every `people` row is wiped — the
// next /api/me call by the OIDC user will recreate their `people` row via
// EnsurePersonForUser, so tests can rely on the seeded admin/member showing up
// after a single round-trip.
func resetHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := pool.Exec(r.Context(), `
			TRUNCATE TABLE logged_time, assignments, time_off, milestones, phases, projects, people, audit_log, deleted_log RESTART IDENTITY CASCADE;
		`)
		if err != nil {
			WriteProblem(w, r, http.StatusInternalServerError, "reset_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	}
}
