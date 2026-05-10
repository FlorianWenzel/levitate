// Package auth provides OIDC JWT validation middleware and request-scoped principal access.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
)

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// Principal represents an authenticated request actor.
type Principal struct {
	Subject string
	Email   string
	Name    string
	Roles   []Role
}

func (p Principal) HasRole(r Role) bool {
	return slices.Contains(p.Roles, r)
}

type ctxKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, ctxKey{}, p)
}

// FromContext returns the request's principal. The boolean is false on unauthenticated requests.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(ctxKey{}).(Principal)
	return p, ok
}

// Verifier wraps a coreos/go-oidc verifier with the role-claim convention.
type Verifier struct {
	verifier  *oidc.IDTokenVerifier
	roleClaim string
}

type Options struct {
	Issuer       string // expected `iss` claim, e.g. http://localhost:8081/realms/levitate
	DiscoveryURL string // URL the backend uses to fetch discovery + JWKs (may differ from Issuer in local dev)
	Audience     string // expected `aud` claim
	RoleClaim    string // claim name carrying role strings, e.g. "roles"
}

func NewVerifier(ctx context.Context, opts Options) (*Verifier, error) {
	if opts.Issuer == "" {
		return nil, errors.New("auth: issuer is required")
	}
	discoveryURL := opts.DiscoveryURL
	if discoveryURL == "" {
		discoveryURL = opts.Issuer
	}
	roleClaim := opts.RoleClaim
	if roleClaim == "" {
		roleClaim = "roles"
	}

	// When the backend reaches Keycloak via a different URL than the public issuer
	// (docker network vs. browser-facing), we still want the verifier to enforce
	// the public issuer in token `iss` claims.
	discoveryCtx := oidc.InsecureIssuerURLContext(ctx, opts.Issuer)
	provider, err := oidc.NewProvider(discoveryCtx, discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("auth: oidc discovery: %w", err)
	}

	cfg := &oidc.Config{}
	if opts.Audience != "" {
		cfg.ClientID = opts.Audience
	} else {
		// Keycloak access tokens typically don't include the client as audience by default.
		cfg.SkipClientIDCheck = true
	}

	return &Verifier{
		verifier:  provider.Verifier(cfg),
		roleClaim: roleClaim,
	}, nil
}

// Middleware verifies a Bearer JWT and attaches the resulting Principal to the request context.
// Requests without a valid token are rejected with 401.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := bearerFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		idt, err := v.verifier.Verify(r.Context(), raw)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid token")
			return
		}
		p, err := principalFromToken(idt, v.roleClaim)
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "unauthorized", err.Error())
			return
		}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), p)))
	})
}

// RequireRole gates a handler on the principal having one of the listed roles.
func RequireRole(roles ...Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := FromContext(r.Context())
			if !ok {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing principal")
				return
			}
			for _, role := range roles {
				if p.HasRole(role) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeAuthError(w, http.StatusForbidden, "forbidden", "insufficient role")
		})
	}
}

func bearerFromHeader(h string) (string, error) {
	if h == "" {
		return "", errors.New("missing Authorization header")
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", errors.New("Authorization header must use Bearer scheme")
	}
	return strings.TrimSpace(h[len(prefix):]), nil
}

func principalFromToken(idt *oidc.IDToken, roleClaim string) (Principal, error) {
	var raw map[string]json.RawMessage
	if err := idt.Claims(&raw); err != nil {
		return Principal{}, fmt.Errorf("read claims: %w", err)
	}

	p := Principal{Subject: idt.Subject}

	if v, ok := raw["email"]; ok {
		_ = json.Unmarshal(v, &p.Email)
	}
	if p.Email == "" {
		if v, ok := raw["preferred_username"]; ok {
			_ = json.Unmarshal(v, &p.Email)
		}
	}
	if v, ok := raw["name"]; ok {
		_ = json.Unmarshal(v, &p.Name)
	}

	p.Roles = extractRoles(raw, roleClaim)
	return p, nil
}

// extractRoles supports two layouts: a top-level array of strings under roleClaim, or a
// nested object (e.g. realm_access.roles) when roleClaim is dotted.
func extractRoles(raw map[string]json.RawMessage, claim string) []Role {
	parts := strings.Split(claim, ".")
	current := raw
	for i, part := range parts {
		v, ok := current[part]
		if !ok {
			return nil
		}
		if i == len(parts)-1 {
			var arr []string
			if err := json.Unmarshal(v, &arr); err != nil {
				return nil
			}
			out := make([]Role, 0, len(arr))
			for _, s := range arr {
				out = append(out, Role(s))
			}
			return out
		}
		var next map[string]json.RawMessage
		if err := json.Unmarshal(v, &next); err != nil {
			return nil
		}
		current = next
	}
	return nil
}

func writeAuthError(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	body, _ := json.Marshal(struct {
		Type   string `json:"type"`
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail,omitempty"`
	}{"about:blank", title, status, detail})
	_, _ = w.Write(body)
}
