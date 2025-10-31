package bloghandlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	blogservice "constructor-script-backend/plugins/blog/service"
)

type PostHandler struct {
	postService *blogservice.PostService
}

func NewPostHandler(postService *blogservice.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

// SetService updates the underlying post service reference.
func (h *PostHandler) SetService(postService *blogservice.PostService) {
	if h == nil {
		return
	}
	h.postService = postService
}

func (h *PostHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.postService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "blog plugin is not active"})
		return false
	}
	return true
}

func (h *PostHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	post, err := h.postService.Create(req, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"post": post})
}

func (h *PostHandler) GetAll(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var categoryID *uint
	if catID := c.Query("category_id"); catID != "" {
		id, _ := strconv.ParseUint(catID, 10, 32)
		cid := uint(id)
		categoryID = &cid
	}

	var tagName *string
	if tag := c.Query("tag"); tag != "" {
		tagName = &tag
	}

	var authorID *uint
	if authID := c.Query("author_id"); authID != "" {
		id, _ := strconv.ParseUint(authID, 10, 32)
		aid := uint(id)
		authorID = &aid
	}

	posts, total, err := h.postService.GetAll(page, limit, categoryID, tagName, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *PostHandler) GetByID(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	post, err := h.postService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"post": post})
}

func (h *PostHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req models.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)

	post, err := h.postService.Update(uint(id), req, userID, canManageAll)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"post": post})
}

func (h *PostHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)

	if err := h.postService.Delete(uint(id), userID, canManageAll); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post deleted successfully"})
}

func (h *PostHandler) GetBySlug(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	slug := c.Param("slug")

	post, err := h.postService.GetBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"post": post})
}

func (h *PostHandler) GetAllTags(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	tags, err := h.postService.GetAllTags()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func (h *PostHandler) DeleteTag(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
		return
	}

	if err := h.postService.DeleteTag(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "tag deleted successfully"})
}

func (h *PostHandler) GetAnalytics(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil {
		days = 30
	}

	analytics, err := h.postService.GetAnalytics(uint(id), days)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "post not found"})
			return
		case errors.Is(err, blogservice.ErrPostNotPublished):
			c.JSON(http.StatusBadRequest, gin.H{"error": "analytics available only for published posts"})
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"analytics": analytics})
}

func (h *PostHandler) GetPostsByTag(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	posts, total, err := h.postService.GetPostsByTag(slug, page, limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *PostHandler) GetAllAdmin(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	posts, total, err := h.postService.GetAllAdmin(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *PostHandler) PublishPost(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	if err := h.postService.PublishPost(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post published successfully"})
}

func (h *PostHandler) UnpublishPost(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	if err := h.postService.UnpublishPost(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "post unpublished successfully"})
}
