package handlers

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/logger"
	blogservice "constructor-script-backend/plugins/blog/service"
	languageservice "constructor-script-backend/plugins/language/service"

	"github.com/gin-gonic/gin"
)

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod,omitempty"`
	ChangeFreq string `xml:"changefreq,omitempty"`
	Priority   string `xml:"priority,omitempty"`
}

type sitemapURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// SEOHandler provides responses for SEO-focused endpoints like sitemap.xml and
// robots.txt.
type SEOHandler struct {
	postService     *blogservice.PostService
	pageService     *service.PageService
	categoryService *blogservice.CategoryService
	setupService    *service.SetupService
	languageService *languageservice.LanguageService
	config          *config.Config
}

// NewSEOHandler creates a new SEO handler with the required dependencies.
func NewSEOHandler(
	postService *blogservice.PostService,
	pageService *service.PageService,
	categoryService *blogservice.CategoryService,
	setupService *service.SetupService,
	languageService *languageservice.LanguageService,
	cfg *config.Config,
) *SEOHandler {
	return &SEOHandler{
		postService:     postService,
		pageService:     pageService,
		categoryService: categoryService,
		setupService:    setupService,
		languageService: languageService,
		config:          cfg,
	}
}

// SetBlogServices updates the services backing post and category lookups.
func (h *SEOHandler) SetBlogServices(postService *blogservice.PostService, categoryService *blogservice.CategoryService) {
	if h == nil {
		return
	}
	h.postService = postService
	h.categoryService = categoryService
}

// SetLanguageService updates the language service dependency used by the SEO handler.
func (h *SEOHandler) SetLanguageService(languageService *languageservice.LanguageService) {
	if h == nil {
		return
	}
	h.languageService = languageService
}

// Sitemap renders an XML sitemap that includes the key public sections of the
// site along with all published posts, pages, categories and tags.
func (h *SEOHandler) Sitemap(c *gin.Context) {
	if h.postService == nil || h.categoryService == nil {
		c.String(http.StatusServiceUnavailable, "Posts plugin is not active")
		return
	}

	siteSettings, err := ResolveSiteSettings(h.config, h.setupService, h.languageService)
	if err != nil {
		logger.Error(err, "Failed to resolve site settings", nil)
	}

	baseURL := h.normalizedBaseURL(siteSettings.URL)
	if baseURL == "" {
		c.String(http.StatusInternalServerError, "Unable to determine site URL")
		return
	}

	posts, err := h.postService.ListPublishedForSitemap()
	if err != nil {
		logger.Error(err, "Failed to load posts for sitemap", nil)
		c.String(http.StatusInternalServerError, "Failed to build sitemap")
		return
	}

	pages, err := h.pageService.GetAll()
	if err != nil {
		logger.Error(err, "Failed to load pages for sitemap", nil)
		c.String(http.StatusInternalServerError, "Failed to build sitemap")
		return
	}

	categories, err := h.categoryService.GetAll()
	if err != nil {
		logger.Error(err, "Failed to load categories for sitemap", nil)
		c.String(http.StatusInternalServerError, "Failed to build sitemap")
		return
	}

	tags, err := h.postService.GetTagsInUse()
	if err != nil {
		logger.Error(err, "Failed to load tags for sitemap", nil)
		c.String(http.StatusInternalServerError, "Failed to build sitemap")
		return
	}

	urls := []sitemapURL{
		{Loc: baseURL + "/", ChangeFreq: "daily", Priority: "1.0"},
		{Loc: h.joinURL(baseURL, "/blog"), ChangeFreq: "daily", Priority: "0.8"},
	}

	for _, post := range posts {
		loc := h.joinURL(baseURL, h.postPath(post))
		lastMod := post.UpdatedAt
		if lastMod.IsZero() {
			lastMod = post.CreatedAt
		}

		urls = append(urls, sitemapURL{
			Loc:        loc,
			LastMod:    h.formatLastMod(lastMod),
			ChangeFreq: "weekly",
			Priority:   "0.7",
		})
	}

	for _, page := range pages {
		if page.Slug == "" && strings.TrimSpace(page.Path) == "" {
			continue
		}

		path := strings.TrimSpace(page.Path)
		if path == "" {
			path = fmt.Sprintf("/page/%s", page.Slug)
		}

		urls = append(urls, sitemapURL{
			Loc:        h.joinURL(baseURL, path),
			LastMod:    h.formatLastMod(page.UpdatedAt),
			ChangeFreq: "monthly",
			Priority:   "0.6",
		})
	}

	for _, category := range categories {
		if category.Slug == "" {
			continue
		}

		urls = append(urls, sitemapURL{
			Loc:        h.joinURL(baseURL, fmt.Sprintf("/category/%s", category.Slug)),
			ChangeFreq: "weekly",
			Priority:   "0.5",
		})
	}

	for _, tag := range tags {
		if tag.Slug == "" {
			continue
		}

		urls = append(urls, sitemapURL{
			Loc:        h.joinURL(baseURL, fmt.Sprintf("/tag/%s", tag.Slug)),
			ChangeFreq: "weekly",
			Priority:   "0.4",
		})
	}

	response := sitemapURLSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.XML(http.StatusOK, response)
}

// Robots renders a robots.txt file that guides crawlers and references the
// generated sitemap.
func (h *SEOHandler) Robots(c *gin.Context) {
	siteSettings, err := ResolveSiteSettings(h.config, h.setupService, h.languageService)
	if err != nil {
		logger.Error(err, "Failed to resolve site settings", nil)
	}

	baseURL := h.normalizedBaseURL(siteSettings.URL)
	sitemapURL := ""
	if baseURL != "" {
		sitemapURL = h.joinURL(baseURL, "/sitemap.xml")
	}

	lines := []string{
		"User-agent: *",
		"Allow: /",
		"Disallow: /admin",
		"Disallow: /profile",
		"Disallow: /api/",
	}

	if sitemapURL != "" {
		lines = append(lines, fmt.Sprintf("Sitemap: %s", sitemapURL))
	}

	body := strings.Join(lines, "\n") + "\n"

	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

func (h *SEOHandler) normalizedBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = strings.TrimSpace(h.config.SiteURL)
	}

	trimmed = strings.TrimSuffix(trimmed, "/")
	return trimmed
}

func (h *SEOHandler) joinURL(base, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return base + path
}

func (h *SEOHandler) postPath(post models.Post) string {
	if post.Slug != "" {
		return fmt.Sprintf("/blog/post/%s", post.Slug)
	}
	return fmt.Sprintf("/blog/post/%d", post.ID)
}

func (h *SEOHandler) formatLastMod(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
