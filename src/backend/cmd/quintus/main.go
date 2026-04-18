package main

import (
	"context"
	"log"
	"net/http"

	"github.com/fredrik/quintus/internal/config"
	"github.com/fredrik/quintus/internal/db"
	apphttp "github.com/fredrik/quintus/internal/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	pool, err := db.Open(context.Background(), cfg.DBDSN)
	if err != nil {
		log.Fatalf("db open failed: %v", err)
	}
	defer pool.Close()

	deps := apphttp.NewDeps(cfg, pool)
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