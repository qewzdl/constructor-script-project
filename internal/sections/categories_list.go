package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
	blogservice "constructor-script-backend/plugins/blog/service"
)

// RegisterCategoriesList registers the categories list section renderer.
func RegisterCategoriesList(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("categories_list", renderCategoriesList)
}

// RegisterCategoriesListWithMetadata registers categories list with full metadata.
func RegisterCategoriesListWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	desc := &SectionDescriptor{
		Renderer: renderCategoriesList,
		Metadata: SectionMetadata{
			Type:        "categories_list",
			Name:        "Categories List",
			Description: "Displays a list of blog categories",
			Category:    "navigation",
			Icon:        "tag",
			Schema: map[string]interface{}{
				"limit": map[string]interface{}{
					"type":    "number",
					"default": constants.DefaultCategoryListSectionLimit,
					"min":     1,
					"max":     constants.MaxCategoryListSectionLimit,
				},
			},
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderCategoriesList(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	emptyClass := fmt.Sprintf("%s__category-list-empty content__empty", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultCategoryListSectionLimit
	}
	if limit > constants.MaxCategoryListSectionLimit {
		limit = constants.MaxCategoryListSectionLimit
	}

	services := ctx.Services()
	if services == nil {
		return `<p class="` + emptyClass + `">Categories are not available right now.</p>`, nil
	}

	categorySvc, ok := services.CategoryService().(*blogservice.CategoryService)
	if !ok || categorySvc == nil {
		return `<p class="` + emptyClass + `">Categories are not available right now.</p>`, nil
	}

	categories, err := categorySvc.GetAll()
	if err != nil {
		logger.Error(err, "Failed to load categories for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load categories at the moment. Please try again later.</p>`, nil
	}

	filtered := make([]models.Category, 0, len(categories))
	for _, category := range categories {
		if strings.EqualFold(category.Slug, "uncategorized") || strings.EqualFold(category.Name, "uncategorized") {
			continue
		}
		filtered = append(filtered, category)
	}

	if len(filtered) == 0 {
		// If all categories were filtered out (for example only an "uncategorized"
		// category exists), fall back to showing the original categories so the
		// user still sees available categories instead of an empty message.
		filtered = categories
	}

	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	listClass := fmt.Sprintf("%s__category-list", prefix)
	itemClass := fmt.Sprintf("%s__category-item", prefix)
	linkClass := fmt.Sprintf("%s__category-link", prefix)

	var sb strings.Builder
	sb.WriteString(`<ul class="` + listClass + `">`)
	for _, category := range filtered {
		sb.WriteString(`<li class="` + itemClass + `">`)
		sb.WriteString(`<a href="/blog/category/` + template.HTMLEscapeString(category.Slug) + `" class="` + linkClass + `">`)
		sb.WriteString(template.HTMLEscapeString(category.Name))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}
	sb.WriteString(`</ul>`)

	return sb.String(), nil
}
