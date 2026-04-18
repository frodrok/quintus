// Package http wires up the chi router and handler packages.
package http

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/fredrik/quintus/internal/config"
	"github.com/fredrik/quintus/internal/identity"
)

//go:embed web/dist/*
var spaFS embed.FS

func NewRouter(cfg *config.Config, deps *Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", deps.Healthz)
	r.Get("/readyz", deps.Readyz)

	r.Route("/api", func(r chi.Router) {
	r.Use(identity.Middleware(cfg))

	r.Get("/me", deps.HandleMe)
	r.Get("/logout", deps.HandleLogout)

	r.Get("/queries", deps.ListQueries)
	r.With(identity.RequireRole("editor")).Post("/queries", deps.CreateQuery)
	r.Get("/queries/{id}", deps.GetQuery)
	r.With(identity.RequireRole("editor")).Put("/queries/{id}", deps.UpdateQuery)

	r.Post("/runs", deps.CreateRun)
	r.Get("/runs/{id}", deps.GetRun)
	r.Get("/runs/{id}/stream", deps.StreamRun)

	r.Get("/connections/{id}/schema", deps.GetConnectionSchema)

	r.With(identity.RequireRole("admin")).Get("/audit/runs", deps.ListAuditRuns)

	r.With(identity.RequireRole("admin")).Route("/connections", func(r chi.Router) {
    r.Get("/", deps.ListConnections)
    r.Post("/", deps.CreateConnection)
    r.Get("/{id}", deps.GetConnection)
    r.Put("/{id}", deps.UpdateConnection)
    r.Delete("/{id}", deps.DeleteConnection)
    r.Post("/{id}/test", deps.TestConnection)
		
})
})

	r.NotFound(spaHandler())
	return r
}

func handleLogout(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"url": cfg.LogoutURL,
		})
	}
}

func spaHandler() http.HandlerFunc {
	dist, err := fs.Sub(spaFS, "web/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(dist))

	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "." || p == "" {
			p = "index.html"
		}

		if _, err := fs.Stat(dist, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Do not serve index.html for missing asset files.
		if strings.HasPrefix(p, "assets/") || strings.Contains(path.Base(p), ".") {
			http.NotFound(w, r)
			return
		}

		index, err := fs.ReadFile(dist, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(index)
	}
}

// handleMe returns the current identity, role, and capability flags.
func (d *Deps) HandleMe(w http.ResponseWriter, r *http.Request) {
	id := identity.FromContext(r.Context())

	resp := struct {
		Sub      string   `json:"sub"`
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Groups   []string `json:"groups"`
		Role     string   `json:"role"`
		CanPII   bool     `json:"can_pii"`
		CanAdhoc bool     `json:"can_adhoc"`
	}{
		Sub:      id.Sub,
		Email:    id.Email,
		Name:     id.Name,
		Groups:   id.Groups,
		Role:     id.Role,
		CanPII:   id.HasAnyGroup(d.Cfg.PIIGroups),
		CanAdhoc: id.HasAnyGroup(d.Cfg.AdhocGroups),
	}

	writeJSON(w, http.StatusOK, resp)
}

func (d *Deps) HandleLogout(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"url": d.Cfg.LogoutURL,
	})
}