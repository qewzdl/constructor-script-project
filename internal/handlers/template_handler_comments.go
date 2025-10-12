package handlers

import (
	"html/template"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
)

type CommentView struct {
	ID         uint
	AuthorID   uint
	AuthorName string
	CreatedAt  time.Time
	Content    template.HTML
	RawContent string
	Replies    []CommentView
}

func (h *TemplateHandler) buildCommentViews(comments []models.Comment) []CommentView {
	if len(comments) == 0 {
		return nil
	}

	views := make([]CommentView, 0, len(comments))
	for i := range comments {
		comment := &comments[i]
		if !comment.Approved {
			continue
		}
		views = append(views, h.buildCommentView(comment))
	}

	return views
}

func (h *TemplateHandler) buildCommentView(comment *models.Comment) CommentView {
	authorName := "Anonymous"
	if comment.Author.Username != "" {
		authorName = comment.Author.Username
	}

	view := CommentView{
		ID:         comment.ID,
		AuthorID:   comment.AuthorID,
		AuthorName: authorName,
		CreatedAt:  comment.CreatedAt,
		Content:    h.sanitizeCommentContent(comment.Content),
		RawContent: comment.Content,
	}

	if len(comment.Replies) > 0 {
		replies := make([]CommentView, 0, len(comment.Replies))
		for _, reply := range comment.Replies {
			if reply == nil || !reply.Approved {
				continue
			}
			replies = append(replies, h.buildCommentView(reply))
		}
		view.Replies = replies
	}

	return view
}

func (h *TemplateHandler) countComments(comments []models.Comment) int {
	total := 0
	for i := range comments {
		comment := &comments[i]
		if !comment.Approved {
			continue
		}
		total++
		total += h.countCommentReplies(comment.Replies)
	}
	return total
}

func (h *TemplateHandler) countCommentReplies(replies []*models.Comment) int {
	total := 0
	for _, reply := range replies {
		if reply == nil || !reply.Approved {
			continue
		}
		total++
		total += h.countCommentReplies(reply.Replies)
	}
	return total
}

func (h *TemplateHandler) sanitizeCommentContent(content string) template.HTML {
	if content == "" {
		return ""
	}

	if h.sanitizer == nil {
		escaped := template.HTMLEscapeString(content)
		escaped = strings.ReplaceAll(escaped, "\n", "<br />")
		return template.HTML(escaped)
	}

	sanitized := h.sanitizer.Sanitize(content)
	sanitized = strings.ReplaceAll(sanitized, "\n", "<br />")
	return template.HTML(sanitized)
}
