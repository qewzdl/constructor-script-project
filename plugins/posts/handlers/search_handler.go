package postshandlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	postservice "constructor-script-backend/plugins/posts/service"
)

type SearchHandler struct {
	searchService *postservice.SearchService
}

func NewSearchHandler(searchService *postservice.SearchService) *SearchHandler {
	return &SearchHandler{searchService: searchService}
}

// SetService updates the search service reference.
func (h *SearchHandler) SetService(searchService *postservice.SearchService) {
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
	limitStr := c.DefaultQuery("limit", strconv.Itoa(postservice.DefaultSearchLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = postservice.DefaultSearchLimit
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
	limitStr := c.DefaultQuery("limit", strconv.Itoa(postservice.DefaultSuggestionLimit))

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = postservice.DefaultSuggestionLimit
	}

	tags, err := h.searchService.SuggestTags(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
