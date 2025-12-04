package handlers

import (
	"errors"
	"net/http"
	"strings"

	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/validator"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadService *service.UploadService
}

func NewUploadHandler(uploadService *service.UploadService) *UploadHandler {
	return &UploadHandler{uploadService: uploadService}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		file, err = c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
			return
		}
	}

	// Validate Content-Type header
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing Content-Type header"})
		return
	}

	// Validate file content against allowed types
	preferredName := strings.TrimSpace(c.PostForm("name"))

	upload, err := h.uploadService.Upload(file, preferredName)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnsupportedUpload),
			errors.Is(err, service.ErrUploadTooLarge),
			errors.Is(err, service.ErrUploadMissing):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"upload":   upload,
		"url":      upload.URL,
		"filename": upload.Filename,
		"size":     upload.Size,
	})
}

func (h *UploadHandler) UploadMultiple(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse form"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}

	// Validate Content-Type headers for all files
	for _, file := range files {
		contentType := file.Header.Get("Content-Type")
		if contentType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing Content-Type header for one or more files"})
			return
		}

		// Validate it's an image MIME type
		if !validator.ValidateImageContentType(contentType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Content-Type header - images only"})
			return
		}
	}

	urls, err := h.uploadService.UploadMultipleImages(files)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"urls":  urls,
		"count": len(urls),
	})
}

func (h *UploadHandler) List(c *gin.Context) {

	uploads, err := h.uploadService.ListUploads()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"uploads": uploads})
}

func (h *UploadHandler) Rename(c *gin.Context) {
	var request struct {
		Current string `json:"current"`
		Name    string `json:"name"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	upload, err := h.uploadService.RenameUpload(request.Current, request.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidUploadName):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrUploadNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"upload": upload})
}

func (h *UploadHandler) Delete(c *gin.Context) {
	var request struct {
		Target string `json:"target"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}

	target := strings.TrimSpace(request.Target)
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target is required"})
		return
	}

	if err := h.uploadService.DeleteUpload(target); err != nil {
		switch {
		case errors.Is(err, service.ErrUploadNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
