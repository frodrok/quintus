// Package config loads QueryExplorer configuration from environment variables.
//
// All variables are QE_-prefixed. See the spec section 3 and 6 for the
// authoritative list.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config is the full app configuration. Parsing happens once at startup;
// handlers receive this by value or pointer but never re-read the environment.
type Config struct {
	HTTPAddr string

	DBDSN string

	// Auth / identity
	ProxySharedSecret string
	HeaderSub         string
	HeaderEmail       string
	HeaderName        string
	HeaderGroups      string
	GroupSeparator    string

	// Group → role mapping. A user gets the *highest* role any of their groups
	// maps to. Unmapped users are rejected with 403.
	RoleMapping map[string]string // group -> role

	// Capability groups (orthogonal to role).
	PIIGroups    []string
	AdhocGroups  []string

	LogoutURL string

	// DSN encryption key (32 bytes, base64 or hex). Used for AES-256-GCM on
	// the stored connection DSN bytes.
	DSNEncryptionKey []byte
}

// Load reads environment variables and validates them.
func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:          getEnv("QE_HTTP_ADDR", ":8080"),
		DBDSN:             os.Getenv("QE_DB_DSN"),
		ProxySharedSecret: os.Getenv("QE_PROXY_SHARED_SECRET"),
		HeaderSub:         getEnv("QE_HEADER_SUB", "X-User-Sub"),
		HeaderEmail:       getEnv("QE_HEADER_EMAIL", "X-User-Email"),
		HeaderName:        getEnv("QE_HEADER_NAME", "X-User-Name"),
		HeaderGroups:      getEnv("QE_HEADER_GROUPS", "X-User-Groups"),
		GroupSeparator:    getEnv("QE_GROUP_SEPARATOR", ","),
		LogoutURL:         os.Getenv("QE_LOGOUT_URL"),
	}

	cfg.RoleMapping = parseRoleMapping(os.Getenv("QE_ROLE_MAPPING"))
	cfg.PIIGroups = splitCSV(os.Getenv("QE_PII_GROUPS"))
	cfg.AdhocGroups = splitCSV(os.Getenv("QE_ADHOC_GROUPS"))

	// TODO(M3.1): decode QE_DSN_ENCRYPTION_KEY (base64) and validate length=32.

	return cfg, cfg.validate()
}

func (c *Config) validate() error {
	var missing []string
	if c.DBDSN == "" {
		missing = append(missing, "QE_DB_DSN")
	}
	if c.ProxySharedSecret == "" {
		missing = append(missing, "QE_PROXY_SHARED_SECRET")
	}
	if len(c.RoleMapping) == 0 {
		missing = append(missing, "QE_ROLE_MAPPING")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required env vars: %s", strings.Join(missing, ", "))
	}
	return nil
}

// IsPIIGroup reports whether the given group is in QE_PII_GROUPS.
func (c *Config) IsPIIGroup(g string) bool {
	for _, p := range c.PIIGroups {
		if p == g {
			return true
		}
	}
	return false
}

// IsAdhocGroup reports whether the given group is in QE_ADHOC_GROUPS.
func (c *Config) IsAdhocGroup(g string) bool {
	for _, p := range c.AdhocGroups {
		if p == g {
			return true
		}
	}
	return false
}

// ResolveRole picks the highest-privilege role the user's groups map to.
// Returns "" when no group matches — caller should 403.
func (c *Config) ResolveRole(userGroups []string) string {
	const (
		admin  = "admin"
		editor = "editor"
		viewer = "viewer"
	)
	rank := map[string]int{viewer: 1, editor: 2, admin: 3}
	best := ""
	bestRank := 0
	for _, g := range userGroups {
		role, ok := c.RoleMapping[g]
		if !ok {
			continue
		}
		if r := rank[role]; r > bestRank {
			best, bestRank = role, r
		}
	}
	return best
}

// parseRoleMapping parses "admin:g1,g2,editor:g3,g4" style config.
// Actually the spec uses "admin:g1,editor:g2" — role:group pairs separated by
// commas. We flip it to group->role for O(1) lookups.
func parseRoleMapping(s string) map[string]string {
	out := make(map[string]string)
	if s == "" {
		return out
	}
	for _, pair := range strings.Split(s, ",") {
		parts := strings.SplitN(strings.TrimSpace(pair), ":", 2)
		if len(parts) != 2 {
			continue
		}
		role := strings.TrimSpace(parts[0])
		group := strings.TrimSpace(parts[1])
		if role == "" || group == "" {
			continue
		}
		out[group] = role
	}
	return out
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Sentinel error — unused for now, kept so the file compiles with the import.
var _ = errors.New
