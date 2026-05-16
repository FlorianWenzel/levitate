package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/florianwenzel/levitate/backend/internal/auth"
	"github.com/florianwenzel/levitate/backend/internal/db"
	"github.com/florianwenzel/levitate/backend/internal/frontend"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Logger         *slog.Logger
	Verifier       *auth.Verifier
	Pool           *pgxpool.Pool
	CORSOrigins    []string
	AllowTestReset bool

	// Public config exposed to the SPA via GET /api/public/config so a single
	// binary deployment can be configured at runtime via env vars.
	OIDCIssuerPublic string
	OIDCClientID     string
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(requestLogger(d.Logger))
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   nonEmpty(d.CORSOrigins, []string{"*"}),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Content-Disposition", "X-Pagination-Next-Cursor", "X-Pagination-Has-More"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/healthz", healthz)

	// Public config is reachable without auth so the SPA can resolve OIDC
	// before it has a token.
	r.Get("/api/public/config", publicConfigHandler(d.OIDCIssuerPublic, d.OIDCClientID))

	if d.Pool != nil && d.AllowTestReset {
		r.Post("/api/test/reset", resetHandler(d.Pool))
	}

	if d.Verifier == nil || d.Pool == nil {
		return r
	}
	q := db.New(d.Pool)
	people := newPeopleHandler(q)
	projects := newProjectsHandler(q)
	assignments := newAssignmentsHandler(q)
	timeOff := newTimeOffHandler(q)
	reports := newReportsHandler(q)
	floatImport := newFloatImportHandler(q, d.Pool)
	milestones := newMilestonesHandler(q)
	phases := newPhasesHandler(q)
	deleted := newDeletedHandler(q)
	loggedTime := newLoggedTimeHandler(q)
	statuses := newStatusesHandler(q)

	r.Route("/api", func(r chi.Router) {
		r.Use(d.Verifier.Middleware)
		r.Use(syncUser(q))
		r.Get("/me", handleMe)
		r.Route("/people", people.routes)
		r.Route("/projects", projects.routes)
		r.Route("/projects/{id}/milestones", milestones.projectRoutes)
		r.Route("/milestones", milestones.itemRoutes)
		r.Route("/projects/{id}/phases", phases.projectRoutes)
		r.Route("/phases", phases.itemRoutes)
		r.Route("/assignments", assignments.routes)
		r.Route("/time-off", timeOff.routes)
		r.Route("/logged-time", loggedTime.routes)
		r.Route("/statuses", statuses.routes)
		r.Get("/reports/utilization", reports.utilizationJSON)
		r.Get("/reports/utilization.csv", reports.utilizationCSV)
		r.Get("/reports/assignments.csv", reports.assignmentsCSV)
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireRole(auth.RoleAdmin))
			r.Post("/import/float", floatImport.importFloat)
			r.Route("/deleted", deleted.routes)
		})
	})

	// In production builds (the multi-stage Dockerfile) the embedded SPA is
	// available; in dev we don't ship the SPA in the binary and let the Nuxt
	// dev server handle the frontend on its own port.
	if frontend.HasBuild() {
		spa := frontend.Handler()
		r.NotFound(spa.ServeHTTP)
	}

	return r
}

func publicConfigHandler(oidcIssuer, oidcClientID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"oidcIssuer":   oidcIssuer,
			"oidcClientId": oidcClientID,
		})
	}
}

func handleMe(w http.ResponseWriter, r *http.Request) {
	p, ok := auth.FromContext(r.Context())
	if !ok {
		WriteProblem(w, r, http.StatusUnauthorized, "unauthorized", "missing principal")
		return
	}
	roles := make([]string, 0, len(p.Roles))
	for _, r := range p.Roles {
		roles = append(roles, string(r))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sub":   p.Subject,
		"email": p.Email,
		"name":  p.Name,
		"roles": roles,
	})
}

// syncUser upserts the authenticated principal into the users table on each
// request and ensures a corresponding people row exists so the user can show
// up on the schedule grid. Both queries are idempotent.
func syncUser(q *db.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := auth.FromContext(r.Context())
			if ok && p.Subject != "" {
				role := "member"
				if p.HasRole(auth.RoleAdmin) {
					role = "admin"
				}
				ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
				u, err := q.UpsertUser(ctx, db.UpsertUserParams{
					Sub:   p.Subject,
					Email: p.Email,
					Name:  p.Name,
					Role:  role,
				})
				if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
					slog.Default().Warn("user sync failed", "err", err, "sub", p.Subject)
				} else if err == nil {
					name := p.Name
					if name == "" {
						name = p.Email
					}
					if err := q.EnsurePersonForUser(ctx, db.EnsurePersonForUserParams{
						UserID: u.ID,
						Name:   name,
						Email:  p.Email,
					}); err != nil {
						slog.Default().Warn("person sync failed", "err", err, "user_id", uuidString(u.ID))
					}
				}
				cancel()
			}
			next.ServeHTTP(w, r)
		})
	}
}

func healthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func nonEmpty[T any](v, fallback []T) []T {
	if len(v) == 0 {
		return fallback
	}
	return v
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			logger.Info("http_request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"remote", r.RemoteAddr,
			)
		})
	}
}
