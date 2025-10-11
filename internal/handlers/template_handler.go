package handlers

import (
	"bytes"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	"html/template"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/microcosm-cc/bluemonday"
)

type TemplateHandler struct {
	postService    *service.PostService
	pageService    *service.PageService
	authService    *service.AuthService
	commentService *service.CommentService
	templates      *template.Template
	config         *config.Config
	sanitizer      *bluemonday.Policy
}

type CommentView struct {
	ID         uint
	AuthorName string
	CreatedAt  time.Time
	Content    template.HTML
	Replies    []CommentView
}

func NewTemplateHandler(postService *service.PostService, pageService *service.PageService, authService *service.AuthService, commentService *service.CommentService, cfg *config.Config, templatesDir string) (*TemplateHandler, error) {
	tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
	templates, err := tmpl.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		return nil, err
	}

	logger.Info("Loaded templates:", map[string]interface{}{
		"templates": templates.DefinedTemplates(),
	})

	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class", "id").Globally()
	policy.AllowAttrs("style").OnElements("span", "div", "p")

	return &TemplateHandler{
		postService:    postService,
		pageService:    pageService,
		authService:    authService,
		commentService: commentService,
		templates:      templates,
		config:         cfg,
		sanitizer:      policy,
	}, nil
}

func (h *TemplateHandler) currentUser(c *gin.Context) (*models.User, bool) {
	if h.authService == nil {
		return nil, false
	}

	clearCookie := func() {
		secure := c.Request.TLS != nil
		c.SetCookie(authTokenCookieName, "", -1, "/", "", secure, false)
	}

	tokenString, err := c.Cookie(authTokenCookieName)
	if err != nil || tokenString == "" {
		return nil, false
	}

	parsed, err := h.authService.ValidateToken(tokenString)
	if err != nil || parsed == nil || !parsed.Valid {
		clearCookie()
		return nil, false
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		clearCookie()
		return nil, false
	}

	userIDValue, ok := claims["user_id"]
	if !ok {
		clearCookie()
		return nil, false
	}

	var userID uint
	switch value := userIDValue.(type) {
	case float64:
		userID = uint(value)
	case int:
		userID = uint(value)
	default:
		clearCookie()
		return nil, false
	}

	user, err := h.authService.GetUserByID(userID)
	if err != nil {
		clearCookie()
		return nil, false
	}

	return user, true
}

func (h *TemplateHandler) addUserContext(c *gin.Context, data gin.H) {
	user, ok := h.currentUser(c)
	if !ok {
		data["IsAuthenticated"] = false
		return
	}

	data["IsAuthenticated"] = true
	data["CurrentUser"] = user
}

func (h *TemplateHandler) basePageData(title, description string, extra gin.H) gin.H {
	data := gin.H{
		"Title":       title + " - " + h.config.SiteName,
		"Description": description,
		"Site": gin.H{
			"Name":        h.config.SiteName,
			"Description": h.config.SiteDescription,
			"URL":         h.config.SiteURL,
			"Favicon":     h.config.SiteFavicon,
		},
	}

	for k, v := range extra {
		data[k] = v
	}

	return data
}

func (h *TemplateHandler) renderTemplate(c *gin.Context, templateName, title, description string, extra gin.H) {
	data := h.basePageData(title, description, extra)
	if templateName == "" {
		templateName = "page"
	}
	h.renderWithLayout(c, "base.html", templateName+".html", data)
}

func (h *TemplateHandler) renderSinglePost(c *gin.Context, post *models.Post) {
	related, _ := h.postService.GetRelatedPosts(post.ID, 3)

	var (
		comments     []CommentView
		commentCount int
	)

	if h.commentService != nil {
		if loaded, err := h.commentService.GetByPostID(post.ID); err != nil {
			logger.Error(err, "Failed to load comments for post", map[string]interface{}{"post_id": post.ID})
		} else {
			comments = h.buildCommentViews(loaded)
			commentCount = h.countComments(loaded)
		}
	}

	data := h.basePageData(post.Title, post.Description, gin.H{
		"Post":         post,
		"RelatedPosts": related,
		"Content":      h.renderSections(post.Sections),
		"TOC":          h.generateTOC(post.Sections),
		"Comments":     comments,
		"CommentCount": commentCount,
	})

	templateName := post.Template
	if templateName == "" {
		templateName = "post"
	}

	h.renderWithLayout(c, "base.html", templateName+".html", data)
}

func (h *TemplateHandler) RenderIndex(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	h.renderTemplate(c, "index", "Home", h.config.SiteDescription, gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *TemplateHandler) RenderPost(c *gin.Context) {
	param := c.Param("slug")

	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		post, err := h.postService.GetByID(uint(id))
		if err != nil {
			h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
			return
		}
		h.renderSinglePost(c, post)
		return
	}

	post, err := h.postService.GetBySlug(param)
	if err != nil {
		h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
		return
	}
	h.renderSinglePost(c, post)
}

func (h *TemplateHandler) RenderPage(c *gin.Context) {
	slug := c.Param("slug")
	page, err := h.pageService.GetBySlug(slug)
	if err != nil {
		h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
		return
	}

	h.renderTemplate(c, page.Template, page.Title, page.Description, gin.H{
		"Page":    page,
		"Content": h.renderSections(page.Sections),
	})
}

func (h *TemplateHandler) RenderBlog(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	h.renderTemplate(c, "blog", "Blog", h.config.SiteName+" Blog — insights about Go programming, web technologies, performance, and best practices in backend design.", gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
	})
}

func (h *TemplateHandler) RenderCategory(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	h.renderTemplate(c, "category", "Category: "+slug, "", gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
		"Category":    slug,
	})
}

func (h *TemplateHandler) RenderTag(c *gin.Context) {
	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	posts, total, err := h.postService.GetPostsByTag(slug, page, limit)
	if err != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	h.renderTemplate(c, "tag", "Tag: "+slug, "Articles tagged with \""+slug+"\" — development topics, code patterns, and Go programming insights on "+h.config.SiteName+".", gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
		"Tag":         slug,
	})
}

func (h *TemplateHandler) renderWithLayout(c *gin.Context, layout, content string, data gin.H) {
	h.addUserContext(c, data)

	tmpl, err := h.templates.Clone()
	if err != nil {
		logger.Error(err, "Failed to clone templates", nil)
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Template error")
		return
	}

	contentTmpl := tmpl.Lookup(content)
	if contentTmpl == nil {
		logger.Error(nil, "Content template not found", map[string]interface{}{"template": content})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Template not found")
		return
	}

	buf, err := h.executeTemplate(contentTmpl, data)
	if err != nil {
		logger.Error(err, "Failed to render content", map[string]interface{}{"template": content})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to render content")
		return
	}

	data["Content"] = template.HTML(buf)

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

	var sb strings.Builder

	for _, section := range sections {
		sb.WriteString(`<section class="post__section" id="section-` + template.HTMLEscapeString(section.ID) + `">`)
		sb.WriteString(`<h2 class="post__section-title">` + template.HTMLEscapeString(section.Title) + `</h2>`)

		if section.Image != "" {
			sb.WriteString(`<div class="post__section-image">`)
			sb.WriteString(`<img class="post__section-img" src="` + template.HTMLEscapeString(section.Image) + `" alt="` + template.HTMLEscapeString(section.Title) + `" />`)
			sb.WriteString(`</div>`)
		}

		for _, elem := range section.Elements {
			sb.WriteString(h.renderSectionElement(elem))
		}

		sb.WriteString(`</section>`)
	}

	return template.HTML(sb.String())
}

func (h *TemplateHandler) renderSectionElement(elem models.SectionElement) string {
	var sb strings.Builder

	contentMap, _ := elem.Content.(map[string]interface{})

	switch elem.Type {
	case "paragraph":
		if text, ok := contentMap["text"].(string); ok {
			sanitized := h.sanitizer.Sanitize(text)
			sb.WriteString(`<div class="post__paragraph">` + sanitized + `</div>`)
		}

	case "image":
		url, _ := contentMap["url"].(string)
		alt, _ := contentMap["alt"].(string)
		caption, _ := contentMap["caption"].(string)

		sb.WriteString(`<figure class="post__image">`)
		sb.WriteString(`<img class="post__image-img" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
		if caption != "" {
			sanitizedCaption := h.sanitizer.Sanitize(caption)
			sb.WriteString(`<figcaption class="post__image-caption">` + sanitizedCaption + `</figcaption>`)
		}
		sb.WriteString(`</figure>`)

	case "image_group":
		layout, _ := contentMap["layout"].(string)
		if layout == "" {
			layout = "grid"
		}
		sb.WriteString(`<div class="post__image-group post__image-group--` + template.HTMLEscapeString(layout) + `">`)

		if images, ok := contentMap["images"].([]interface{}); ok {
			for _, img := range images {
				if imgMap, ok := img.(map[string]interface{}); ok {
					url, _ := imgMap["url"].(string)
					alt, _ := imgMap["alt"].(string)
					caption, _ := imgMap["caption"].(string)

					sb.WriteString(`<figure class="post__image-group-item">`)
					sb.WriteString(`<img class="post__image-group-img" src="` + template.HTMLEscapeString(url) + `" alt="` + template.HTMLEscapeString(alt) + `" />`)
					if caption != "" {
						sanitizedCaption := h.sanitizer.Sanitize(caption)
						sb.WriteString(`<figcaption class="post__image-group-caption">` + sanitizedCaption + `</figcaption>`)
					}
					sb.WriteString(`</figure>`)
				}
			}
		}
		sb.WriteString(`</div>`)
	}

	return sb.String()
}

func (h *TemplateHandler) generateTOC(sections models.PostSections) template.HTML {
	if len(sections) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(`<nav class="post__toc" aria-label="Table of contents">`)
	sb.WriteString(`<h2 class="post__toc-title">Table of Contents</h2>`)
	sb.WriteString(`<ol class="post__toc-list">`)

	for _, section := range sections {
		sb.WriteString(`<li class="post__toc-item">`)
		sb.WriteString(`<a href="#section-` + template.HTMLEscapeString(section.ID) + `" class="post__toc-link">`)
		sb.WriteString(template.HTMLEscapeString(section.Title))
		sb.WriteString(`</a>`)
		sb.WriteString(`</li>`)
	}

	sb.WriteString(`</ol>`)
	sb.WriteString(`</nav>`)

	return template.HTML(sb.String())
}

func (h *TemplateHandler) renderError(c *gin.Context, status int, title, msg string) {
	data := gin.H{
		"Title":      title,
		"error":      msg,
		"StatusCode": status,
		"Site": gin.H{
			"Name":        h.config.SiteName,
			"Description": h.config.SiteDescription,
			"URL":         h.config.SiteURL,
			"Favicon":     h.config.SiteFavicon,
		},
	}
	c.HTML(status, "error.html", data)
}

func (h *TemplateHandler) RenderLogin(c *gin.Context) {
	if _, ok := h.currentUser(c); ok {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo == "" {
		redirectTo = "/profile"
	}

	h.renderTemplate(c, "login", "Sign in", "Access your dashboard and manage your content.", gin.H{
		"AuthAction": "/api/v1/login",
		"RedirectTo": redirectTo,
	})
}

func (h *TemplateHandler) RenderRegister(c *gin.Context) {
	if _, ok := h.currentUser(c); ok {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	h.renderTemplate(c, "register", "Create an account", "Join the community to publish articles and leave comments.", gin.H{
		"RegisterAction": "/api/v1/register",
	})
}

func (h *TemplateHandler) RenderProfile(c *gin.Context) {
	_, ok := h.currentUser(c)
	if !ok {
		redirectTo := url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, "/login?redirect="+redirectTo)
		return
	}

	h.renderTemplate(c, "profile", "Profile", "Manage personal details, account security, and connected devices.", gin.H{
		"ProfileAction":        "/api/v1/profile",
		"PasswordChangeAction": "/api/v1/profile/password",
	})
}

func (h *TemplateHandler) buildCommentViews(comments []models.Comment) []CommentView {
	if len(comments) == 0 {
		return nil
	}

	views := make([]CommentView, 0, len(comments))
	for i := range comments {
		comment := &comments[i]
		if !comment.Approved {
			continue
		}
		views = append(views, h.buildCommentView(comment))
	}

	return views
}

func (h *TemplateHandler) buildCommentView(comment *models.Comment) CommentView {
	authorName := "Anonymous"
	if comment.Author.Username != "" {
		authorName = comment.Author.Username
	}

	view := CommentView{
		ID:         comment.ID,
		AuthorName: authorName,
		CreatedAt:  comment.CreatedAt,
		Content:    h.sanitizeCommentContent(comment.Content),
	}

	if len(comment.Replies) > 0 {
		replies := make([]CommentView, 0, len(comment.Replies))
		for _, reply := range comment.Replies {
			if reply == nil || !reply.Approved {
				continue
			}
			replies = append(replies, h.buildCommentView(reply))
		}
		view.Replies = replies
	}

	return view
}

func (h *TemplateHandler) countComments(comments []models.Comment) int {
	total := 0
	for i := range comments {
		comment := &comments[i]
		if !comment.Approved {
			continue
		}
		total++
		total += h.countCommentReplies(comment.Replies)
	}
	return total
}

func (h *TemplateHandler) countCommentReplies(replies []*models.Comment) int {
	total := 0
	for _, reply := range replies {
		if reply == nil || !reply.Approved {
			continue
		}
		total++
		total += h.countCommentReplies(reply.Replies)
	}
	return total
}

func (h *TemplateHandler) sanitizeCommentContent(content string) template.HTML {
	if content == "" {
		return ""
	}

	if h.sanitizer == nil {
		escaped := template.HTMLEscapeString(content)
		escaped = strings.ReplaceAll(escaped, "\n", "<br />")
		return template.HTML(escaped)
	}

	sanitized := h.sanitizer.Sanitize(content)
	sanitized = strings.ReplaceAll(sanitized, "\n", "<br />")
	return template.HTML(sanitized)
}
