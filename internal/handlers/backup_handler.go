package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

type BackupHandler struct {
	service *service.BackupService
}

func NewBackupHandler(service *service.BackupService) *BackupHandler {
	return &BackupHandler{service: service}
}

func (h *BackupHandler) Export(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Backup service not available"})
		return
	}

	archive, err := h.service.CreateArchive(c.Request.Context())
	if err != nil {
		logger.Error(err, "Failed to create backup archive", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup archive"})
		return
	}
	defer archive.Close()

	if err := archive.Reset(); err != nil {
		logger.Error(err, "Failed to prepare backup archive for download", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare backup archive"})
		return
	}

	file := archive.File()
	if file == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Backup archive is unavailable"})
		return
	}

	summary := archive.Summary
	counts := []string{
		fmt.Sprintf("users=%d", summary.Users),
		fmt.Sprintf("categories=%d", summary.Categories),
		fmt.Sprintf("tags=%d", summary.Tags),
		fmt.Sprintf("posts=%d", summary.Posts),
		fmt.Sprintf("pages=%d", summary.Pages),
		fmt.Sprintf("comments=%d", summary.Comments),
		fmt.Sprintf("settings=%d", summary.Settings),
		fmt.Sprintf("menu_items=%d", summary.MenuItems),
		fmt.Sprintf("social_links=%d", summary.SocialLinks),
		fmt.Sprintf("post_tags=%d", summary.PostTags),
		fmt.Sprintf("uploads=%d", summary.Uploads),
	}

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", archive.Filename))
	c.Header("X-Backup-Schema", summary.SchemaVersion)
	c.Header("X-Backup-Generated-At", summary.GeneratedAt.UTC().Format(time.RFC3339Nano))
	c.Header("X-Backup-Application", summary.Application)
	c.Header("X-Backup-Counts", strings.Join(counts, ";"))
	if size, err := archive.Size(); err == nil {
		c.Header("X-Backup-Size", strconv.FormatInt(size, 10))
	}

	http.ServeContent(c.Writer, c.Request, archive.Filename, summary.GeneratedAt, file)
}

func (h *BackupHandler) Import(c *gin.Context) {
	if h == nil || h.service == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Backup service not available"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Backup file is required"})
		return
	}

	uploaded, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to open uploaded file"})
		return
	}
	defer uploaded.Close()

	summary, restoreErr := h.service.RestoreArchive(c.Request.Context(), uploaded, fileHeader.Size)
	if restoreErr != nil {
		status := http.StatusInternalServerError
		if errors.Is(restoreErr, service.ErrInvalidBackup) || errors.Is(restoreErr, service.ErrBackupVersion) {
			status = http.StatusBadRequest
		}
		logger.Error(restoreErr, "Failed to restore backup", nil)
		c.JSON(status, gin.H{"error": restoreErr.Error()})
		return
	}

	responseSummary := gin.H{
		"schema_version": summary.SchemaVersion,
		"generated_at":   summary.GeneratedAt,
		"restored_at":    summary.RestoredAt,
		"application":    summary.Application,
		"users":          summary.Users,
		"categories":     summary.Categories,
		"tags":           summary.Tags,
		"posts":          summary.Posts,
		"pages":          summary.Pages,
		"comments":       summary.Comments,
		"settings":       summary.Settings,
		"menu_items":     summary.MenuItems,
		"social_links":   summary.SocialLinks,
		"post_tags":      summary.PostTags,
		"uploads":        summary.Uploads,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Backup restored successfully",
		"summary": responseSummary,
	})
}
