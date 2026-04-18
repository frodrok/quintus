package http

import (
	"net/http"

	"github.com/fredrik/quintus/internal/config"
	"github.com/fredrik/quintus/internal/db"
	"github.com/fredrik/quintus/internal/executor"
	"github.com/fredrik/quintus/internal/crypto"
)

type Deps struct {
	Cfg *config.Config
	DB  *db.DB
	Executor *executor.Executor
}

func NewDeps(cfg *config.Config, pool *db.DB) *Deps {
	return &Deps{Cfg: cfg, 
		DB: pool,
	Executor: executor.New(pool)}
}

// stub handlers for now
func (d *Deps) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (d *Deps) Readyz(w http.ResponseWriter, r *http.Request) {
	if err := d.DB.Pool().PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (d *Deps) ListAuditRuns(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func (d *Deps) cryptoKey() ([32]byte, error) {
    return crypto.KeyFromHex(d.Cfg.DSNEncryptionKey)
}

func (d *Deps) cryptoDecrypt(key [32]byte, encrypted []byte) ([]byte, error) {
    return crypto.Decrypt(key, encrypted)
}