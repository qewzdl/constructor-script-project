package middleware

import (
	"strings"
	"testing"
)

func TestBuildContentSecurityPolicyAddsMediaSrc(t *testing.T) {
	policy := buildContentSecurityPolicy(nil, nil)
	directives := parseContentSecurityPolicy(policy)

	mediaSrc, ok := directives["media-src"]
	if !ok {
		t.Fatalf("expected media-src directive to be present in policy: %s", policy)
	}

	for _, required := range []string{"'self'", "data:", "blob:"} {
		if _, allowed := mediaSrc[required]; !allowed {
			t.Fatalf("expected media-src to allow %s, policy: %s", required, policy)
		}
	}
}

func parseContentSecurityPolicy(policy string) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})

	for _, directive := range strings.Split(policy, ";") {
		directive = strings.TrimSpace(directive)
		if directive == "" {
			continue
		}

		parts := strings.Fields(directive)
		if len(parts) == 0 {
			continue
		}

		name := parts[0]
		values := make(map[string]struct{}, len(parts)-1)
		for _, value := range parts[1:] {
			values[value] = struct{}{}
		}

		result[name] = values
	}

	return result
}
