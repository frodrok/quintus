// Package identity contains the request-scoped user identity derived from
// Traefik-injected headers, plus the middleware that enforces proxy-secret
// trust and populates the context.
package identity

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"

	"github.com/fredrik/quintus/internal/config"
)

// Identity captures everything we know about the caller on a given request.
// It is populated by Middleware and read via FromContext.
type Identity struct {
	Sub    string
	Email  string
	Name   string
	Groups []string
	Role   string // admin | editor | viewer
}

// HasGroup reports membership in a specific group.
func (i *Identity) HasGroup(g string) bool {
	for _, x := range i.Groups {
		if x == g {
			return true
		}
	}
	return false
}

// HasAnyGroup reports membership in any of the provided groups.
func (i *Identity) HasAnyGroup(gs []string) bool {
	for _, g := range gs {
		if i.HasGroup(g) {
			return true
		}
	}
	return false
}

type ctxKey struct{}

// FromContext returns the identity injected by Middleware.
// Returns nil if the middleware didn't run — call sites inside /api should
// treat nil as a programming error.
func FromContext(ctx context.Context) *Identity {
	id, _ := ctx.Value(ctxKey{}).(*Identity)
	return id
}

// ErrNoProxySecret and ErrMissingSub are returned by the header parser to
// guide which status code to return. Not exported because the middleware
// handles the mapping itself.
var errNoProxySecret = errors.New("missing or invalid proxy secret")
var errMissingIdentity = errors.New("missing identity headers")

// Middleware verifies the proxy shared secret and extracts identity headers.
// On failure it writes 401 or 403 and stops the chain.
func Middleware(cfg *config.Config) func(http.Handler) http.Handler {
	expected := []byte(cfg.ProxySharedSecret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Proxy secret: constant-time compare.
			got := r.Header.Get("X-Auth-Proxy-Secret")
			if subtle.ConstantTimeCompare([]byte(got), expected) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			// 2. Identity headers.
			sub := r.Header.Get(cfg.HeaderSub)
			email := r.Header.Get(cfg.HeaderEmail)
			if sub == "" || email == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			groupsHeader := r.Header.Get(cfg.HeaderGroups)
			var groups []string
			if groupsHeader != "" {
				for _, g := range strings.Split(groupsHeader, cfg.GroupSeparator) {
					if t := strings.TrimSpace(g); t != "" {
						groups = append(groups, t)
					}
				}
			}

			role := cfg.ResolveRole(groups)
			if role == "" {
				http.Error(w, "forbidden: no matching role", http.StatusForbidden)
				return
			}

			id := &Identity{
				Sub:    sub,
				Email:  email,
				Name:   r.Header.Get(cfg.HeaderName),
				Groups: groups,
				Role:   role,
			}
			ctx := context.WithValue(r.Context(), ctxKey{}, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole wraps a handler and enforces minimum role. Ordering: admin > editor > viewer.
func RequireRole(minRole string) func(http.Handler) http.Handler {
	rank := map[string]int{"viewer": 1, "editor": 2, "admin": 3}
	min := rank[minRole]
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := FromContext(r.Context())
			if id == nil || rank[id.Role] < min {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyGroup wraps a handler and enforces membership in at least one
// of the given groups. Used for PII and ad-hoc gates.
func RequireAnyGroup(groups []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := FromContext(r.Context())
			if id == nil || !id.HasAnyGroup(groups) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
