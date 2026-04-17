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

func NewRouter(cfg *config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api", func(r chi.Router) {
		r.Use(identity.Middleware(cfg))
		r.Get("/me", handleMe(cfg))
	})

	// Serve the SPA for everything else.
	r.NotFound(spaHandler())

	return r
}

func spaHandler() http.HandlerFunc {
	dist, err := fs.Sub(spaFS, "web/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(dist))

	return func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "." {
			p = "index.html"
		}

		// If the requested asset exists, serve it directly.
		if _, err := fs.Stat(dist, p); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Otherwise hand back SPA entrypoint so client-side routing works.
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
func handleMe(cfg *config.Config) http.HandlerFunc {
	type response struct {
		Sub      string   `json:"sub"`
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Groups   []string `json:"groups"`
		Role     string   `json:"role"`
		CanPII   bool     `json:"can_pii"`
		CanAdhoc bool     `json:"can_adhoc"`
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