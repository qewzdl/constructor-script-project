package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// SectionTemplate provides a base for creating common section patterns.
type SectionTemplate struct {
	Type            string
	WrapperClass    string
	ItemClass       string
	EmptyMessage    string
	RequiresService string // e.g., "post", "category", "course"
}

// TemplateRenderer creates a renderer from a template configuration.
func (st *SectionTemplate) TemplateRenderer(
	fetchData func(ctx RenderContext, section models.Section) (interface{}, error),
	renderItems func(ctx RenderContext, data interface{}, prefix string) (string, []string, error),
) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		section, ok := extractSection(elem)
		if !ok {
			return "", nil
		}

		// Fetch data
		data, err := fetchData(ctx, section)
		if err != nil || data == nil {
			emptyClass := fmt.Sprintf("%s__%s-empty", prefix, strings.ReplaceAll(st.Type, "_", "-"))
			msg := st.EmptyMessage
			if msg == "" {
				msg = "No content available."
			}
			return fmt.Sprintf(`<p class="%s">%s</p>`, emptyClass, template.HTMLEscapeString(msg)), nil
		}

		// Render items
		html, scripts, err := renderItems(ctx, data, prefix)
		if err != nil {
			emptyClass := fmt.Sprintf("%s__%s-empty", prefix, strings.ReplaceAll(st.Type, "_", "-"))
			return fmt.Sprintf(`<p class="%s">Unable to display content.</p>`, emptyClass), nil
		}

		// Wrap in container if needed
		if st.WrapperClass != "" {
			wrapperClass := fmt.Sprintf("%s__%s", prefix, st.WrapperClass)
			html = fmt.Sprintf(`<div class="%s">%s</div>`, wrapperClass, html)
		}

		return html, scripts
	}
}

// SimpleContentRenderer creates a renderer for simple content types (paragraph, image, etc).
func SimpleContentRenderer(
	contentKey string,
	renderFunc func(content string, prefix string) string,
) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		content := sectionContent(elem)
		value, ok := content[contentKey].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return "", nil
		}

		html := renderFunc(value, prefix)
		return html, nil
	}
}

// ListRenderer creates a renderer for list-based sections.
func ListRenderer(
	itemsKey string,
	ordered bool,
	renderItem func(item interface{}, index int, prefix string) string,
) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		content := sectionContent(elem)

		// Get items
		var items []interface{}
		if rawItems, ok := content[itemsKey]; ok {
			if itemsSlice, ok := rawItems.([]interface{}); ok {
				items = itemsSlice
			}
		}

		if len(items) == 0 {
			return "", nil
		}

		// Determine list tag
		listTag := "ul"
		if ordered {
			if orderValue, ok := content["ordered"].(bool); ok && orderValue {
				listTag = "ol"
			}
		}

		listClass := fmt.Sprintf("%s__list", prefix)
		itemClass := fmt.Sprintf("%s__list-item", prefix)

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`<%s class="%s">`, listTag, listClass))

		for i, item := range items {
			itemHTML := renderItem(item, i, prefix)
			if itemHTML != "" {
				sb.WriteString(fmt.Sprintf(`<li class="%s">%s</li>`, itemClass, itemHTML))
			}
		}

		sb.WriteString(fmt.Sprintf(`</%s>`, listTag))
		return sb.String(), nil
	}
}

// ConditionalRenderer wraps a renderer with a condition check.
func ConditionalRenderer(
	condition func(ctx RenderContext) bool,
	renderer Renderer,
	fallback Renderer,
) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		if condition(ctx) {
			return renderer(ctx, prefix, elem)
		}
		if fallback != nil {
			return fallback(ctx, prefix, elem)
		}
		return "", nil
	}
}

// CachedRenderer wraps a renderer with simple in-memory caching based on element ID.
func CachedRenderer(renderer Renderer, cache map[string]cachedResult) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		// Check cache
		if cached, ok := cache[elem.ID]; ok {
			return cached.html, cached.scripts
		}

		// Render and cache
		html, scripts := renderer(ctx, prefix, elem)
		cache[elem.ID] = cachedResult{html: html, scripts: scripts}
		return html, scripts
	}
}

type cachedResult struct {
	html    string
	scripts []string
}

// ChainRenderer executes multiple renderers in sequence and combines their outputs.
func ChainRenderer(renderers ...Renderer) Renderer {
	return func(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
		var htmlParts []string
		var allScripts []string

		for _, renderer := range renderers {
			if renderer == nil {
				continue
			}
			html, scripts := renderer(ctx, prefix, elem)
			if html != "" {
				htmlParts = append(htmlParts, html)
			}
			if len(scripts) > 0 {
				allScripts = append(allScripts, scripts...)
			}
		}

		// Deduplicate scripts
		uniqueScripts := make([]string, 0, len(allScripts))
		seen := make(map[string]bool)
		for _, script := range allScripts {
			if !seen[script] {
				uniqueScripts = append(uniqueScripts, script)
				seen[script] = true
			}
		}

		return strings.Join(htmlParts, ""), uniqueScripts
	}
}
