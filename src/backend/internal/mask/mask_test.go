package mask_test

import (
	"encoding/json"
	"testing"

	"github.com/fredrik/quintus/internal/mask"
)

func TestActiveMasks_UserInGroup(t *testing.T) {
	rules := []mask.Rule{
		{Column: "personnummer", VisibleToGroups: []string{"pii-approved"}, Mask: "redacted"},
	}
	// User in pii-approved should see the column — no active masks.
	active := mask.ActiveMasks(rules, []string{"pii-approved"})
	if len(active) != 0 {
		t.Fatalf("expected 0 active masks, got %d", len(active))
	}
}

func TestActiveMasks_UserNotInGroup(t *testing.T) {
	rules := []mask.Rule{
		{Column: "personnummer", VisibleToGroups: []string{"pii-approved"}, Mask: "redacted"},
	}
	// User not in pii-approved should have the mask applied.
	active := mask.ActiveMasks(rules, []string{"qe-viewers"})
	if len(active) != 1 {
		t.Fatalf("expected 1 active mask, got %d", len(active))
	}
	if active[0].Column != "personnummer" {
		t.Fatalf("expected personnummer, got %s", active[0].Column)
	}
}

func TestApply_Redacted(t *testing.T) {
	r := mask.Rule{Mask: "redacted"}
	got := mask.Apply(r, "19850312-4821")
	if got != "***REDACTED***" {
		t.Fatalf("got %v", got)
	}
}

func TestApply_Null(t *testing.T) {
	r := mask.Rule{Mask: "null"}
	got := mask.Apply(r, "anything")
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestApply_Hash(t *testing.T) {
	r := mask.Rule{Mask: "hash"}
	got := mask.Apply(r, "19850312-4821")
	s, ok := got.(string)
	if !ok || len(s) != 64 {
		t.Fatalf("expected 64-char hex, got %v", got)
	}
	// Hash should be stable.
	got2 := mask.Apply(r, "19850312-4821")
	if got != got2 {
		t.Fatal("hash not stable")
	}
}

func TestApply_Partial_Email(t *testing.T) {
	r := mask.Rule{Mask: "partial"}
	got := mask.Apply(r, "erik.johansson@example.com")
	s := got.(string)
	if s[0] != 'e' {
		t.Fatalf("expected first char 'e', got %v", s)
	}
	if s != "e***@example.com" {
		t.Fatalf("got %v", s)
	}
}

func TestApply_Partial_String(t *testing.T) {
	r := mask.Rule{Mask: "partial"}
	got := mask.Apply(r, "Johansson")
	if got != "J***n" {
		t.Fatalf("got %v", got)
	}
}

func TestApply_NilValue(t *testing.T) {
	r := mask.Rule{Mask: "redacted"}
	got := mask.Apply(r, nil)
	if got != nil {
		t.Fatalf("expected nil for nil input, got %v", got)
	}
}

func TestParseRules(t *testing.T) {
	raw := json.RawMessage(`[
		{"column":"personnummer","visible_to_groups":["pii-approved"],"mask":"redacted"},
		{"column":"email","visible_to_groups":["pii-approved","pii-support-partial"],"mask":"partial"}
	]`)
	rules, err := mask.ParseRules(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
}

func TestThreeUsers_DifferentOutcomes(t *testing.T) {
	rules := []mask.Rule{
		{Column: "personnummer", VisibleToGroups: []string{"pii-approved"}, Mask: "redacted"},
		{Column: "email", VisibleToGroups: []string{"pii-approved", "pii-support-partial"}, Mask: "partial"},
	}

	adminPII := mask.ActiveMasks(rules, []string{"qe-admins", "pii-approved"})
	editorPII := mask.ActiveMasks(rules, []string{"qe-editors", "pii-approved"})
	viewerNoPII := mask.ActiveMasks(rules, []string{"qe-viewers"})

	if len(adminPII) != 0 {
		t.Fatalf("admin+PII should see everything, got %d masks", len(adminPII))
	}
	if len(editorPII) != 0 {
		t.Fatalf("editor+PII should see everything, got %d masks", len(editorPII))
	}
	if len(viewerNoPII) != 2 {
		t.Fatalf("viewer without PII should have 2 masks, got %d", len(viewerNoPII))
	}
}