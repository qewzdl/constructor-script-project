package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MenuHandler struct {
	service *service.MenuService
}

func NewMenuHandler(service *service.MenuService) *MenuHandler {
	return &MenuHandler{service: service}
}

func (h *MenuHandler) List(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	items, err := h.service.List()
	if err != nil {
		logger.Error(err, "Failed to load menu items", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load menu items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"menu_items": items})
}

func (h *MenuHandler) Create(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	var req models.CreateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.service.Create(req)
	if err != nil {
		logger.Error(err, "Failed to create menu item", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"menu_item": item})
}

func (h *MenuHandler) Update(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	idParam := c.Param("id")
	idValue, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	var req models.UpdateMenuItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	item, err := h.service.Update(uint(idValue), req)
	if err != nil {
		logger.Error(err, "Failed to update menu item", map[string]interface{}{"id": idValue})
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"menu_item": item})
}

func (h *MenuHandler) Delete(c *gin.Context) {
	if h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Service not configured"})
		return
	}

	idParam := c.Param("id")
	idValue, err := strconv.ParseUint(idParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menu item ID"})
		return
	}

	if err := h.service.Delete(uint(idValue)); err != nil {
		logger.Error(err, "Failed to delete menu item", map[string]interface{}{"id": idValue})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete menu item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Menu item deleted"})
}
