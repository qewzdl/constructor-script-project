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
			"Favicon":     site.Favicon,
			"FaviconType": site.FaviconType,
			"Logo":        site.Logo,
		},
		"SearchQuery": "",
		"SearchType":  "all",
	}

	if len(h.navigation) > 0 {
		data["Navigation"] = h.navigation
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
		FaviconType: models.DetectFaviconType(h.config.SiteFavicon),
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

	if settings.FaviconType == "" {
		settings.FaviconType = models.DetectFaviconType(settings.Favicon)
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

	contentTmpl := h.templates.Lookup(content)
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
	var siteData gin.H
	if s, ok := data["Site"].(gin.H); ok {
		siteData = s
		if urlStr, ok := siteData["URL"].(string); ok {
			siteURL = urlStr
		}
	}
	if siteURL == "" {
		siteURL = h.config.SiteURL
	}

	if normalized := h.normalizeBaseURL(siteURL, c.Request); normalized != "" {
		siteURL = normalized
	}

	if siteData != nil {
		siteData["URL"] = siteURL
		siteData["Favicon"] = h.resolveAbsoluteURL(siteURL, getString(siteData, "Favicon"), c.Request)
		siteData["Logo"] = h.resolveAbsoluteURL(siteURL, getString(siteData, "Logo"), c.Request)
	}

	title, _ := data["Title"].(string)
	description, _ := data["Description"].(string)

	canonical := strings.TrimSpace(getString(data, "Canonical"))
	if canonical == "" {
		canonical = h.buildCanonicalURL(siteURL, c.Request.URL)
	} else {
		canonical = h.resolveAbsoluteURL(siteURL, canonical, c.Request)
	}
	data["Canonical"] = canonical

	ogURL := strings.TrimSpace(getString(data, "OGURL"))
	if ogURL == "" {
		ogURL = canonical
	} else {
		ogURL = h.resolveAbsoluteURL(siteURL, ogURL, c.Request)
	}
	data["OGURL"] = ogURL

	ogType := strings.TrimSpace(getString(data, "OGType"))
	if ogType == "" {
		ogType = "website"
	}
	data["OGType"] = ogType

	ogImage := strings.TrimSpace(getString(data, "OGImage"))
	if ogImage != "" {
		ogImage = h.resolveAbsoluteURL(siteURL, ogImage, c.Request)
	} else if siteData != nil {
		if logo := strings.TrimSpace(getString(siteData, "Logo")); logo != "" {
			ogImage = h.resolveAbsoluteURL(siteURL, logo, c.Request)
		}
	}
	data["OGImage"] = ogImage

	if strings.TrimSpace(getString(data, "OGImageAlt")) == "" {
		data["OGImageAlt"] = title
	}

	twitterImage := strings.TrimSpace(getString(data, "TwitterImage"))
	if twitterImage != "" {
		twitterImage = h.resolveAbsoluteURL(siteURL, twitterImage, c.Request)
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

func (h *TemplateHandler) resolveAbsoluteURL(baseURL, value string, r *http.Request) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if strings.HasPrefix(value, "//") {
		scheme := requestScheme(r)
		if scheme == "" {
			return value
		}
		return scheme + ":" + value
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return value
		}

		if r != nil {
			reqHost := requestHost(r)
			if reqHost != "" && parsed.Host != "" && strings.EqualFold(parsed.Host, reqHost) {
				scheme := requestScheme(r)
				if scheme != "" && parsed.Scheme != scheme {
					parsed.Scheme = scheme
					return parsed.String()
				}
			}
		}

		return value
	}

	base := strings.TrimSpace(baseURL)
	if base == "" {
		base = strings.TrimSpace(h.config.SiteURL)
	}

	if base != "" {
		base = strings.TrimSuffix(base, "/")
		if !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
		return base + value
	}

	if r != nil {
		scheme := requestScheme(r)
		host := requestHost(r)
		if scheme != "" && host != "" {
			if !strings.HasPrefix(value, "/") {
				value = "/" + value
			}
			return scheme + "://" + host + value
		}
	}

	return value
}

func (h *TemplateHandler) normalizeBaseURL(baseURL string, r *http.Request) string {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" || r == nil {
		return baseURL
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}

	reqHost := requestHost(r)
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, reqHost) {
		return baseURL
	}

	scheme := requestScheme(r)
	if scheme == "" {
		return baseURL
	}

	if parsed.Scheme != "" && parsed.Scheme != scheme {
		parsed.Scheme = scheme
		return parsed.String()
	}

	if parsed.Scheme == "" {
		parsed.Scheme = scheme
		return parsed.String()
	}

	return baseURL
}

func requestScheme(r *http.Request) string {
	if r == nil {
		return ""
	}

	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		parts := strings.Split(proto, ",")
		if len(parts) > 0 {
			value := strings.ToLower(strings.TrimSpace(parts[0]))
			if value != "" {
				return value
			}
		}
	}

	if r.TLS != nil {
		return "https"
	}

	if r.URL != nil && r.URL.Scheme != "" {
		return strings.ToLower(r.URL.Scheme)
	}

	return "http"
}

func requestHost(r *http.Request) string {
	if r == nil {
		return ""
	}

	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		parts := strings.Split(forwardedHost, ",")
		if len(parts) > 0 {
			host := strings.TrimSpace(parts[0])
			if host != "" {
				return host
			}
		}
	}

	if r.Host != "" {
		return r.Host
	}

	if r.URL != nil {
		return r.URL.Host
	}

	return ""
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
