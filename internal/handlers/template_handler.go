package handlers

import (
	"bytes"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TemplateHandler struct {
	postService *service.PostService
	pageService *service.PageService
	templates   *template.Template
}

func NewTemplateHandler(postService *service.PostService, pageService *service.PageService, templatesDir string) (*TemplateHandler, error) {

	tmpl := template.New("").Funcs(utils.GetTemplateFuncs())

	templates, err := tmpl.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, err
	}

	logger.Info("Loaded templates:", map[string]interface{}{
		"templates": templates.DefinedTemplates(),
	})

	return &TemplateHandler{
		postService: postService,
		pageService: pageService,
		templates:   templates,
	}, nil
}

func (h *TemplateHandler) RenderIndex(c *gin.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error(nil, "Panic in RenderIndex", map[string]interface{}{
				"panic": r,
			})
			c.String(http.StatusInternalServerError, "Internal error: %v", r)
		}
	}()

	logger.Info("RenderIndex called", nil)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		logger.Error(err, "Failed to load posts", nil)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Title":      "500 - Server Error",
			"error":      "Failed to load posts",
			"StatusCode": 500,
		})
		return
	}

	logger.Info("Posts loaded", map[string]interface{}{
		"count": len(posts),
		"total": total,
	})

	data := gin.H{
		"Title":       "Constructor Script - Home",
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
	}

	h.renderWithLayout(c, "base.html", "index.html", data)
}

func (h *TemplateHandler) RenderPost(c *gin.Context) {
	param := c.Param("slug")

	if id, err := strconv.ParseUint(param, 10, 32); err == nil {

		post, err := h.postService.GetByID(uint(id))
		if err != nil {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"Title":      "404 - Post Not Found",
				"error":      "Requested post not found",
				"StatusCode": 404,
			})
			return
		}
		h.renderPost(c, post)
		return
	}

	post, err := h.postService.GetBySlug(param)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title":      "404 - Post Not Found",
			"error":      "Requested post not found",
			"StatusCode": 404,
		})
		return
	}

	h.renderPost(c, post)
}

func (h *TemplateHandler) renderPost(c *gin.Context, post *models.Post) {

	relatedPosts, _ := h.postService.GetRelatedPosts(post.ID, 3)

	data := gin.H{
		"Title":        post.Title,
		"Description":  post.Description,
		"Post":         post,
		"RelatedPosts": relatedPosts,
		"Content":      h.renderSections(post.Sections),
	}

	templateName := post.Template
	if templateName == "" {
		templateName = "post"
	}
	templateFile := templateName + ".html"

	h.renderWithLayout(c, "base.html", templateFile, data)
}

func (h *TemplateHandler) RenderPage(c *gin.Context) {
	slug := c.Param("slug")

	page, err := h.pageService.GetBySlug(slug)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"Title":      "404 - Page Not Found",
			"error":      "Requested page not found",
			"StatusCode": 404,
		})
		return
	}

	data := gin.H{
		"Title":       page.Title,
		"Description": page.Description,
		"Page":        page,
		"Content":     h.renderSections(page.Sections),
	}

	templateName := page.Template
	if templateName == "" {
		templateName = "page"
	}
	templateFile := templateName + ".html"

	h.renderWithLayout(c, "base.html", templateFile, data)
}

func (h *TemplateHandler) renderWithLayout(c *gin.Context, layout, content string, data gin.H) {
	logger.Info("renderWithLayout called", map[string]interface{}{
		"layout":  layout,
		"content": content,
	})

	tmpl, err := h.templates.Clone()
	if err != nil {
		logger.Error(err, "Failed to clone templates", nil)
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Template error",
		})
		return
	}

	contentTmpl := tmpl.Lookup(content)
	if contentTmpl == nil {
		logger.Error(nil, "Content template not found", map[string]interface{}{
			"template": content,
		})
		c.HTML(http.StatusNotFound, "error.html", gin.H{
			"error": "Template not found: " + content,
		})
		return
	}

	var contentBuffer []byte
	contentBuffer, err = h.executeTemplate(contentTmpl, data)
	if err != nil {
		logger.Error(err, "Failed to render content", map[string]interface{}{
			"template": content,
		})
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to render content",
		})
		return
	}

	data["Content"] = template.HTML(contentBuffer)

	logger.Info("Rendering layout", map[string]interface{}{
		"layout": layout,
	})

	c.HTML(http.StatusOK, layout, data)
}

func (h *TemplateHandler) executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *TemplateHandler) renderSections(sections models.PostSections) template.HTML {
	if len(sections) == 0 {
		return ""
	}

	var html string

	for _, section := range sections {
		html += `<section class="post-section" id="section-` + section.ID + `">`

		html += `<h2 class="section-title">` + template.HTMLEscapeString(section.Title) + `</h2>`

		if section.Image != "" {
			html += `<div class="section-image">`
			html += `<img src="` + template.HTMLEscapeString(section.Image) + `" alt="` + template.HTMLEscapeString(section.Title) + `" />`
			html += `</div>`
		}

		for _, elem := range section.Elements {
			html += h.renderSectionElement(elem)
		}

		html += `</section>`
	}

	return template.HTML(html)
}

func (h *TemplateHandler) RenderBlog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"Title":      "500 - Server Error",
			"error":      "Failed to load posts",
			"StatusCode": 500,
		})
		return
	}

	data := gin.H{
		"Title":       "Blog - Constructor Script",
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
	}

	h.renderWithLayout(c, "base.html", "blog.html", data)
}

func (h *TemplateHandler) RenderCategory(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load posts",
		})
		return
	}

	data := gin.H{
		"Title":       "Category: " + slug,
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
		"Category":    slug,
	}

	h.renderWithLayout(c, "base.html", "category.html", data)
}

func (h *TemplateHandler) RenderTag(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetPostsByTag(slug, page, limit)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": "Failed to load posts",
		})
		return
	}

	data := gin.H{
		"Title":       "Tag: " + slug,
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
		"Tag":         slug,
	}

	h.renderWithLayout(c, "base.html", "tag.html", data)
}

func (h *TemplateHandler) renderSectionElement(elem models.SectionElement) string {
	var html string

	switch elem.Type {
	case "paragraph":
		if contentMap, ok := elem.Content.(map[string]interface{}); ok {
			if text, ok := contentMap["text"].(string); ok {
				html += `<div class="paragraph">` + text + `</div>`
			}
		}

	case "image":
		if contentMap, ok := elem.Content.(map[string]interface{}); ok {
			url, _ := contentMap["url"].(string)
			alt, _ := contentMap["alt"].(string)
			caption, _ := contentMap["caption"].(string)

			html += `<figure class="image-element">`
			html += `<img src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`
			if caption != "" {
				html += `<figcaption>` + template.HTMLEscapeString(caption) + `</figcaption>`
			}
			html += `</figure>`
		}

	case "image_group":
		if contentMap, ok := elem.Content.(map[string]interface{}); ok {
			layout, _ := contentMap["layout"].(string)
			if layout == "" {
				layout = "grid"
			}

			html += `<div class="image-group image-group-` + layout + `">`

			if images, ok := contentMap["images"].([]interface{}); ok {
				for _, img := range images {
					if imgMap, ok := img.(map[string]interface{}); ok {
						url, _ := imgMap["url"].(string)
						alt, _ := imgMap["alt"].(string)
						caption, _ := imgMap["caption"].(string)

						html += `<figure class="image-group-item">`
						html += `<img src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`
						if caption != "" {
							html += `<figcaption>` + template.HTMLEscapeString(caption) + `</figcaption>`
						}
						html += `</figure>`
					}
				}
			}

			html += `</div>`
		}
	}

	return html
}
