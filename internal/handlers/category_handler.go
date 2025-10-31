package handlers

import (
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// SetService updates the category service reference.
func (h *CategoryHandler) SetService(categoryService *service.CategoryService) {
	if h == nil {
		return
	}
	h.categoryService = categoryService
}

func (h *CategoryHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.categoryService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "posts plugin is not active"})
		return false
	}
	return true
}

func (h *CategoryHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	var req models.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.categoryService.Create(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"category": category})
}

func (h *CategoryHandler) GetAll(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	categories, err := h.categoryService.GetAll()
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

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	category, err := h.categoryService.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "category not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}

func (h *CategoryHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	var req models.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	category, err := h.categoryService.Update(uint(id), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"category": category})
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category id"})
		return
	}

	if err := h.categoryService.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "category deleted successfully"})
}
