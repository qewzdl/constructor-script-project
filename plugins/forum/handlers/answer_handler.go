package forumhandlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	forumservice "constructor-script-backend/plugins/forum/service"
)

type AnswerHandler struct {
	service *forumservice.AnswerService
}

func NewAnswerHandler(service *forumservice.AnswerService) *AnswerHandler {
	return &AnswerHandler{service: service}
}

func (h *AnswerHandler) SetService(service *forumservice.AnswerService) {
	if h == nil {
		return
	}
	h.service = service
}

func (h *AnswerHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.service == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "forum plugin is not active"})
		return false
	}
	return true
}

func (h *AnswerHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	questionIDValue, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid question id"})
		return
	}
	var req models.CreateForumAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authorID := c.GetUint("user_id")
	answer, err := h.service.Create(uint(questionIDValue), authorID, req)
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrQuestionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, gin.H{"answer": answer})
}

func (h *AnswerHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id"})
		return
	}
	var req models.UpdateForumAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)
	answer, err := h.service.Update(uint(id), req, userID, canManageAll)
	if err != nil {
		switch {
		case errors.Is(err, forumservice.ErrAnswerNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case errors.Is(err, forumservice.ErrUnauthorized):
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"answer": answer})
}

func (h *AnswerHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id"})
		return
	}
	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canManageAll := authorization.RoleHasPermission(role, authorization.PermissionManageAllContent)
	if err := h.service.Delete(uint(id), userID, canManageAll); err != nil {
		switch {
		case errors.Is(err, forumservice.ErrAnswerNotFound):
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

func (h *AnswerHandler) Vote(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid answer id"})
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
		case errors.Is(err, forumservice.ErrAnswerNotFound):
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
