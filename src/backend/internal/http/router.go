// Package http wires up the chi router and handler packages.
package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/fredrik/quintus/internal/config"
	"github.com/fredrik/quintus/internal/identity"
)

// NewRouter builds the root router. Handlers are stubs until the respective
// milestones fill them in — see the spec task breakdown.
func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	// Public: health checks, no auth.
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// TODO(M1.2): ping the DB pool here.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Protected API.
	r.Route("/api", func(r chi.Router) {
		r.Use(identity.Middleware(cfg))

		r.Get("/me", handleMe(cfg))

		// TODO(M3): /connections CRUD — admin only
		// r.With(identity.RequireRole("admin")).Route("/connections", ...)

		// TODO(M5): /queries CRUD — editor to write, viewer+ to read
		// r.With(identity.RequireRole("editor")).Post("/queries", ...)

		// TODO(M4): /runs — preview + export
		// TODO(M4.3b): ad-hoc endpoints wrapped in identity.RequireAnyGroup(cfg.AdhocGroups)

		// TODO(M7): /audit — admin only
	})

	// TODO(M1.3): embed.FS for the built SPA, served on everything else.

	return r
}

// handleMe returns the current identity, role, and capability flags.
// The frontend uses this to shape the UI (e.g., hide the ad-hoc tab for
// users not in QE_ADHOC_GROUPS).
func handleMe(cfg *config.Config) http.HandlerFunc {
	type response struct {
		Sub        string   `json:"sub"`
		Email      string   `json:"email"`
		Name       string   `json:"name"`
		Groups     []string `json:"groups"`
		Role       string   `json:"role"`
		CanPII     bool     `json:"can_pii"`
		CanAdhoc   bool     `json:"can_adhoc"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		id := identity.FromContext(r.Context())
		resp := response{
			Sub:      id.Sub,
			Email:    id.Email,
			Name:     id.Name,
			Groups:   id.Groups,
			Role:     id.Role,
			CanPII:   id.HasAnyGroup(cfg.PIIGroups),
			CanAdhoc: id.HasAnyGroup(cfg.AdhocGroups),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}
