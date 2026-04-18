// Package param handles named parameter substitution for saved queries.
package param

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Definition declares a single parameter on a saved query.
type Definition struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"` // string | int | float | date | bool | enum
	Required bool     `json:"required"`
	Default  any      `json:"default,omitempty"`
	Values   []string `json:"values,omitempty"` // for enum
}

// ParseDefinitions parses the parameters JSONB from a saved query.
func ParseDefinitions(raw json.RawMessage) ([]Definition, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "[]" {
		return nil, nil
	}
	var defs []Definition
	if err := json.Unmarshal(raw, &defs); err != nil {
		return nil, fmt.Errorf("parse parameter definitions: %w", err)
	}
	return defs, nil
}

// ParseValues parses the parameter values from a run request.
// Values are a JSON object: {"name": value, ...}
func ParseValues(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var vals map[string]any
	if err := json.Unmarshal(raw, &vals); err != nil {
		return nil, fmt.Errorf("parse parameter values: %w", err)
	}
	return vals, nil
}

// Bind validates values against definitions and rewrites :name placeholders
// to positional $N placeholders for pgx. Returns the rewritten SQL and args.
func Bind(sql string, defs []Definition, values map[string]any) (string, []any, error) {
	if len(defs) == 0 {
		return sql, nil, nil
	}

	// Merge defaults into values.
	merged := make(map[string]any)
	for _, d := range defs {
    name := strings.TrimPrefix(d.Name, ":")
    if d.Default != nil {
        merged[name] = d.Default
    }
}
	for k, v := range values {
		merged[k] = v
	}

	// Validate required params.
for _, d := range defs {
    name := strings.TrimPrefix(d.Name, ":")
    if d.Required {
        if _, ok := merged[name]; !ok {
            return "", nil, fmt.Errorf("required parameter %q is missing", name)
        }
    }
}

	// Find all :name occurrences in order of first appearance.
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	seen := make(map[string]int) // name -> positional index (1-based)
	var args []any
	counter := 1

	rewritten := re.ReplaceAllStringFunc(sql, func(match string) string {
		name := strings.TrimPrefix(match, ":")
		if idx, ok := seen[name]; ok {
			return fmt.Sprintf("$%d", idx)
		}
		val, ok := merged[name]
		if !ok {
			// Unknown param — leave as-is (will fail at DB level).
			return match
		}
		seen[name] = counter
		args = append(args, val)
		result := fmt.Sprintf("$%d", counter)
		counter++
		return result
	})

	_ = matches
	return rewritten, args, nil
}