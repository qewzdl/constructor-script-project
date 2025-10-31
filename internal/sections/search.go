package sections

import (
	"bytes"
	"fmt"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
)

// RegisterSearch registers the default search renderer on the provided registry.
func RegisterSearch(reg *Registry) {
	if reg == nil {
		return
	}
	reg.MustRegister("search", renderSearch)
}

func renderSearch(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	if title == "" {
		title = "Search"
	}

	description := strings.TrimSpace(getString(content, "description"))
	action := strings.TrimSpace(getString(content, "action"))
	if action == "" {
		action = "/search"
	}

	placeholder := strings.TrimSpace(getString(content, "placeholder"))
	if placeholder == "" {
		placeholder = "Start typing to search"
	}

	submitLabel := strings.TrimSpace(getString(content, "submit_label"))
	if submitLabel == "" {
		submitLabel = "Search"
	}

	filterLabel := strings.TrimSpace(getString(content, "filter_label"))
	if filterLabel == "" {
		filterLabel = "Filter by"
	}

	heading := normalizeHeading(getString(content, "heading"))
	if heading == "" {
		heading = "h2"
	}

	hint := strings.TrimSpace(getString(content, "hint"))
	if hint == "" {
		hint = "Use the search form above to explore the knowledge base."
	}

	showFilters := true
	if value, ok := content["show_filters"]; ok {
		showFilters = parseBool(value, true)
	}

	searchType := normalizeSearchType(getString(content, "default_type"))
	if searchType == "" {
		searchType = "all"
	}

	id := strings.TrimSpace(getString(content, "id"))
	if id == "" {
		id = fmt.Sprintf("%s-search-%s", prefix, elem.ID)
	}

	data := map[string]interface{}{
		"ID":          id,
		"Heading":     heading,
		"Title":       title,
		"Description": description,
		"Action":      action,
		"Placeholder": placeholder,
		"SubmitLabel": submitLabel,
		"FilterLabel": filterLabel,
		"ShowFilters": showFilters,
		"ShowResults": false,
		"Query":       "",
		"SearchType":  searchType,
		"HasQuery":    false,
		"Hint":        hint,
		"Result":      nil,
	}

	tmpl, err := ctx.CloneTemplates()
	if err != nil {
		logger.Error(err, "Failed to clone templates for search section", map[string]interface{}{"element_id": elem.ID})
		return "", nil
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "components/search", data); err != nil {
		logger.Error(err, "Failed to render search section", map[string]interface{}{"element_id": elem.ID})
		return "", nil
	}

	return buf.String(), []string{"/static/js/search.js"}
}
