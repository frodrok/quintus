package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/fredrik/quintus/internal/db"
	"github.com/fredrik/quintus/internal/identity"
)

type queryResponse struct {
	ID           uuid.UUID       `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	ConnectionID uuid.UUID       `json:"connection_id"`
	SQL          string          `json:"sql"`
	Parameters   json.RawMessage `json:"parameters"`
	ColumnMasks  json.RawMessage `json:"column_masks"`
	RowMask      json.RawMessage `json:"row_mask"`
	OwnerSub     string          `json:"owner_sub"`
	OwnerEmail   string          `json:"owner_email"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

func toQueryResponse(q *db.Query) queryResponse {
	return queryResponse{
		ID:           q.ID,
		Name:         q.Name,
		Description:  q.Description,
		ConnectionID: q.ConnectionID,
		SQL:          q.SQL,
		Parameters:   q.Parameters,
		ColumnMasks:  q.ColumnMasks,
		RowMask: q.RowMask,
		OwnerSub:     q.OwnerSub,
		OwnerEmail:   q.OwnerEmail,
		CreatedAt:    q.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    q.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

type queryRequest struct {
	Name         string          `json:"name"`
	Description  *string         `json:"description"`
	ConnectionID uuid.UUID       `json:"connection_id"`
	SQL          string          `json:"sql"`
	Parameters   json.RawMessage `json:"parameters"`
	ColumnMasks  json.RawMessage `json:"column_masks"`
	RowMask      json.RawMessage `json:"row_mask"`
}

func (d *Deps) ListQueries(w http.ResponseWriter, r *http.Request) {
	queries, err := d.DB.ListQueries(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	resp := make([]queryResponse, len(queries))
	for i, q := range queries {
		resp[i] = toQueryResponse(q)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (d *Deps) GetQuery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	q, err := d.DB.GetQuery(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if q == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, toQueryResponse(q))
}

func (d *Deps) CreateQuery(w http.ResponseWriter, r *http.Request) {
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Name == "" || req.SQL == "" || req.ConnectionID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, sql, and connection_id are required"})
		return
	}
	if req.Parameters == nil {
		req.Parameters = json.RawMessage("[]")
	}
	if req.ColumnMasks == nil {
		req.ColumnMasks = json.RawMessage("[]")
	}
	id := identity.FromContext(r.Context())
	q, err := d.DB.CreateQuery(r.Context(), db.CreateQueryParams{
		Name:         req.Name,
		Description:  req.Description,
		ConnectionID: req.ConnectionID,
		SQL:          req.SQL,
		Parameters:   req.Parameters,
		ColumnMasks:  req.ColumnMasks,
		RowMask: req.RowMask,
		OwnerSub:     id.Sub,
		OwnerEmail:   id.Email,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, toQueryResponse(q))
}

func (d *Deps) UpdateQuery(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if req.Name == "" || req.SQL == "" || req.ConnectionID == uuid.Nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, sql, and connection_id are required"})
		return
	}
	if req.Parameters == nil {
		req.Parameters = json.RawMessage("[]")
	}
	if req.ColumnMasks == nil {
		req.ColumnMasks = json.RawMessage("[]")
	}
	q, err := d.DB.UpdateQuery(r.Context(), id, db.CreateQueryParams{
		Name:         req.Name,
		Description:  req.Description,
		ConnectionID: req.ConnectionID,
		SQL:          req.SQL,
		Parameters:   req.Parameters,
		ColumnMasks:  req.ColumnMasks,
		RowMask: req.RowMask,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if q == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		return
	}
	writeJSON(w, http.StatusOK, toQueryResponse(q))
}