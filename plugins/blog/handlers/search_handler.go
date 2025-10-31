package bloghandlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	blogservice "constructor-script-backend/plugins/blog/service"
)

type SearchHandler struct {
	searchService *blogservice.SearchService
}

func NewSearchHandler(searchService *blogservice.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// SetService updates the search service reference.
func (h *SearchHandler) SetService(searchService *blogservice.SearchService) {
	if h == nil {
		return
	}
	h.searchService = searchService
}

func (h *SearchHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.searchService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "blog plugin is not active"})
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
	limitStr := c.DefaultQuery("limit", strconv.Itoa(blogservice.DefaultSearchLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = blogservice.DefaultSearchLimit
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
	limitStr := c.DefaultQuery("limit", strconv.Itoa(blogservice.DefaultSuggestionLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = blogservice.DefaultSuggestionLimit
	}

	tags, err := h.searchService.SuggestTags(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
