package theme

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollectSectionStyles(t *testing.T) {
	dir := t.TempDir()

	files := []string{
		"index.css",
		"page-view.css",
		"features-variation-cards.css",
		"hero-variation-split.css",
		"notes.txt",
	}

	for _, fileName := range files {
		path := filepath.Join(dir, fileName)
		if err := os.WriteFile(path, []byte("/* test */"), 0o644); err != nil {
			t.Fatalf("write %s: %v", fileName, err)
		}
	}

	styles := collectSectionStyles(dir, "/static/css/sections")
	expected := []string{
		"/static/css/sections/page-view.css",
		"/static/css/sections/features-variation-cards.css",
		"/static/css/sections/hero-variation-split.css",
	}

	if !reflect.DeepEqual(styles, expected) {
		t.Fatalf("unexpected styles order\nwant: %#v\ngot:  %#v", expected, styles)
	}
}

func TestCollectSectionStylesMissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing")
	styles := collectSectionStyles(dir, "/static/css/sections")
	if len(styles) != 0 {
		t.Fatalf("expected no styles, got %#v", styles)
	}
}

func TestCollectSectionStylesRespectsIndexImportOrder(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(
		filepath.Join(dir, "index.css"),
		[]byte(
			`@import url("./page-view.css");
@import url("./hero-variation-split.css");
@import url("./hero-variation-centered.css");
`,
		),
		0o644,
	); err != nil {
		t.Fatalf("write index.css: %v", err)
	}

	files := []string{
		"page-view.css",
		"hero-variation-centered.css",
		"hero-variation-split.css",
		"features-variation-cards.css",
	}

	for _, fileName := range files {
		path := filepath.Join(dir, fileName)
		if err := os.WriteFile(path, []byte("/* test */"), 0o644); err != nil {
			t.Fatalf("write %s: %v", fileName, err)
		}
	}

	styles := collectSectionStyles(dir, "/static/css/sections")
	expected := []string{
		"/static/css/sections/page-view.css",
		"/static/css/sections/hero-variation-split.css",
		"/static/css/sections/hero-variation-centered.css",
		"/static/css/sections/features-variation-cards.css",
	}

	if !reflect.DeepEqual(styles, expected) {
		t.Fatalf("unexpected styles order\nwant: %#v\ngot:  %#v", expected, styles)
	}
}
