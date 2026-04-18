package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr          string
	DBDSN             string
	ProxySharedSecret string
	HeaderSub         string
	HeaderEmail       string
	HeaderName        string
	HeaderGroups      string
	GroupSeparator    string
	RoleMapping       map[string]string
	PIIGroups         []string
	AdhocGroups       []string
	LogoutURL         string
	MaxUIRows         int
	DSNEncryptionKey string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPAddr:          getenvDefault("QE_HTTP_ADDR", ":8080"),
		DBDSN:             os.Getenv("QE_DB_DSN"),
		ProxySharedSecret: os.Getenv("QE_PROXY_SHARED_SECRET"),
		HeaderSub:         getenvDefault("QE_HEADER_SUB", "X-User-Sub"),
		HeaderEmail:       getenvDefault("QE_HEADER_EMAIL", "X-User-Email"),
		HeaderName:        getenvDefault("QE_HEADER_NAME", "X-User-Name"),
		HeaderGroups:      getenvDefault("QE_HEADER_GROUPS", "X-User-Groups"),
		GroupSeparator:    getenvDefault("QE_GROUP_SEPARATOR", ","),
		RoleMapping:       parseRoleMapping(os.Getenv("QE_ROLE_MAPPING")),
		PIIGroups:         splitCSV(os.Getenv("QE_PII_GROUPS")),
		AdhocGroups:       splitCSV(os.Getenv("QE_ADHOC_GROUPS")),
		LogoutURL:         os.Getenv("QE_LOGOUT_URL"),
		MaxUIRows:         10000,
		DSNEncryptionKey: os.Getenv("QE_DSN_ENCRYPTION_KEY"),
	}

	if cfg.ProxySharedSecret == "" {
		return nil, fmt.Errorf("QE_PROXY_SHARED_SECRET is required")
	}
	if len(cfg.RoleMapping) == 0 {
		return nil, fmt.Errorf("QE_ROLE_MAPPING is required")
	}

	if cfg.DSNEncryptionKey == "" {
    return nil, fmt.Errorf("QE_DSN_ENCRYPTION_KEY is required")
}

	return cfg, nil
}

func parseRoleMapping(s string) map[string]string {
	out := make(map[string]string)
	for _, pair := range splitCSV(s) {
		parts := strings.SplitN(pair, ":", 2)
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
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) ResolveRole(groups []string) string {
	rank := map[string]int{
		"viewer": 1,
		"editor": 2,
		"admin":  3,
	}

	best := ""
	bestRank := 0

	for _, g := range groups {
		role, ok := c.RoleMapping[g]
		if !ok {
			continue
		}
		if rank[role] > bestRank {
			best = role
			bestRank = rank[role]
		}
	}

	return best
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}