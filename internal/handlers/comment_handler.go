package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
	authService    *service.AuthService
	guard          *CommentGuard
}

func NewCommentHandler(commentService *service.CommentService, authService *service.AuthService, guard *CommentGuard) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
		authService:    authService,
		guard:          guard,
	}
}

// SetService updates the comment service reference.
func (h *CommentHandler) SetService(commentService *service.CommentService) {
	if h == nil {
		return
	}
	h.commentService = commentService
}

func (h *CommentHandler) ensureService(c *gin.Context) bool {
	if h == nil || h.commentService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "posts plugin is not active"})
		return false
	}
	return true
}

func (h *CommentHandler) Create(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	var req models.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")

	if h.authService != nil && h.guard != nil {
		user, err := h.authService.GetUserByID(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "user account not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to verify commenting permissions"})
			}
			return
		}

		decision := h.guard.Evaluate(user, req.Content)
		if decision.Err != nil {
			switch {
			case errors.Is(decision.Err, ErrCommentContentInvalid):
				c.JSON(http.StatusBadRequest, gin.H{"error": decision.Err.Error()})
			case errors.Is(decision.Err, ErrCommentRateLimited):
				payload := gin.H{"error": decision.Err.Error()}
				if decision.RetryAfter > 0 {
					payload["retry_after_seconds"] = int(math.Ceil(decision.RetryAfter.Seconds()))
				}
				c.JSON(http.StatusTooManyRequests, payload)
			default:
				c.JSON(http.StatusForbidden, gin.H{"error": decision.Err.Error()})
			}
			return
		}
	}

	comment, err := h.commentService.Create(uint(postID), userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"comment": comment})
}

func (h *CommentHandler) GetByPostID(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	postID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid post id"})
		return
	}

	comments, err := h.commentService.GetByPostID(uint(postID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (h *CommentHandler) GetAll(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	comments, err := h.commentService.GetAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"comments": comments})
}

func (h *CommentHandler) Update(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	var req models.UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canModerate := authorization.RoleHasPermission(role, authorization.PermissionModerateComments)

	comment, err := h.commentService.Update(uint(id), userID, canModerate, req)
	if err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"comment": comment})
}

func (h *CommentHandler) Delete(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	userID := c.GetUint("user_id")
	roleValue, _ := c.Get("role")
	role, _ := authorization.ParseUserRole(roleValue)
	canModerate := authorization.RoleHasPermission(role, authorization.PermissionModerateComments)

	if err := h.commentService.Delete(uint(id), userID, canModerate); err != nil {
		if err.Error() == "unauthorized" {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment deleted successfully"})
}

func (h *CommentHandler) ApproveComment(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	if err := h.commentService.ApproveComment(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment approved"})
}

func (h *CommentHandler) RejectComment(c *gin.Context) {
	if !h.ensureService(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid comment id"})
		return
	}

	if err := h.commentService.RejectComment(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "comment rejected"})
}
