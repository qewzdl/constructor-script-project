package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"unicode"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/payments/stripe"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/lang"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type FooterMenuGroup struct {
	Key   string
	Title string
	Items []models.MenuItem
}

type advertisingTemplateData struct {
	Enabled      bool
	HeadSnippets []template.HTML
	Placements   map[string][]template.HTML
}

func (d advertisingTemplateData) hasPlacements() bool {
	return len(d.Placements) > 0
}

type courseCheckoutTemplateData struct {
	Enabled        bool
	Endpoint       string
	PublishableKey string
}

func (h *TemplateHandler) basePageData(title, description string, extra gin.H) gin.H {
	site := h.siteSettings()

	headerMenu, footerMenu := splitMenuItems(site.MenuItems)

	advertising := h.advertisingTemplateData()

	checkoutData := courseCheckoutTemplateData{
		Enabled:  h.courseCheckoutEnabled(),
		Endpoint: "/api/v1/courses/checkout",
	}
	if key := strings.TrimSpace(site.StripePublishableKey); stripe.IsPublishableKey(key) {
		checkoutData.PublishableKey = key
	}
	if h.config != nil {
		if checkoutData.PublishableKey == "" {
			if key := strings.TrimSpace(h.config.StripePublishableKey); stripe.IsPublishableKey(key) {
				checkoutData.PublishableKey = key
			}
		}
	}

	data := gin.H{
		"Title":       fmt.Sprintf("%s - %s", title, site.Name),
		"Description": description,
		"Language":    site.DefaultLanguage,
		"Site": gin.H{
			"Name":               site.Name,
			"Description":        site.Description,
			"URL":                site.URL,
			"Favicon":            site.Favicon,
			"FaviconType":        site.FaviconType,
			"Logo":               site.Logo,
			"SocialLinks":        site.SocialLinks,
			"MenuItems":          site.MenuItems,
			"HeaderMenuItems":    headerMenu,
			"FooterMenuItems":    footerMenu,
			"DefaultLanguage":    site.DefaultLanguage,
			"SupportedLanguages": site.SupportedLanguages,
			"Fonts":              site.Fonts,
			"FontPreconnects":    site.FontPreconnects,
		},
		"SearchQuery":    "",
		"SearchType":     "all",
		"Advertising":    advertising,
		"CourseCheckout": checkoutData,
	}

	for k, v := range extra {
		data[k] = v
	}

	return data
}

func (h *TemplateHandler) siteSettings() models.SiteSettings {
	settings, err := ResolveSiteSettings(h.config, h.setupService, h.languageService)
	if err != nil {
		logger.Error(err, "Failed to load site settings", nil)
	}

	if h.socialLinkService != nil {
		links, err := h.socialLinkService.ListPublic()
		if err != nil {
			logger.Error(err, "Failed to load social links", nil)
		} else {
			settings.SocialLinks = links
		}
	}

	if h.menuService != nil {
		items, err := h.menuService.ListPublic()
		if err != nil {
			logger.Error(err, "Failed to load menu items", nil)
		} else {
			settings.MenuItems = items
		}
	}

	fonts := []models.FontAsset{}
	if h.fontService != nil {
		if list, err := h.fontService.ListActive(); err != nil {
			logger.Error(err, "Failed to load fonts", nil)
			fonts = service.DefaultFontAssets()
		} else {
			fonts = list
		}
	} else {
		fonts = service.DefaultFontAssets()
	}
	settings.Fonts = fonts
	settings.FontPreconnects = service.CollectFontPreconnects(fonts)

	if strings.TrimSpace(settings.DefaultLanguage) == "" || len(settings.SupportedLanguages) == 0 {
		if h.languageService != nil {
			fallbackDefault, fallbackSupported := h.languageService.Defaults()
			if strings.TrimSpace(settings.DefaultLanguage) == "" {
				settings.DefaultLanguage = fallbackDefault
			}
			if len(settings.SupportedLanguages) == 0 {
				settings.SupportedLanguages = append([]string(nil), fallbackSupported...)
			}
		}
	}
	if strings.TrimSpace(settings.DefaultLanguage) == "" && h.config != nil {
		settings.DefaultLanguage = h.config.DefaultLanguage
	}
	if strings.TrimSpace(settings.DefaultLanguage) == "" {
		settings.DefaultLanguage = lang.Default
	}
	if len(settings.SupportedLanguages) == 0 {
		if h.config != nil && len(h.config.SupportedLanguages) > 0 {
			settings.SupportedLanguages = append([]string(nil), h.config.SupportedLanguages...)
		} else {
			settings.SupportedLanguages = []string{settings.DefaultLanguage}
		}
	}

	return settings
}

func (h *TemplateHandler) advertisingTemplateData() advertisingTemplateData {
	result := advertisingTemplateData{
		Placements: make(map[string][]template.HTML),
	}

	if h.advertisingService == nil {
		return result
	}

	rendered, err := h.advertisingService.RenderActive()
	if err != nil {
		logger.Error(err, "Failed to render advertising", nil)
		return result
	}

	if !rendered.Enabled {
		return result
	}

	if len(rendered.HeadSnippets) > 0 {
		head := make([]template.HTML, 0, len(rendered.HeadSnippets))
		for _, snippet := range rendered.HeadSnippets {
			trimmed := strings.TrimSpace(snippet)
			if trimmed == "" {
				continue
			}
			head = append(head, template.HTML(trimmed))
		}
		if len(head) > 0 {
			result.HeadSnippets = head
		}
	}

	if len(rendered.Placements) > 0 {
		for key, snippets := range rendered.Placements {
			if len(snippets) == 0 {
				continue
			}
			cleaned := make([]template.HTML, 0, len(snippets))
			for _, snippet := range snippets {
				trimmed := strings.TrimSpace(snippet)
				if trimmed == "" {
					continue
				}
				cleaned = append(cleaned, template.HTML(trimmed))
			}
			if len(cleaned) > 0 {
				result.Placements[key] = cleaned
			}
		}
	}

	result.Enabled = len(result.HeadSnippets) > 0 || result.hasPlacements()
	return result
}

func splitMenuItems(items []models.MenuItem) ([]models.MenuItem, []FooterMenuGroup) {
	if len(items) == 0 {
		return nil, nil
	}

	var header []models.MenuItem
	groups := make(map[string]*FooterMenuGroup)

	for _, item := range items {
		location := normalizeLocation(item.Location)

		switch {
		case isFooterLocation(location):
			key := footerGroupKey(location)
			group, ok := groups[key]
			if !ok {
				group = &FooterMenuGroup{
					Key:   key,
					Title: footerGroupTitle(key),
				}
				groups[key] = group
			}
			group.Items = append(group.Items, item)
		case location == "header" || location == "":
			header = append(header, item)
		default:
			header = append(header, item)
		}
	}

	header = sortMenuItems(header)

	if len(groups) == 0 {
		return header, nil
	}

	for _, group := range groups {
		group.Items = sortMenuItems(group.Items)
	}

	orderedKeys := orderedFooterGroupKeys(groups)

	footer := make([]FooterMenuGroup, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		if group, ok := groups[key]; ok && len(group.Items) > 0 {
			footer = append(footer, *group)
		}
	}

	if len(footer) == 0 {
		return header, nil
	}

	return header, footer
}

func sortMenuItems(items []models.MenuItem) []models.MenuItem {
	if len(items) == 0 {
		return nil
	}

	sorted := make([]models.MenuItem, len(items))
	copy(sorted, items)

	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Order == sorted[j].Order {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].Order < sorted[j].Order
	})

	return sorted
}

func normalizeLocation(location string) string {
	return strings.TrimSpace(strings.ToLower(location))
}

func isFooterLocation(location string) bool {
	return strings.HasPrefix(location, "footer")
}

func footerGroupKey(location string) string {
	cleaned := normalizeLocation(location)
	if cleaned == "" {
		return "footer"
	}

	if cleaned == "footer" {
		return cleaned
	}

	suffix := strings.TrimPrefix(cleaned, "footer")
	suffix = strings.TrimLeft(suffix, ":_- ")
	if suffix == "" {
		return "footer"
	}

	return "footer:" + suffix
}

var footerMenuLabels = map[string]string{
	"footer":         "Footer",
	"footer:explore": "Explore",
	"footer:account": "Account",
	"footer:legal":   "Legal",
}

var footerMenuOrder = []string{
	"footer:explore",
	"footer:account",
	"footer:legal",
	"footer",
}

func footerGroupTitle(key string) string {
	if label, ok := footerMenuLabels[key]; ok {
		return label
	}

	if strings.HasPrefix(key, "footer:") {
		return formatFooterLabel(strings.TrimPrefix(key, "footer:"))
	}

	return "Footer"
}

func formatFooterLabel(value string) string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		switch r {
		case '-', '_', ' ', ':', '/':
			return true
		}
		return false
	})

	if len(parts) == 0 {
		return "Footer"
	}

	for i, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = unicode.ToUpper(runes[0])
		for j := 1; j < len(runes); j++ {
			runes[j] = unicode.ToLower(runes[j])
		}
		parts[i] = string(runes)
	}

	return strings.Join(parts, " ")
}

func orderedFooterGroupKeys(groups map[string]*FooterMenuGroup) []string {
	if len(groups) == 0 {
		return nil
	}

	seen := make(map[string]bool, len(groups))
	ordered := make([]string, 0, len(groups))

	for _, key := range footerMenuOrder {
		if _, ok := groups[key]; ok {
			ordered = append(ordered, key)
			seen[key] = true
		}
	}

	var extras []string
	for key := range groups {
		if seen[key] {
			continue
		}
		extras = append(extras, key)
	}

	sort.SliceStable(extras, func(i, j int) bool {
		left := groups[extras[i]].Title
		right := groups[extras[j]].Title
		if left == right {
			return extras[i] < extras[j]
		}
		return left < right
	})

	ordered = append(ordered, extras...)

	return ordered
}

func (h *TemplateHandler) renderTemplate(c *gin.Context, templateName, title, description string, extra gin.H) {
	data := h.basePageData(title, description, extra)
	if templateName == "" {
		templateName = "page"
	}

	layout := "base.html"
	if value, ok := data["Layout"].(string); ok && value != "" {
		layout = value
	}

	h.renderWithLayout(c, layout, templateName+".html", data)
}

func (h *TemplateHandler) renderWithLayout(c *gin.Context, layout, content string, data gin.H) {
	h.addUserContext(c, data)
	h.applySEOMetadata(c, data)
	h.setNavigationState(c, data)

	if noIndex, ok := data["NoIndex"].(bool); ok && noIndex {
		c.Header("X-Robots-Tag", "noindex, nofollow")
	}

	tmpl, err := h.templateClone()
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

	layoutTmpl := tmpl.Lookup(layout)
	if layoutTmpl == nil {
		logger.Error(nil, "Layout template not found", map[string]interface{}{"template": layout})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Template not found")
		return
	}

	output, err := h.executeTemplate(layoutTmpl, data)
	if err != nil {
		logger.Error(err, "Failed to render layout", map[string]interface{}{"template": layout})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to render layout")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", output)
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

	if language := strings.TrimSpace(getString(data, "Language")); language == "" {
		if siteData != nil {
			language = strings.TrimSpace(getString(siteData, "DefaultLanguage"))
		}
		if language == "" {
			language = lang.Default
		}
		data["Language"] = language
	}

	if siteData != nil {
		if locale := strings.TrimSpace(getString(siteData, "DefaultLanguage")); locale != "" {
			data["OGLocale"] = locale
		}
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

func (h *TemplateHandler) setNavigationState(c *gin.Context, data gin.H) {
	path := c.Request.URL.Path
	cleanedPath := utils.NormalizePath(path)
	data["ActivePath"] = cleanedPath

	if _, exists := data["ActiveNav"]; exists {
		return
	}

	active := ""

	switch {
	case cleanedPath == "/" || cleanedPath == "":
		active = "home"
	case strings.HasPrefix(cleanedPath, "/blog"):
		active = "blog"
	case strings.HasPrefix(cleanedPath, "/search"):
		active = "search"
	case strings.HasPrefix(cleanedPath, "/admin"):
		active = "admin"
	case strings.HasPrefix(cleanedPath, "/profile"):
		active = "profile"
	case strings.HasPrefix(cleanedPath, "/login"):
		active = "login"
	case strings.HasPrefix(cleanedPath, "/register"):
		active = "register"
	}

	data["ActiveNav"] = active
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
