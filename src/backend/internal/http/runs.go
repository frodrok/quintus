package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/fredrik/quintus/internal/executor"
	"github.com/fredrik/quintus/internal/identity"
	"github.com/fredrik/quintus/internal/param"
)

type createRunRequest struct {
    ConnectionID  uuid.UUID              `json:"connection_id"`
    QueryID       *uuid.UUID             `json:"query_id"`
    SQL           string                 `json:"sql"`
    Parameters    map[string]any         `json:"parameters"`
    ParameterDefs []param.Definition     `json:"parameter_defs"`
    ExportFormat  *string                `json:"export_format"`
}
func (d *Deps) decryptDSN() func([]byte) (string, error) {
	return func(encrypted []byte) (string, error) {
		key, err := d.cryptoKey()
		if err != nil {
			return "", err
		}
		plain, err := d.cryptoDecrypt(key, encrypted)
		if err != nil {
			return "", err
		}
		return string(plain), nil
	}
}

func (d *Deps) CreateRun(w http.ResponseWriter, r *http.Request) {
	var req createRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.ConnectionID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "connection_id is required"})
		return
	}
	if req.SQL == "" && req.QueryID == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "sql or query_id is required"})
		return
	}

	// If query_id provided, load the SQL from DB.
	sql := req.SQL
	// If query_id provided, load the SQL and column masks from DB.
var paramDefs []param.Definition
var columnMasks json.RawMessage
var rowMask json.RawMessage

if req.QueryID == nil {
    id := identity.FromContext(r.Context())
    if !id.HasAnyGroup(d.Cfg.AdhocGroups) {
        writeJSON(w, http.StatusForbidden, map[string]string{"error": "ad-hoc queries not allowed"})
        return
    }
}

if req.QueryID != nil {

	
    q, err := d.DB.GetQuery(r.Context(), *req.QueryID)
    if err != nil || q == nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query not found"})
        return
    }
    if sql == "" {
        sql = q.SQL
    }
    columnMasks = q.ColumnMasks
    rowMask = q.RowMask
    defs, err := param.ParseDefinitions(q.Parameters)
    if err != nil {
        writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
        return
    }
    paramDefs = defs
} else if len(req.ParameterDefs) > 0 {
	paramDefs = req.ParameterDefs
}

// Serialise raw params for audit log.
rawParams, _ := json.Marshal(req.Parameters)

id := identity.FromContext(r.Context())

result, err := d.Executor.Run(r.Context(), executor.Request{
    ConnectionID: req.ConnectionID,
    QueryID:      req.QueryID,
    SQL:          sql,
    ParamDefs:    paramDefs,
    ParamValues:  req.Parameters,
    RawParams:    rawParams,
    ExportFormat: req.ExportFormat,
    ColumnMasks:  columnMasks,
    RowMask:      rowMask,
    MaxRows:      d.Cfg.MaxUIRows,
    Identity:     id,
    ClientIP:     executor.ClientIP(r),
    UserAgent:    executor.StrPtr(r.UserAgent()),
    DecryptDSN:   d.decryptDSN(),
})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (d *Deps) GetRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	run, err := d.DB.GetRun(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if run == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (d *Deps) StreamRun(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, map[string]string{"error": "not implemented"})
}