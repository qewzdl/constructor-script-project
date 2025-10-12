package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (h *TemplateHandler) basePageData(title, description string, extra gin.H) gin.H {
	site := h.siteSettings()

	data := gin.H{
		"Title":       fmt.Sprintf("%s - %s", title, site.Name),
		"Description": description,
		"Site": gin.H{
			"Name":        site.Name,
			"Description": site.Description,
			"URL":         site.URL,
			"Favicon":     site.Favicon,
			"Logo":        site.Logo,
		},
		"SearchQuery": "",
		"SearchType":  "all",
	}

	for k, v := range extra {
		data[k] = v
	}

	return data
}

func (h *TemplateHandler) siteSettings() models.SiteSettings {
	defaults := models.SiteSettings{
		Name:        h.config.SiteName,
		Description: h.config.SiteDescription,
		URL:         h.config.SiteURL,
		Favicon:     h.config.SiteFavicon,
		Logo:        "/static/icons/logo.svg",
	}

	if h.setupService == nil {
		return defaults
	}

	settings, err := h.setupService.GetSiteSettings(defaults)
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
		return defaults
	}

	return settings
}

func (h *TemplateHandler) renderTemplate(c *gin.Context, templateName, title, description string, extra gin.H) {
	data := h.basePageData(title, description, extra)
	if templateName == "" {
		templateName = "page"
	}
	h.renderWithLayout(c, "base.html", templateName+".html", data)
}

func (h *TemplateHandler) renderWithLayout(c *gin.Context, layout, content string, data gin.H) {
	h.addUserContext(c, data)

	if noIndex, ok := data["NoIndex"].(bool); ok && noIndex {
		c.Header("X-Robots-Tag", "noindex, nofollow")
	}

	tmpl, err := h.templates.Clone()
	if err != nil {
		logger.Error(err, "Failed to clone templates", nil)
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Template error")
		return
	}

	contentTmpl := tmpl.Lookup(content)
	if contentTmpl == nil {
		logger.Error(nil, "Content template not found", map[string]interface{}{"template": content})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Template not found")
		return
	}

	buf, err := h.executeTemplate(contentTmpl, data)
	if err != nil {
		logger.Error(err, "Failed to render content", map[string]interface{}{"template": content})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to render content")
		return
	}

	data["Content"] = template.HTML(buf)

	c.HTML(http.StatusOK, layout, data)
}

func (h *TemplateHandler) executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
