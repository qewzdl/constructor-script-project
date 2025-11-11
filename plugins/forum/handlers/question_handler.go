package forumhandlers

import (
        "errors"
        "net/http"
        "strconv"
        "strings"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	forumservice "constructor-script-backend/plugins/forum/service"
)

type QuestionHandler struct {
	service *forumservice.QuestionService
}

func NewQuestionHandler(service *forumservice.QuestionService) *QuestionHandler {
	return &QuestionHandler{service: service}
}

func (h *QuestionHandler) SetService(service *forumservice.QuestionService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *QuestionHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "forum plugin is not active"})
		return false
	}
	return true
}

func (h *QuestionHandler) List(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")

	var authorID *uint
	if authorParam := c.Query("author_id"); authorParam != "" {
		if parsed, err := strconv.ParseUint(authorParam, 10, 64); err == nil {
			value := uint(parsed)
			authorID = &value
		}
	}

	questions, total, err := h.service.List(page, limit, search, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"questions": questions,
		"total":     total,
		"page":      page,
		"limit":     limit,
	})
}

func (h *QuestionHandler) GetByID(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
        increment := true
        switch strings.ToLower(strings.TrimSpace(c.DefaultQuery("increment", "true"))) {
        case "false", "0", "no":
                increment = false
        }

        var question *models.ForumQuestion
        if increment {
                question, err = h.service.GetByID(uint(id))
        } else {
                question, err = h.service.GetByIDWithoutIncrement(uint(id))
        }
        if err != nil {
                if errors.Is(err, forumservice.ErrQuestionNotFound) {
                        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
                        return
                }
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"question": question})
}

func (h *QuestionHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	var req models.CreateForumQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorID := c.GetUint("user_id")
	question, err := h.service.Create(req, authorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"question": question})
}

func (h *QuestionHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	var req models.UpdateForumQuestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)
	question, err := h.service.Update(uint(id), req, userID, canManageAll)
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrQuestionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, forumservice.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"question": question})
}

func (h *QuestionHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)
	if err := h.service.Delete(uint(id), userID, canManageAll); err != nil {
		switch {
		case errors.Is(err, forumservice.ErrQuestionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, forumservice.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *QuestionHandler) Vote(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	var req models.ForumVoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	rating, err := h.service.Vote(uint(id), userID, req.Value)
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrQuestionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, forumservice.ErrInvalidVoteValue):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"rating": rating})
}
