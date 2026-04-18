// Package mask implements column masking functions.
package mask

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
)

// Rule declares masking for a single column.
type Rule struct {
	Column          string   `json:"column"`
	VisibleToGroups []string `json:"visible_to_groups"`
	Mask            string   `json:"mask"` // redacted | partial | null | hash
}

// ParseRules parses a JSONB column_masks value.
func ParseRules(raw json.RawMessage) ([]Rule, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rules []Rule
	if err := json.Unmarshal(raw, &rules); err != nil {
		return nil, fmt.Errorf("parse column masks: %w", err)
	}
	return rules, nil
}

// RowMask declares a row-level mask driven by a boolean condition column.
type RowMask struct {
	ConditionColumn string   `json:"condition_column"`
	VisibleToGroups []string `json:"visible_to_groups"`
}

// ParseRowMask parses a single row_mask object from JSONB.
func ParseRowMask(raw json.RawMessage) (*RowMask, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var rm RowMask
	if err := json.Unmarshal(raw, &rm); err != nil {
		return nil, fmt.Errorf("parse row mask: %w", err)
	}
	return &rm, nil
}

// RowMaskActive returns true if the row mask should be applied to this user.
func RowMaskActive(rm *RowMask, userGroups []string) bool {
	if rm == nil {
		return false
	}
	groupSet := make(map[string]bool, len(userGroups))
	for _, g := range userGroups {
		groupSet[g] = true
	}
	for _, g := range rm.VisibleToGroups {
		if groupSet[g] {
			return false
		}
	}
	return true
}

// ApplyRowMask replaces all values in a row with ***REDACTED*** except the
// condition column, which is stripped entirely. Returns the masked row and
// the index of the condition column to strip (-1 if not found).
func ApplyRowMask(cols []string, vals []any, conditionCol string, active bool) []any {
	if !active {
		return vals
	}
	out := make([]any, len(vals))
	for i, col := range cols {
		if col == conditionCol {
			out[i] = nil // will be stripped
		} else {
			out[i] = "***REDACTED***"
		}
	}
	return out
}

// StripColumn removes a column by index from cols and each row.
func StripColumn(cols []string, rows [][]any, idx int) ([]string, [][]any) {
	if idx < 0 {
		return cols, rows
	}
	newCols := make([]string, 0, len(cols)-1)
	for i, c := range cols {
		if i != idx {
			newCols = append(newCols, c)
		}
	}
	newRows := make([][]any, len(rows))
	for r, row := range rows {
		newRow := make([]any, 0, len(row)-1)
		for i, v := range row {
			if i != idx {
				newRow = append(newRow, v)
			}
		}
		newRows[r] = newRow
	}
	return newCols, newRows
}


// ActiveMasks returns the subset of rules that apply to a user with the given groups.
// A rule applies when the user is NOT in any of the visible_to_groups.
func ActiveMasks(rules []Rule, userGroups []string) []Rule {
	groupSet := make(map[string]bool, len(userGroups))
	for _, g := range userGroups {
		groupSet[g] = true
	}
	var active []Rule
	for _, r := range rules {
		visible := false
		for _, g := range r.VisibleToGroups {
			if groupSet[g] {
				visible = true
				break
			}
		}
		if !visible {
			active = append(active, r)
		}
	}
	return active
}

// BuildIndex returns a map from column name to active Rule for fast lookup.
func BuildIndex(active []Rule) map[string]Rule {
	idx := make(map[string]Rule, len(active))
	for _, r := range active {
		idx[r.Column] = r
	}
	return idx
}

// Apply applies the mask function to a value.
func Apply(rule Rule, val any) any {
	if val == nil {
		return nil
	}
	switch rule.Mask {
	case "redacted":
		return "***REDACTED***"
	case "null":
		return nil
	case "hash":
		s := fmt.Sprintf("%v", val)
		sum := sha256.Sum256([]byte(s))
		return fmt.Sprintf("%x", sum)
	case "partial":
		return partial(val)
	default:
		return "***REDACTED***"
	}
}

func partial(val any) any {
	s := fmt.Sprintf("%v", val)
	// Email: f***@example.com
	if idx := strings.Index(s, "@"); idx > 0 {
		return s[:1] + "***" + s[idx:]
	}
	// Number-like: ****
	allDigits := true
	for _, c := range s {
		if c < '0' || c > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return "****"
	}
	// String: first + *** + last char
	runes := []rune(s)
	if len(runes) <= 2 {
		return "***"
	}
	return string(runes[0]) + "***" + string(runes[len(runes)-1])
}

// ActiveMasksJSON serialises the active mask rules for storage in the runs table.
func ActiveMasksJSON(active []Rule) json.RawMessage {
	if len(active) == 0 {
		b, _ := json.Marshal([]Rule{})
		return b
	}
	b, _ := json.Marshal(active)
	return b
}