package main

import (
	"log"
	"net/http"

	"github.com/fredrik/quintus/internal/config"
	apphttp "github.com/fredrik/quintus/internal/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	deps := apphttp.NewDeps(cfg)
	handler := apphttp.NewRouter(cfg, deps)

	srv := &http.Server{
		Addr:    cfg.HTTPAddr,
		Handler: handler,
	}

	log.Printf("listening on %s", cfg.HTTPAddr)

	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}