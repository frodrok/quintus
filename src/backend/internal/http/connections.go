package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/fredrik/quintus/internal/crypto"
	"github.com/fredrik/quintus/internal/db"
	"github.com/fredrik/quintus/internal/identity"
)

type connectionResponse struct {
	ID                 uuid.UUID `json:"id"`
	Name               string    `json:"name"`
	Driver             string    `json:"driver"`
	ReadOnly           bool      `json:"read_only"`
	StatementTimeoutMs int       `json:"statement_timeout_ms"`
	CreatedAt          string    `json:"created_at"`
	CreatedBySub       *string   `json:"created_by_sub,omitempty"`
	CreatedByEmail     *string   `json:"created_by_email,omitempty"`
}

func toConnectionResponse(c *db.Connection) connectionResponse {
	return connectionResponse{
		ID:                 c.ID,
		Name:               c.Name,
		Driver:             c.Driver,
		ReadOnly:           c.ReadOnly,
		StatementTimeoutMs: c.StatementTimeoutMs,
		CreatedAt:          c.CreatedAt.Format("2006-01-02T15:04:05Z"),
		CreatedBySub:       c.CreatedBySub,
		CreatedByEmail:     c.CreatedByEmail,
	}
}

type connectionRequest struct {
	Name               string `json:"name"`
	Driver             string `json:"driver"`
	DSN                string `json:"dsn"`
	ReadOnly           bool   `json:"read_only"`
	StatementTimeoutMs int    `json:"statement_timeout_ms"`
}

func (d *Deps) ListConnections(w http.ResponseWriter, r *http.Request) {
	conns, err := d.DB.ListConnections(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := make([]connectionResponse, len(conns))
	for i, c := range conns {
		resp[i] = toConnectionResponse(c)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (d *Deps) GetConnection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	c, err := d.DB.GetConnection(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if c == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, toConnectionResponse(c))
}

func (d *Deps) CreateConnection(w http.ResponseWriter, r *http.Request) {
	var req connectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Name == "" || req.Driver == "" || req.DSN == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, driver, and dsn are required"})
		return
	}
	if req.StatementTimeoutMs == 0 {
		req.StatementTimeoutMs = 30000
	}
	key, err := crypto.KeyFromHex(d.Cfg.DSNEncryptionKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key invalid"})
		return
	}
	encrypted, err := crypto.Encrypt(key, []byte(req.DSN))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
		return
	}
	id := identity.FromContext(r.Context())
	c, err := d.DB.CreateConnection(r.Context(), db.CreateConnectionParams{
		Name:               req.Name,
		Driver:             req.Driver,
		DSNEncrypted:       encrypted,
		ReadOnly:           req.ReadOnly,
		StatementTimeoutMs: req.StatementTimeoutMs,
		CreatedBySub:       &id.Sub,
		CreatedByEmail:     &id.Email,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, toConnectionResponse(c))
}

func (d *Deps) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req connectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Name == "" || req.Driver == "" || req.DSN == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, driver, and dsn are required"})
		return
	}
	key, err := crypto.KeyFromHex(d.Cfg.DSNEncryptionKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key invalid"})
		return
	}
	encrypted, err := crypto.Encrypt(key, []byte(req.DSN))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
		return
	}
	c, err := d.DB.UpdateConnection(r.Context(), id, db.CreateConnectionParams{
		Name:               req.Name,
		Driver:             req.Driver,
		DSNEncrypted:       encrypted,
		ReadOnly:           req.ReadOnly,
		StatementTimeoutMs: req.StatementTimeoutMs,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if c == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, toConnectionResponse(c))
}

func (d *Deps) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if err := d.DB.DeleteConnection(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *Deps) TestConnection(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	key, err := crypto.KeyFromHex(d.Cfg.DSNEncryptionKey)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "encryption key invalid"})
		return
	}

	decryptDSN := func(encrypted []byte) (string, error) {
		plain, err := crypto.Decrypt(key, encrypted)
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}

	if err := d.DB.TestConnection(r.Context(), id, decryptDSN); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

