package theme

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// BuilderAssets describes the additional assets required to power the
// administrative section and element builder.
type BuilderAssets struct {
	ElementScripts []string
	SectionScripts []string
}

func discoverBuilderAssets(staticDir string) BuilderAssets {
	assets := BuilderAssets{}
	assets.ElementScripts = collectBuilderScripts(
		filepath.Join(staticDir, "js", "admin", "elements"),
		"/static/js/admin/elements",
	)
	assets.SectionScripts = collectBuilderScripts(
		filepath.Join(staticDir, "js", "admin", "sections"),
		"/static/js/admin/sections",
	)
	return assets
}

func collectBuilderScripts(dir, publicPrefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return nil
	}

	scripts := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		lower := strings.ToLower(name)
		if lower == "registry.js" {
			continue
		}

		if entry.IsDir() {
			scriptPath := filepath.Join(dir, name, "index.js")
			if _, err := os.Stat(scriptPath); err == nil {
				scripts = append(scripts, path.Join(publicPrefix, name, "index.js"))
			}
			continue
		}

		if filepath.Ext(name) != ".js" {
			continue
		}

		scripts = append(scripts, path.Join(publicPrefix, name))
	}

	sort.Strings(scripts)
	return scripts
}
