package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(page, totalPages, func(p int) string {
		return fmt.Sprintf("/?page=%d", p)
	})

	h.renderTemplate(c, "index", "Home", h.config.SiteDescription, gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
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

	var tags []models.Tag
	if loadedTags, tagErr := h.postService.GetAllTags(); tagErr != nil {
		logger.Error(tagErr, "Failed to load tags", nil)
	} else {
		tags = loadedTags
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(page, totalPages, func(p int) string {
		return fmt.Sprintf("/blog?page=%d", p)
	})

	data := gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	h.renderTemplate(c, "blog", "Blog", h.config.SiteName+" Blog — insights about Go programming, web technologies, performance, and best practices in backend design.", data)
}

func (h *TemplateHandler) RenderSearch(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	searchType := c.DefaultQuery("type", "all")
	limitValue := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitValue)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	var result *service.SearchResult
	if query != "" {
		searchResult, searchErr := h.searchService.Search(query, searchType, limit)
		if searchErr != nil {
			logger.Error(searchErr, "Failed to execute search", map[string]interface{}{"query": query, "type": searchType})
			h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to perform search")
			return
		}
		result = searchResult
	} else {
		result = &service.SearchResult{Posts: []models.Post{}, Total: 0, Query: query}
	}

	hasQuery := query != ""
	title := "Search"
	description := fmt.Sprintf("Search articles on %s to discover tutorials, guides, and engineering stories.", h.config.SiteName)
	if hasQuery {
		title = fmt.Sprintf("Search results for \"%s\"", query)
		description = fmt.Sprintf("Search results for \"%s\" on %s.", query, h.config.SiteName)
	}

	data := gin.H{
		"Query":       query,
		"SearchType":  searchType,
		"Limit":       limit,
		"HasQuery":    hasQuery,
		"Result":      result,
		"SearchQuery": query,
		"Styles":      []string{"/static/css/search.css"},
	}

	if result != nil {
		data["Total"] = result.Total
	}

	h.renderTemplate(c, "search", title, description, data)
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

	tag, posts, total, err := h.postService.GetTagWithPosts(slug, page, limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested tag not found")
		} else {
			h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		}
		return
	}

	var tags []models.Tag
	if loadedTags, tagErr := h.postService.GetAllTags(); tagErr != nil {
		logger.Error(tagErr, "Failed to load tags", nil)
	} else {
		tags = loadedTags
	}

	totalCount := int(total)

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(page, totalPages, func(p int) string {
		return fmt.Sprintf("/tag/%s?page=%d", slug, p)
	})

	data := gin.H{
		"Posts":       posts,
		"Total":       totalCount,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
		"Tag":         tag,
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	description := "Articles tagged with \"" + tag.Name + "\" — development topics, code patterns, and Go programming insights on " + h.config.SiteName + "."
	h.renderTemplate(c, "tag", "Tag: "+tag.Name, description, data)
}

func (h *TemplateHandler) RenderLogin(c *gin.Context) {
	if _, ok := h.currentUser(c); ok {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo != "" {
		if decoded, err := url.QueryUnescape(redirectTo); err == nil {
			redirectTo = decoded
		}

		if !strings.HasPrefix(redirectTo, "/") {
			redirectTo = "/profile"
		}
	} else {
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

func (h *TemplateHandler) RenderSetup(c *gin.Context) {
	if h.setupService == nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Setup is not available")
		return
	}

	complete, err := h.setupService.IsSetupComplete()
	if err != nil {
		logger.Error(err, "Failed to determine setup status", nil)
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to verify setup status")
		return
	}

	if complete {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	h.renderTemplate(c, "setup", "Initial setup", "Create the first administrator account and configure the site.", gin.H{
		"Scripts":     []string{"/static/js/setup.js"},
		"SetupAction": "/api/v1/setup",
		"SetupStatus": "/api/v1/setup/status",
		"HideChrome":  true,
		"NoIndex":     true,
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

func (h *TemplateHandler) RenderAdmin(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		redirectTo := url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, "/login?redirect="+redirectTo)
		return
	}

	if user.Role != "admin" {
		h.renderError(c, http.StatusForbidden, "403 - Forbidden", "Administrator access required")
		return
	}

	h.renderTemplate(c, "admin", "Admin dashboard", "Monitor site activity, review content performance, and manage published resources in one place.", gin.H{
		"Styles":  []string{"/static/css/admin.css"},
		"Scripts": []string{"/static/js/admin.js"},
		"AdminEndpoints": gin.H{
			"Stats":           "/api/v1/admin/stats",
			"Posts":           "/api/v1/admin/posts",
			"Pages":           "/api/v1/admin/pages",
			"Categories":      "/api/v1/admin/categories",
			"CategoriesIndex": "/api/v1/categories",
			"Comments":        "/api/v1/admin/comments",
			"Tags":            "/api/v1/tags",
		},
		"NoIndex": true,
	})
}

func (h *TemplateHandler) buildPagination(current, total int, buildURL func(page int) string) gin.H {
	if total <= 1 || buildURL == nil {
		return nil
	}

	data := gin.H{
		"CurrentPage": current,
		"TotalPages":  total,
	}

	if current > 1 {
		data["PrevURL"] = buildURL(current - 1)
	}

	if current < total {
		data["NextURL"] = buildURL(current + 1)
	}

	return data
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
