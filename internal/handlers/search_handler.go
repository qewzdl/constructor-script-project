package handlers

import (
	"constructor-script-backend/internal/service"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	searchService *service.SearchService
}

func NewSearchHandler(searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// SetService updates the search service reference.
func (h *SearchHandler) SetService(searchService *service.SearchService) {
	if h == nil {
		return
	}
	h.searchService = searchService
}

func (h *SearchHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.searchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "posts plugin is not active"})
		return false
	}
	return true
}

func (h *SearchHandler) Search(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	query := c.Query("q")
	searchType := c.DefaultQuery("type", "all")
	limitStr := c.DefaultQuery("limit", strconv.Itoa(service.DefaultSearchLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = service.DefaultSearchLimit
	}

	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query is required"})
		return
	}

	result, err := h.searchService.Search(query, searchType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *SearchHandler) SuggestTags(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	query := c.Query("q")
	limitStr := c.DefaultQuery("limit", strconv.Itoa(service.DefaultSuggestionLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = service.DefaultSuggestionLimit
	}

	tags, err := h.searchService.SuggestTags(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
