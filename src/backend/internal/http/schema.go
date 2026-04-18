package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"log"
	"fmt"
)

type SchemaTable struct {
	Schema  string   `json:"schema"`
	Table   string   `json:"table"`
	Columns []Column `json:"columns"`
}

type Column struct {
	Name     string `json:"name"`
	DataType string `json:"data_type"`
	Nullable bool   `json:"nullable"`
}

func (d *Deps) GetConnectionSchema(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))

	log.Println(fmt.Sprintf("getting schema for %s", id))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}

	pool, err := d.DB.TargetPool(r.Context(), id, d.decryptDSN())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}

	rows, err := pool.QueryContext(r.Context(), `
		SELECT
			c.table_schema,
			c.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' as nullable
		FROM information_schema.columns c
		JOIN information_schema.tables t
			ON t.table_schema = c.table_schema
			AND t.table_name = c.table_name
		WHERE c.table_schema NOT IN ('pg_catalog','information_schema','pg_toast')
			AND t.table_type = 'BASE TABLE'
		ORDER BY c.table_schema, c.table_name, c.ordinal_position
	`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	// Build schema → table → columns tree.
	type key struct{ schema, table string }
	index := make(map[key]*SchemaTable)
	var order []key

	for rows.Next() {
		var schema, table, col, dataType string
		var nullable bool
		if err := rows.Scan(&schema, &table, &col, &dataType, &nullable); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		k := key{schema, table}
		if _, ok := index[k]; !ok {
			index[k] = &SchemaTable{Schema: schema, Table: table}
			order = append(order, k)
		}
		index[k].Columns = append(index[k].Columns, Column{
			Name:     col,
			DataType: dataType,
			Nullable: nullable,
		})
	}

	// Bygg tabell-resultat
	result := make([]*SchemaTable, 0, len(order))
	for _, k := range order {
		result = append(result, index[k])
	}

	// Funktioner
	funcRows, err := pool.QueryContext(r.Context(), `
    SELECT
        r.routine_schema,
        r.routine_name,
        r.data_type as return_type,
        p.parameter_name,
        p.data_type as param_type
    FROM information_schema.routines r
    LEFT JOIN information_schema.parameters p
        ON p.specific_schema = r.specific_schema
        AND p.specific_name = r.specific_name
        AND p.parameter_mode = 'IN'
    WHERE r.routine_schema NOT IN ('pg_catalog','information_schema','pg_toast')
        AND r.routine_type = 'FUNCTION'
    ORDER BY r.routine_schema, r.routine_name, p.ordinal_position
`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer funcRows.Close()

	type funcKey struct{ schema, name string }
	funcIndex := make(map[funcKey]*SchemaTable)
	var funcOrder []funcKey

	for funcRows.Next() {
		var schema, name, returnType string
		var paramName, dataType *string
		if err := funcRows.Scan(&schema, &name, &returnType, &paramName, &dataType); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		k := funcKey{schema, name}
		if _, ok := funcIndex[k]; !ok {
			funcIndex[k] = &SchemaTable{
				Schema: schema,
				Table:  "ƒ " + name,
				Columns: []Column{
					{Name: "returns", DataType: returnType},
				},
			}
			funcOrder = append(funcOrder, k)
		}
		if paramName != nil && dataType != nil {
			funcIndex[k].Columns = append(funcIndex[k].Columns, Column{
				Name:     *paramName,
				DataType: *dataType,
			})
		}
	}

	for _, k := range funcOrder {
		result = append(result, funcIndex[k])
	}

	writeJSON(w, http.StatusOK, result)
}