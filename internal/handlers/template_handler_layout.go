package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"

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
			"Favicon":     h.ensureAbsoluteURL(site.URL, site.Favicon),
			"Logo":        h.ensureAbsoluteURL(site.URL, site.Logo),
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
	h.applySEOMetadata(c, data)

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

func (h *TemplateHandler) applySEOMetadata(c *gin.Context, data gin.H) {
	siteURL := ""
	if siteData, ok := data["Site"].(gin.H); ok {
		if urlStr, ok := siteData["URL"].(string); ok {
			siteURL = urlStr
		}
	}
	if siteURL == "" {
		siteURL = h.config.SiteURL
	}

	title, _ := data["Title"].(string)
	description, _ := data["Description"].(string)

	canonical := strings.TrimSpace(getString(data, "Canonical"))
	if canonical == "" {
		canonical = h.buildCanonicalURL(siteURL, c.Request.URL)
	} else {
		canonical = h.ensureAbsoluteURL(siteURL, canonical)
	}
	data["Canonical"] = canonical

	ogURL := strings.TrimSpace(getString(data, "OGURL"))
	if ogURL == "" {
		ogURL = canonical
	} else {
		ogURL = h.ensureAbsoluteURL(siteURL, ogURL)
	}
	data["OGURL"] = ogURL

	ogType := strings.TrimSpace(getString(data, "OGType"))
	if ogType == "" {
		ogType = "website"
	}
	data["OGType"] = ogType

	ogImage := strings.TrimSpace(getString(data, "OGImage"))
	if ogImage != "" {
		ogImage = h.ensureAbsoluteURL(siteURL, ogImage)
	} else if siteData, ok := data["Site"].(gin.H); ok {
		if logo, ok := siteData["Logo"].(string); ok && logo != "" {
			ogImage = h.ensureAbsoluteURL(siteURL, logo)
		}
	}
	data["OGImage"] = ogImage

	if strings.TrimSpace(getString(data, "OGImageAlt")) == "" {
		data["OGImageAlt"] = title
	}

	twitterImage := strings.TrimSpace(getString(data, "TwitterImage"))
	if twitterImage != "" {
		twitterImage = h.ensureAbsoluteURL(siteURL, twitterImage)
	} else if ogImage != "" {
		twitterImage = ogImage
	}
	data["TwitterImage"] = twitterImage

	if strings.TrimSpace(getString(data, "TwitterImageAlt")) == "" {
		if alt := strings.TrimSpace(getString(data, "OGImageAlt")); alt != "" {
			data["TwitterImageAlt"] = alt
		} else {
			data["TwitterImageAlt"] = title
		}
	}

	if strings.TrimSpace(getString(data, "TwitterCard")) == "" {
		data["TwitterCard"] = "summary_large_image"
	}

	if strings.TrimSpace(getString(data, "TwitterTitle")) == "" {
		data["TwitterTitle"] = title
	}

	if strings.TrimSpace(getString(data, "TwitterDescription")) == "" {
		data["TwitterDescription"] = description
	}
}

func (h *TemplateHandler) buildCanonicalURL(base string, requestURL *url.URL) string {
	if requestURL == nil {
		return strings.TrimSuffix(base, "/")
	}

	cleaned := *requestURL
	cleaned.Fragment = ""

	if rawQuery := cleaned.Query(); len(rawQuery) > 0 {
		for key := range rawQuery {
			lower := strings.ToLower(key)
			if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
				rawQuery.Del(key)
			}
		}
		cleaned.RawQuery = rawQuery.Encode()
	}

	if cleaned.IsAbs() {
		return cleaned.String()
	}

	base = strings.TrimSuffix(base, "/")
	path := cleaned.Path
	if path == "" {
		path = "/"
	} else if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	canonical := path
	if cleaned.RawQuery != "" {
		canonical = canonical + "?" + cleaned.RawQuery
	}

	if base == "" {
		return canonical
	}

	return base + canonical
}

func (h *TemplateHandler) ensureAbsoluteURL(baseURL, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}

	if strings.HasPrefix(value, "//") {
		return value
	}

	if baseURL == "" {
		baseURL = h.config.SiteURL
	}

	if baseURL == "" {
		return value
	}

	baseURL = strings.TrimSuffix(baseURL, "/")
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}

	return baseURL + value
}

func getString(data gin.H, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}

func (h *TemplateHandler) executeTemplate(tmpl *template.Template, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
