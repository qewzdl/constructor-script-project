package forumhandlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/models"
	forumservice "constructor-script-backend/plugins/forum/service"
)

type CategoryHandler struct {
	service *forumservice.CategoryService
}

func NewCategoryHandler(service *forumservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{service: service}
}

func (h *CategoryHandler) SetService(service *forumservice.CategoryService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *CategoryHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "forum plugin is not active"})
		return false
	}
	return true
}

func (h *CategoryHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	includeCounts := strings.EqualFold(strings.TrimSpace(c.DefaultQuery("include_counts", "false")), "true")

	categories, err := h.service.List(includeCounts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (h *CategoryHandler) GetByID(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	category, err := h.service.GetByID(uint(id))
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}

func (h *CategoryHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	var req models.CreateForumCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := h.service.Create(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"category": category})
}

func (h *CategoryHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}
	var req models.UpdateForumCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	category, err := h.service.Update(uint(id), req)
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"category": category})
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}
	if err := h.service.Delete(uint(id)); err != nil {
		switch {
		case errors.Is(err, forumservice.ErrCategoryNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}
