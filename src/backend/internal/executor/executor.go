// Package executor runs SQL against target connections and streams results.
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/fredrik/quintus/internal/db"
	"github.com/fredrik/quintus/internal/identity"
	"github.com/fredrik/quintus/internal/mask"
	"github.com/fredrik/quintus/internal/param"
)

type Executor struct {
	db *db.DB
}

func New(database *db.DB) *Executor {
	return &Executor{db: database}
}

// Column describes a result-set column.
type Column struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// Result holds a completed preview execution.
type Result struct {
	RunID         uuid.UUID       `json:"run_id"`
	Columns       []Column        `json:"columns"`
	Rows          [][]any         `json:"rows"`
	Truncated     bool            `json:"truncated"`
	RowCount      int             `json:"row_count"`
	MaskedColumns json.RawMessage `json:"masked_columns,omitempty"`
}

// Request describes what to execute.
type Request struct {
    ConnectionID uuid.UUID
    QueryID      *uuid.UUID
    SQL          string
    ParamDefs    []param.Definition // from saved query
    ParamValues  map[string]any     // from run request
    RawParams    json.RawMessage    // stored in runs table as-is
    ExportFormat *string
    ColumnMasks  json.RawMessage
    RowMask      json.RawMessage
    MaxRows      int
    Identity     *identity.Identity
    ClientIP     *string
    UserAgent    *string
    DecryptDSN   func([]byte) (string, error)
}

func (e *Executor) Run(ctx context.Context, req Request) (*Result, error) {
	startedAt := time.Now()

	groups := ""
	if req.Identity != nil {
		for i, g := range req.Identity.Groups {
			if i > 0 {
				groups += ","
			}
			groups += g
		}
	}

	// Compute active masks before execution.
	maskRules, _ := mask.ParseRules(req.ColumnMasks)
	rowMask, _ := mask.ParseRowMask(req.RowMask)
rowMaskActive := mask.RowMaskActive(rowMask, req.Identity.Groups)
	activeMasks := mask.ActiveMasks(maskRules, req.Identity.Groups)
	maskedColumnsJSON := mask.ActiveMasksJSON(activeMasks)
	maskIndex := mask.BuildIndex(activeMasks)

	// Insert run row before touching the target DB.
	run, err := e.db.InsertRun(ctx, db.InsertRunParams{
		UserSub:       req.Identity.Sub,
		UserEmail:     req.Identity.Email,
		UserGroups:    groups,
		UserRole:      req.Identity.Role,
		ConnectionID:  req.ConnectionID,
		QueryID:       req.QueryID,
		SQL:           req.SQL,
		Parameters:    req.RawParams,
		ExportFormat:  req.ExportFormat,
		MaskedColumns: maskedColumnsJSON,
		StartedAt:     startedAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("insert run: %w", err)
	}

	result, execErr := e.execute(ctx, req, run.ID, maskIndex, rowMask, rowMaskActive)

	if execErr != nil {
    fmt.Printf("execute error: %v\n", execErr)
}

	// Update run row with outcome.
	finishedAt := time.Now()
	fp := db.FinishRunParams{
		ID:         run.ID,
		FinishedAt: finishedAt,
		DurationMs: int(finishedAt.Sub(startedAt).Milliseconds()),
		Status:     "success",
	}
	if execErr != nil {
		fp.Status = "error"
		msg := execErr.Error()
		fp.ErrorMessage = &msg
	} else {
		fp.RowCount = result.RowCount
	}
	_ = e.db.FinishRun(ctx, fp)

	if execErr != nil {
		return nil, execErr
	}
	result.RunID = run.ID
	result.MaskedColumns = maskedColumnsJSON
	return result, nil
}

func (e *Executor) execute(ctx context.Context, req Request, runID uuid.UUID, maskIndex map[string]mask.Rule, rowMask *mask.RowMask, rowMaskActive bool) (*Result, error) {
	pool, err := e.db.TargetPool(ctx, req.ConnectionID, req.DecryptDSN)
	if err != nil {
		return nil, fmt.Errorf("get target pool: %w", err)
	}
// req.Parameters on a run request contains values, not definitions.
// Definitions come from the saved query; values come from the run request.
// We need to separate these — for now bind using the query definitions
// passed via req.ParamDefs and values from req.Parameters.
rewrittenSQL, args, err := param.Bind(req.SQL, req.ParamDefs, req.ParamValues)
if err != nil {
    return nil, fmt.Errorf("bind parameters: %w", err)
}
rows, err := pool.QueryContext(ctx, rewrittenSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("column types: %w", err)
	}

	cols := make([]Column, len(colTypes))
	for i, ct := range colTypes {
		cols[i] = Column{Name: ct.Name(), Type: ct.DatabaseTypeName()}
	}

	maxRows := req.MaxRows
	if maxRows == 0 {
		maxRows = 10000
	}

	var resultRows [][]any
	truncated := false

	condColIdx := -1
if rowMask != nil {
    for i, col := range cols {
        if col.Name == rowMask.ConditionColumn {
            condColIdx = i
            break
        }
    }
    // Strip condition column from cols.
    if condColIdx >= 0 {
        cols = append(cols[:condColIdx], cols[condColIdx+1:]...)
    }
}

for rows.Next() {
    if len(resultRows) >= maxRows {
        truncated = true
        break
    }
    // Scan into original column count (including condition column).
    scanCols := colTypes
    vals := make([]any, len(scanCols))
    ptrs := make([]any, len(scanCols))
    for i := range vals {
        ptrs[i] = &vals[i]
    }
    if err := rows.Scan(ptrs...); err != nil {
        return nil, fmt.Errorf("scan row: %w", err)
    }
    // Convert []byte to string.
    for i, v := range vals {
        if b, ok := v.([]byte); ok {
            vals[i] = string(b)
        }
    }
    // Apply row mask before stripping condition column.
    if rowMask != nil && condColIdx >= 0 {
        condition := false
        if condColIdx < len(vals) {
            switch v := vals[condColIdx].(type) {
            case bool:
                condition = v
            case int64:
                condition = v != 0
            }
        }
        if rowMaskActive && condition {
            for i := range vals {
                if i != condColIdx {
                    vals[i] = "***REDACTED***"
                }
            }
        }
        // Strip condition column from row.
        vals = append(vals[:condColIdx], vals[condColIdx+1:]...)
    }
    // Apply column masks using cols (already stripped).
    for i, col := range cols {
        if rule, ok := maskIndex[col.Name]; ok {
            vals[i] = mask.Apply(rule, vals[i])
        }
    }
    resultRows = append(resultRows, vals)
}
	if err := rows.Err(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("cancelled")
		}
		return nil, fmt.Errorf("rows: %w", err)
	}

	return &Result{
		Columns:   cols,
		Rows:      resultRows,
		Truncated: truncated,
		RowCount:  len(resultRows),
	}, nil
}

// ClientIP extracts the real IP from the request, preferring X-Real-IP.
func ClientIP(r *http.Request) *string {
	ip := r.Header.Get("X-Real-Ip")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if ip == "" {
		return nil
	}
	return &ip
}

// StrPtr is a convenience helper.
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// UUIDPtr wraps a UUID pointer check.
func UUIDPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}