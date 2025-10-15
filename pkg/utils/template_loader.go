package utils

import (
	"fmt"
	"html/template"
	"path/filepath"
	"sort"
)

func LoadTemplates(templatesDir string) (*template.Template, error) {
	pattern := filepath.Join(templatesDir, "*.html")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob templates: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no templates found in %s", templatesDir)
	}

	sort.Strings(files)

	ordered := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.Base(file) == "base.html" {
			ordered = append(ordered, file)
		}
	}
	for _, file := range files {
		if filepath.Base(file) != "base.html" {
			ordered = append(ordered, file)
		}
	}

	funcMap := GetTemplateFuncs()
	root := template.New(filepath.Base(ordered[0])).Funcs(funcMap)

	if _, err := root.ParseFiles(ordered...); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return root, nil
}
