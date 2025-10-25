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

func (h *SearchHandler) Search(c *gin.Context) {
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
