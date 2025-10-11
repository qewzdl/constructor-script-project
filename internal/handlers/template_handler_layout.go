package handlers

import (
	"bytes"
	"html/template"
	"net/http"

	"constructor-script-backend/pkg/logger"

	"github.com/gin-gonic/gin"
)

func (h *TemplateHandler) basePageData(title, description string, extra gin.H) gin.H {
	data := gin.H{
		"Title":       title + " - " + h.config.SiteName,
		"Description": description,
		"Site": gin.H{
			"Name":        h.config.SiteName,
			"Description": h.config.SiteDescription,
			"URL":         h.config.SiteURL,
			"Favicon":     h.config.SiteFavicon,
		},
	}

	for k, v := range extra {
		data[k] = v
	}

	return data
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
