package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	courseservice "constructor-script-backend/plugins/courses/service"
)

// AssetHandler serves protected course assets using user-bound signed tokens.
type AssetHandler struct {
	packageService *courseservice.PackageService
	protection     *courseservice.MaterialProtection
	uploadDir      string
}

// NewAssetHandler constructs an asset handler instance.
func NewAssetHandler(packageService *courseservice.PackageService, protection *courseservice.MaterialProtection, uploadDir string) *AssetHandler {
	return &AssetHandler{
		packageService: packageService,
		protection:     protection,
		uploadDir:      uploadDir,
	}
}

// SetDependencies updates handler dependencies.
func (h *AssetHandler) SetDependencies(packageService *courseservice.PackageService, protection *courseservice.MaterialProtection, uploadDir string) {
	if h == nil {
		return
	}
	h.packageService = packageService
	h.protection = protection
	h.uploadDir = uploadDir
}

// Serve streams a protected course asset after validating the signed token and user access.
func (h *AssetHandler) Serve(c *gin.Context) {
	if h == nil || h.packageService == nil || h.protection == nil || !h.protection.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "course asset protection unavailable"})
		return
	}

	token := strings.TrimSpace(c.Param("token"))
	claims, err := h.protection.ParseToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired course asset link"})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 || claims.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	course, err := h.packageService.GetForUser(claims.PackageID, userID)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify course access"})
		}
		return
	}
	if course == nil || course.Package.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "course not found"})
		return
	}

	video := findVideoInCourse(course, claims.VideoID)
	if video == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "video not found"})
		return
	}

	var (
		targetURL    string
		downloadName string
	)

	switch strings.ToLower(strings.TrimSpace(claims.Type)) {
	case courseservice.AssetTypeVideo:
		targetURL = video.FileURL
	case courseservice.AssetTypeAttachment:
		if claims.AttachmentIndex == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "attachment reference missing"})
			return
		}
		idx := *claims.AttachmentIndex
		if idx < 0 || idx >= len(video.Attachments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
			return
		}
		attachment := video.Attachments[idx]
		targetURL = attachment.URL
		downloadName = strings.TrimSpace(attachment.Title)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported asset type"})
		return
	}

	filename := h.uploadFilename(targetURL)
	if filename == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset unavailable"})
		return
	}

	filePath, err := h.resolveFilePath(filename)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "asset unavailable"})
		return
	}

	c.Header("Cache-Control", "private, no-store")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	if strings.ToLower(strings.TrimSpace(claims.Type)) == courseservice.AssetTypeAttachment {
		disposition := sanitizeDispositionName(downloadName)
		if disposition == "" {
			disposition = filename
		}
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", disposition))
	}

	c.File(filePath)
}

func (h *AssetHandler) uploadFilename(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Path != "" {
		trimmed = parsed.Path
	}

	trimmed = strings.SplitN(trimmed, "?", 2)[0]
	trimmed = strings.SplitN(trimmed, "#", 2)[0]

	if !strings.HasPrefix(trimmed, "/uploads/") {
		return ""
	}

	filename := filepath.Base(trimmed)
	if filename == "" || filename == "." || filename == ".." {
		return ""
	}

	return filename
}

func (h *AssetHandler) resolveFilePath(filename string) (string, error) {
	if strings.TrimSpace(filename) == "" {
		return "", errors.New("filename is required")
	}
	uploadDir := strings.TrimSpace(h.uploadDir)
	if uploadDir == "" {
		uploadDir = "./uploads"
	}

	targetPath := filepath.Join(uploadDir, filename)
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}

	absRoot, err := filepath.Abs(uploadDir)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(absTarget, absRoot) {
		return "", errors.New("invalid asset path")
	}

	if _, err := os.Stat(absTarget); err != nil {
		return "", err
	}

	return absTarget, nil
}

func findVideoInCourse(course *models.UserCoursePackage, videoID uint) *models.CourseVideo {
	if course == nil || videoID == 0 {
		return nil
	}

	for topicIndex := range course.Package.Topics {
		topic := &course.Package.Topics[topicIndex]
		for stepIndex := range topic.Steps {
			step := &topic.Steps[stepIndex]
			if step.Video != nil && step.Video.ID == videoID {
				return step.Video
			}
		}
	}

	return nil
}

func sanitizeDispositionName(name string) string {
	if strings.TrimSpace(name) == "" {
		return ""
	}
	normalized := strings.TrimSpace(name)
	normalized = strings.ReplaceAll(normalized, "\"", "'")
	normalized = strings.ReplaceAll(normalized, "\\", "-")
	return normalized
}
