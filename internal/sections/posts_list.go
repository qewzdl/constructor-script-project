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

// RegisterPostsList registers the posts list section renderer.
func RegisterPostsList(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("posts_list", renderPostsList)
}

// RegisterPostsListWithMetadata registers posts list with full metadata.
func RegisterPostsListWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	desc := &SectionDescriptor{
		Renderer: renderPostsList,
		Metadata: SectionMetadata{
			Type:        "posts_list",
			Name:        "Posts List",
			Description: "Displays a list of recent blog posts",
			Category:    "content",
			Icon:        "list",
			Schema: map[string]interface{}{
				"limit": map[string]interface{}{
					"type":    "number",
					"default": constants.DefaultPostListSectionLimit,
					"min":     1,
					"max":     constants.MaxPostListSectionLimit,
				},
			},
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderPostsList(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	listClass := fmt.Sprintf("%s__post-list", prefix)
	emptyClass := fmt.Sprintf("%s__post-list-empty", prefix)
	cardClass := fmt.Sprintf("%s__post-card post-card", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultPostListSectionLimit
	}
	if limit > constants.MaxPostListSectionLimit {
		limit = constants.MaxPostListSectionLimit
	}

	// Get post service from context
	services := ctx.Services()
	if services == nil {
		return `<p class="` + emptyClass + `">Posts are not available right now.</p>`, nil
	}

	postSvc, ok := services.PostService().(*blogservice.PostService)
	if !ok || postSvc == nil {
		return `<p class="` + emptyClass + `">Posts are not available right now.</p>`, nil
	}

	posts, err := postSvc.GetRecentPosts(limit)
	if err != nil {
		logger.Error(err, "Failed to load posts for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load posts at the moment. Please try again later.</p>`, nil
	}

	if len(posts) == 0 {
		return `<p class="` + emptyClass + `">No posts available yet. Check back soon!</p>`, nil
	}

	tmpl, err := ctx.CloneTemplates()
	if err != nil {
		logger.Error(err, "Failed to clone templates for post list section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to display posts at the moment.</p>`, nil
	}

	var sb strings.Builder
	sb.WriteString(`<div class="` + listClass + `">`)
	rendered := 0
	for i := range posts {
		post := posts[i]
		titleID := fmt.Sprintf("%s-post-%d", prefix, i+1)
		card, renderErr := renderPostCard(tmpl, &post, cardClass, titleID)
		if renderErr != nil {
			logger.Error(renderErr, "Failed to render post card", map[string]interface{}{"post_id": post.ID, "section_id": section.ID})
			continue
		}
		sb.WriteString(card)
		rendered++
	}
	sb.WriteString(`</div>`)

	if rendered == 0 {
		return `<p class="` + emptyClass + `">Unable to display posts at the moment.</p>`, nil
	}

	return sb.String(), nil
}

func renderPostCard(tmpl *template.Template, post *models.Post, cardClass, titleID string) (string, error) {
	data := map[string]interface{}{
		"Post":    post,
		"Class":   cardClass,
		"TitleID": titleID,
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "post-card", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func extractSection(elem models.SectionElement) (models.Section, bool) {
	if section, ok := elem.Content.(models.Section); ok {
		return section, true
	}
	if sectionPtr, ok := elem.Content.(*models.Section); ok && sectionPtr != nil {
		return *sectionPtr, true
	}
	return models.Section{}, false
}
