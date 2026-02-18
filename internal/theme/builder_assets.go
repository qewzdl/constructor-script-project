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
	SectionStyles  []string
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
	assets.SectionStyles = collectSectionStyles(
		filepath.Join(staticDir, "css", "sections"),
		"/static/css/sections",
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

func collectSectionStyles(dir, publicPrefix string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return nil
	}

	stylesByName := make(map[string]string, len(entries))
	availableNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		lower := strings.ToLower(name)
		if filepath.Ext(lower) != ".css" {
			continue
		}
		if lower == "index.css" {
			continue
		}

		stylesByName[lower] = path.Join(publicPrefix, name)
		availableNames = append(availableNames, lower)
	}

	if len(stylesByName) == 0 {
		return nil
	}

	orderedNames := make([]string, 0, len(stylesByName))
	seen := make(map[string]struct{}, len(stylesByName))

	indexPath := filepath.Join(dir, "index.css")
	for _, name := range parseSectionIndexImports(indexPath) {
		if _, ok := stylesByName[name]; !ok {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		orderedNames = append(orderedNames, name)
	}

	if _, ok := stylesByName["page-view.css"]; ok {
		if _, exists := seen["page-view.css"]; !exists {
			seen["page-view.css"] = struct{}{}
			orderedNames = append(orderedNames, "page-view.css")
		}
	}

	sort.Strings(availableNames)
	for _, name := range availableNames {
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		orderedNames = append(orderedNames, name)
	}

	styles := make([]string, 0, len(orderedNames))
	for _, name := range orderedNames {
		if stylePath, ok := stylesByName[name]; ok {
			styles = append(styles, stylePath)
		}
	}

	return styles
}

func parseSectionIndexImports(indexPath string) []string {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(trimmed), "@import") {
			continue
		}

		start := strings.Index(trimmed, "\"")
		end := strings.LastIndex(trimmed, "\"")
		if start < 0 || end <= start {
			continue
		}

		importPath := strings.TrimSpace(trimmed[start+1 : end])
		importPath = strings.TrimPrefix(importPath, "./")
		importPath = strings.TrimPrefix(importPath, "/")
		if importPath == "" {
			continue
		}

		baseName := strings.ToLower(filepath.Base(importPath))
		if filepath.Ext(baseName) != ".css" {
			continue
		}
		if baseName == "index.css" {
			continue
		}

		result = append(result, baseName)
	}

	return result
}
