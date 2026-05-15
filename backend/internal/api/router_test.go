package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestMilestoneRouteResolution probes the router with real HTTP requests to
// catch problems with overlapping `{id}` parameters in the project/milestone
// subtree.
func TestMilestoneRouteResolution(t *testing.T) {
	mux := chi.NewRouter()

	matched := ""
	stub := func(label string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			matched = label
			w.WriteHeader(http.StatusOK)
		}
	}

	mux.Route("/api", func(r chi.Router) {
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", stub("list-projects"))
			r.Get("/{id}", stub("get-project"))
		})
		r.Route("/projects/{id}/milestones", func(r chi.Router) {
			r.Get("/", stub("list-milestones"))
			r.Post("/", stub("create-milestone"))
		})
		r.Route("/milestones", func(r chi.Router) {
			r.Patch("/{id}", stub("update-milestone"))
			r.Delete("/{id}", stub("delete-milestone"))
		})
	})

	cases := []struct {
		method, path, want string
	}{
		{"GET", "/api/projects/abc/milestones", "list-milestones"},
		{"POST", "/api/projects/abc/milestones", "create-milestone"},
		{"PATCH", "/api/milestones/xyz", "update-milestone"},
		{"DELETE", "/api/milestones/xyz", "delete-milestone"},
		{"GET", "/api/projects/abc", "get-project"},
	}
	for _, tc := range cases {
		matched = ""
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s %s: got status %d, want 200; matched=%q", tc.method, tc.path, rec.Code, matched)
			continue
		}
		if matched != tc.want {
			t.Errorf("%s %s: matched %q, want %q", tc.method, tc.path, matched, tc.want)
		}
	}
}

// TestPhaseRouteResolution probes the router with real HTTP requests to catch
// problems with the overlapping `{id}` parameter on project/phase routes.
func TestPhaseRouteResolution(t *testing.T) {
	mux := chi.NewRouter()

	matched := ""
	stub := func(label string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			matched = label
			w.WriteHeader(http.StatusOK)
		}
	}

	mux.Route("/api", func(r chi.Router) {
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", stub("list-projects"))
			r.Get("/{id}", stub("get-project"))
		})
		r.Route("/projects/{id}/phases", func(r chi.Router) {
			r.Get("/", stub("list-phases-for-project"))
			r.Post("/", stub("create-phase"))
		})
		r.Route("/phases", func(r chi.Router) {
			r.Get("/", stub("list-phases"))
			r.Get("/{id}", stub("get-phase"))
			r.Patch("/{id}", stub("update-phase"))
			r.Put("/{id}", stub("put-phase"))
			r.Delete("/{id}", stub("delete-phase"))
		})
	})

	cases := []struct {
		method, path, want string
	}{
		{"GET", "/api/projects/abc/phases", "list-phases-for-project"},
		{"POST", "/api/projects/abc/phases", "create-phase"},
		{"GET", "/api/phases", "list-phases"},
		{"GET", "/api/phases/xyz", "get-phase"},
		{"PATCH", "/api/phases/xyz", "update-phase"},
		{"PUT", "/api/phases/xyz", "put-phase"},
		{"DELETE", "/api/phases/xyz", "delete-phase"},
	}
	for _, tc := range cases {
		matched = ""
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("%s %s: got status %d, want 200; matched=%q", tc.method, tc.path, rec.Code, matched)
			continue
		}
		if matched != tc.want {
			t.Errorf("%s %s: matched %q, want %q", tc.method, tc.path, matched, tc.want)
		}
	}
}

// TestRouterIncludesMilestoneRoutes walks the routing tree to confirm that the
// milestone endpoints are registered alongside the existing project subtree
// without chi rejecting the overlapping `{id}` parameter.
func TestRouterIncludesMilestoneRoutes(t *testing.T) {
	mux := chi.NewRouter()
	// Handlers are nil — we never execute, we just walk.
	m := newMilestonesHandler(nil)
	mux.Route("/api", func(r chi.Router) {
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", func(w http.ResponseWriter, _ *http.Request) {})
			r.Get("/{id}", func(w http.ResponseWriter, _ *http.Request) {})
		})
		r.Route("/projects/{id}/milestones", m.projectRoutes)
		r.Route("/milestones", m.itemRoutes)
	})

	want := map[string]bool{
		"GET /api/projects/{id}/milestones":  false,
		"POST /api/projects/{id}/milestones": false,
		"PATCH /api/milestones/{id}":         false,
		"DELETE /api/milestones/{id}":        false,
	}

	if err := chi.Walk(mux, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		key := strings.TrimSpace(method) + " " + strings.TrimSuffix(route, "/")
		if _, expected := want[key]; expected {
			want[key] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("chi.Walk: %v", err)
	}

	for key, found := range want {
		if !found {
			t.Errorf("expected route not registered: %s", key)
		}
	}
}
