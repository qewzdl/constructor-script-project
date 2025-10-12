package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"constructor-script-backend/internal/models"
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

	var tags []models.Tag
	if loadedTags, tagErr := h.postService.GetAllTags(); tagErr != nil {
		logger.Error(tagErr, "Failed to load tags", nil)
	} else {
		tags = loadedTags
	}

	data := gin.H{
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	h.renderTemplate(c, "blog", "Blog", h.config.SiteName+" Blog — insights about Go programming, web technologies, performance, and best practices in backend design.", data)
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

	data := gin.H{
		"Posts":       posts,
		"Total":       totalCount,
		"CurrentPage": page,
		"TotalPages":  int((total + int64(limit) - 1) / int64(limit)),
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
	})
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
