package config

import (
	"os"
	"testing"
)

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	original, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset %s: %v", key, err)
	}
	t.Cleanup(func() {
		if !existed {
			_ = os.Unsetenv(key)
			return
		}
		_ = os.Setenv(key, original)
	})
}

func TestSubtitleGenerationAutoEnablesWithAPIKey(t *testing.T) {
	unsetEnv(t, "SUBTITLE_GENERATION_ENABLED")
	t.Setenv("OPENAI_API_KEY", "test-openai-key")
	t.Setenv("SUBTITLE_PROVIDER", "openai")

	cfg := New()
	if !cfg.SubtitleGenerationEnabled {
		t.Fatalf("expected subtitle generation to auto-enable when API key is provided")
	}
}

func TestSubtitleGenerationRespectsExplicitDisable(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-openai-key")
	t.Setenv("SUBTITLE_GENERATION_ENABLED", "false")

	cfg := New()
	if cfg.SubtitleGenerationEnabled {
		t.Fatalf("expected subtitle generation to remain disabled when flag explicitly set")
	}
}

func TestSubtitleGenerationRemainsDisabledWithoutAPIKey(t *testing.T) {
	unsetEnv(t, "SUBTITLE_GENERATION_ENABLED")
	unsetEnv(t, "OPENAI_API_KEY")

	cfg := New()
	if cfg.SubtitleGenerationEnabled {
		t.Fatalf("expected subtitle generation to remain disabled without API key")
	}
}

func TestFrameAncestorsDefaultsToNone(t *testing.T) {
	unsetEnv(t, "CSP_FRAME_ANCESTORS")

	cfg := New()
	if len(cfg.CSPFrameAncestors) != 1 || cfg.CSPFrameAncestors[0] != "'none'" {
		t.Fatalf("expected frame ancestors to default to 'none', got %#v", cfg.CSPFrameAncestors)
	}
}

func TestFrameAncestorsNormalizeValues(t *testing.T) {
	t.Setenv("CSP_FRAME_ANCESTORS", "self, https://example.com , 'none'")

	cfg := New()
	if len(cfg.CSPFrameAncestors) != 2 {
		t.Fatalf("expected two normalized frame ancestor entries, got %#v", cfg.CSPFrameAncestors)
	}

	expected := []string{"'self'", "https://example.com"}
	for i, value := range expected {
		if cfg.CSPFrameAncestors[i] != value {
			t.Fatalf("expected frame ancestor %d to be %s, got %s", i, value, cfg.CSPFrameAncestors[i])
		}
	}
}
