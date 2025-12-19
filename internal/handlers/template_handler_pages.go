package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/logger"
	archiveservice "constructor-script-backend/plugins/archive/service"
	blogservice "constructor-script-backend/plugins/blog/service"
	courseservice "constructor-script-backend/plugins/courses/service"
	forumservice "constructor-script-backend/plugins/forum/service"
)

func (h *TemplateHandler) renderSinglePost(c *gin.Context, post *models.Post) {
	var related []models.Post
	if h.postService != nil {
		related, _ = h.postService.GetRelatedPosts(post.ID, 3)
	}

	var (
		comments     []CommentView
		commentCount int
	)

	if h.commentService != nil {
		if loaded, err := h.commentService.GetByPostID(post.ID); err != nil {
			logger.Error(err, "Failed to load comments for post", map[string]interface{}{"post_id": post.ID})
		} else {
			comments = h.buildCommentViews(loaded)
			commentCount = h.countComments(loaded)
		}
	}

	site := h.siteSettings()
	canonicalPath := fmt.Sprintf("/blog/post/%s", post.Slug)
	if post.Slug == "" {
		canonicalPath = fmt.Sprintf("/blog/post/%d", post.ID)
	}
	canonicalURL := h.ensureAbsoluteURL(site.URL, canonicalPath)
	if canonicalURL == "" {
		canonicalURL = h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath)
	}

	var keywords []string
	if post.Category.Name != "" {
		keywords = append(keywords, post.Category.Name)
	}
	for _, tag := range post.Tags {
		if tag.Name != "" {
			keywords = append(keywords, tag.Name)
		}
	}

	structuredData := h.buildPostStructuredData(post, site, canonicalURL)

	contentHTML, sectionScripts := h.renderSections(post.Sections)
	scripts := appendScripts([]string{"/static/js/post.js"}, sectionScripts)

	data := h.basePageData(post.Title, post.Description, gin.H{
		"Post":           post,
		"RelatedPosts":   related,
		"Content":        contentHTML,
		"TOC":            h.generateTOC(post.Sections),
		"Comments":       comments,
		"CommentCount":   commentCount,
		"Canonical":      canonicalURL,
		"OGType":         "article",
		"OGImage":        post.FeaturedImg,
		"TwitterImage":   post.FeaturedImg,
		"StructuredData": structuredData,
		"Scripts":        scripts,
	})

	if len(keywords) > 0 {
		data["Keywords"] = strings.Join(keywords, ", ")
	}

	templateName := post.Template
	if templateName == "" {
		templateName = "post"
	}

	h.renderWithLayout(c, "base.html", templateName+".html", data)
}

func (h *TemplateHandler) buildPostStructuredData(post *models.Post, site models.SiteSettings, canonicalURL string) template.JS {
	if post == nil {
		return ""
	}

	baseURL := site.URL
	if baseURL == "" {
		baseURL = h.config.SiteURL
	}

	publishedAt := post.CreatedAt
	if post.PublishedAt != nil {
		publishedAt = post.PublishedAt.UTC()
	} else if post.PublishAt != nil {
		publishedAt = post.PublishAt.UTC()
	}

	article := map[string]interface{}{
		"@context":      "https://schema.org",
		"@type":         "BlogPosting",
		"headline":      post.Title,
		"datePublished": publishedAt.Format(time.RFC3339),
		"dateModified":  post.UpdatedAt.Format(time.RFC3339),
		"mainEntityOfPage": map[string]interface{}{
			"@type": "WebPage",
			"@id":   canonicalURL,
		},
	}

	if langCode := strings.TrimSpace(site.DefaultLanguage); langCode != "" {
		article["inLanguage"] = langCode
	}

	if canonicalURL != "" {
		article["url"] = canonicalURL
	}

	if post.Description != "" {
		article["description"] = post.Description
	}

	if post.Author.Username != "" {
		article["author"] = map[string]interface{}{
			"@type": "Person",
			"name":  post.Author.Username,
		}
	}

	publisher := map[string]interface{}{
		"@type": "Organization",
		"name":  site.Name,
	}

	if logo := h.ensureAbsoluteURL(baseURL, site.Logo); logo != "" {
		publisher["logo"] = map[string]interface{}{
			"@type": "ImageObject",
			"url":   logo,
		}
	}

	article["publisher"] = publisher

	if post.Category.Name != "" {
		article["articleSection"] = post.Category.Name
	}

	if len(post.Tags) > 0 {
		var tags []string
		for _, tag := range post.Tags {
			if tag.Name != "" {
				tags = append(tags, tag.Name)
			}
		}
		if len(tags) > 0 {
			article["keywords"] = strings.Join(tags, ", ")
		}
	}

	if image := h.ensureAbsoluteURL(baseURL, post.FeaturedImg); image != "" {
		article["image"] = []string{image}
	}

	if wordCount := len(strings.Fields(post.Content)); wordCount > 0 {
		article["wordCount"] = wordCount
	}

	data, err := json.Marshal(article)
	if err != nil {
		logger.Error(err, "Failed to build post structured data", map[string]interface{}{"post_id": post.ID})
		return ""
	}

	return template.JS(data)
}

func (h *TemplateHandler) renderCustomPage(c *gin.Context, page *models.Page) {
	if page == nil {
		h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
		return
	}

	var contentHTML template.HTML
	if strings.TrimSpace(page.Content) != "" {
		contentHTML = template.HTML(page.Content)
	}

	sectionsHTML, sectionScripts := h.renderSectionsWithPrefix(page.Sections, "page-view")

	data := gin.H{
		"Page": page,
	}

	if contentHTML != "" {
		data["Content"] = contentHTML
	}

	if sectionsHTML != "" {
		data["Sections"] = sectionsHTML
	}

	if len(sectionScripts) > 0 {
		data["Scripts"] = appendScripts(asScriptSlice(data["Scripts"]), sectionScripts)
	}

	templateName := strings.TrimSpace(page.Template)
	if templateName == "" {
		templateName = "page"
	}

	h.renderTemplate(c, templateName, page.Title, page.Description, data)
}

func (h *TemplateHandler) renderPageByTemplate(c *gin.Context, page *models.Page) {
	if page == nil {
		h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
		return
	}

	templateName := strings.TrimSpace(strings.ToLower(page.Template))
	switch templateName {
	case "blog":
		h.renderBlogWithPage(c, page)
	default:
		h.renderCustomPage(c, page)
	}
}

func (h *TemplateHandler) renderPageForPath(c *gin.Context, path string) bool {
	if h.pageService == nil {
		return false
	}

	page, err := h.pageService.GetByPath(path)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false
		}
		logger.Error(err, "Failed to load page", map[string]interface{}{"path": path})
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load page")
		return true
	}

	h.renderPageByTemplate(c, page)
	return true
}

func (h *TemplateHandler) TryRenderPage(c *gin.Context) bool {
	path := c.Request.URL.Path
	if path == "" {
		path = "/"
	}
	return h.renderPageForPath(c, path)
}

func (h *TemplateHandler) RenderIndex(c *gin.Context) {
	if h.homepageService != nil {
		page, err := h.homepageService.GetActiveHomepage()
		if err != nil {
			logger.Error(err, "Failed to load configured homepage", nil)
		} else if page != nil {
			h.renderPageByTemplate(c, page)
			return
		}
	}

	if h.renderPageForPath(c, "/") {
		return
	}

	h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Homepage is not configured")
}

func (h *TemplateHandler) RenderPost(c *gin.Context) {
	if !h.ensureBlogAvailable(c) {
		return
	}

	param := c.Param("slug")

	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		post, err := h.postService.GetByID(uint(id))
		if err != nil {
			h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
			return
		}
		h.renderSinglePost(c, post)
		return
	}

	post, err := h.postService.GetBySlug(param)
	if err != nil {
		h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
		return
	}
	h.renderSinglePost(c, post)
}

func (h *TemplateHandler) RenderPage(c *gin.Context) {
	if h.TryRenderPage(c) {
		return
	}

	slug := strings.TrimSpace(c.Param("slug"))
	if slug != "" && h.pageService != nil {
		if page, err := h.pageService.GetBySlug(slug); err == nil {
			if strings.TrimSpace(page.Path) != "" {
				c.Redirect(http.StatusMovedPermanently, page.Path)
				return
			}
		}
	}

	h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested page not found")
}

func (h *TemplateHandler) loadBlogCollections(page, limit int) ([]models.Post, int64, []models.Tag, []models.Category, error) {
	if h.postService == nil {
		return nil, 0, nil, nil, errors.New("blog plugin inactive")
	}

	posts, total, err := h.postService.GetAll(page, limit, nil, nil, nil)
	if err != nil {
		return nil, 0, nil, nil, err
	}

	tags, tagErr := h.postService.GetTagsInUse()
	if tagErr != nil {
		logger.Error(tagErr, "Failed to load tags", nil)
	}

	var categories []models.Category
	if h.categoryService != nil {
		if loadedCategories, catErr := h.categoryService.GetAll(); catErr != nil {
			logger.Error(catErr, "Failed to load categories", nil)
		} else if len(loadedCategories) > 0 {
			// Determine whether there are any non-default categories.
			hasNonDefault := false
			for _, category := range loadedCategories {
				if !strings.EqualFold(category.Slug, "uncategorized") && !strings.EqualFold(category.Name, "uncategorized") {
					hasNonDefault = true
					break
				}
			}

			filteredCategories := make([]models.Category, 0, len(loadedCategories))
			for _, category := range loadedCategories {
				if (strings.EqualFold(category.Slug, "uncategorized") || strings.EqualFold(category.Name, "uncategorized")) && hasNonDefault {
					continue
				}
				filteredCategories = append(filteredCategories, category)
			}
			categories = filteredCategories
		}
	}

	return posts, total, tags, categories, nil
}

func (h *TemplateHandler) renderBlogWithPage(c *gin.Context, page *models.Page) {
	if !h.ensureBlogAvailable(c) {
		return
	}

	if page == nil {
		h.renderLegacyBlog(c)
		return
	}

	pageNumber, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "12"))
	if err != nil || limit <= 0 {
		limit = 12
	}

	posts, total, tags, categories, fetchErr := h.loadBlogCollections(pageNumber, limit)
	if fetchErr != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	basePath := page.Path
	if basePath == "" {
		basePath = "/blog"
	}

	pagination := h.buildPagination(pageNumber, totalPages, func(p int) string {
		return fmt.Sprintf("%s?page=%d", basePath, p)
	})

	data := gin.H{
		"Page":        page,
		"Posts":       posts,
		"Total":       total,
		"CurrentPage": pageNumber,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	if len(categories) > 0 {
		data["Categories"] = categories
	}

	if strings.TrimSpace(page.Content) != "" {
		data["Content"] = template.HTML(page.Content)
	}

	sections, sectionScripts := h.renderSectionsWithPrefix(page.Sections, "blog")

	data["PageViewModifiers"] = []string{"blog"}

	if overview := h.renderBlogOverviewSection(posts, tags, categories, pagination, sections); overview != "" {
		data["Sections"] = overview
	} else if sections != "" {
		data["Sections"] = sections
	}

	if len(sectionScripts) > 0 {
		data["Scripts"] = appendScripts(asScriptSlice(data["Scripts"]), sectionScripts)
	}

	title := page.Title
	if strings.TrimSpace(title) == "" {
		title = "Posts"
	}

	description := page.Description
	if strings.TrimSpace(description) == "" {
		description = h.config.SiteName + " — insights, articles, and updates about various topics."
	}

	templateName := strings.TrimSpace(page.Template)
	if templateName == "" {
		templateName = "page"
	}

	h.renderTemplate(c, templateName, title, description, data)
}

func (h *TemplateHandler) renderLegacyBlog(c *gin.Context) {
	if !h.ensureBlogAvailable(c) {
		return
	}

	pageNumber, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "12"))
	if err != nil || limit <= 0 {
		limit = 12
	}

	posts, total, tags, categories, fetchErr := h.loadBlogCollections(pageNumber, limit)
	if fetchErr != nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(pageNumber, totalPages, func(p int) string {
		return fmt.Sprintf("/blog?page=%d", p)
	})

	page := &models.Page{
		Title:       "Posts",
		Description: h.config.SiteName + " — insights, articles, and updates about various topics.",
		Path:        "/blog",
		Template:    "page",
	}

	data := gin.H{
		"Page":              page,
		"Posts":             posts,
		"Total":             total,
		"CurrentPage":       pageNumber,
		"TotalPages":        totalPages,
		"Pagination":        pagination,
		"PageViewModifiers": []string{"blog"},
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	if len(categories) > 0 {
		data["Categories"] = categories
	}

	if overview := h.renderBlogOverviewSection(posts, tags, categories, pagination, ""); overview != "" {
		data["Sections"] = overview
	}

	h.renderTemplate(c, page.Template, page.Title, page.Description, data)
}

func (h *TemplateHandler) RenderBlog(c *gin.Context) {
	path := c.Request.URL.Path
	if path == "" {
		path = "/blog"
	}

	if h.renderPageForPath(c, path) {
		return
	}

	h.renderLegacyBlog(c)
}

func (h *TemplateHandler) renderBlogOverviewSection(posts []models.Post, tags []models.Tag, categories []models.Category, pagination gin.H, extraSections template.HTML) template.HTML {
	tmpl, err := h.templateClone()
	if err != nil {
		logger.Error(err, "Failed to clone templates for blog overview", nil)
		return extraSections
	}

	component := tmpl.Lookup("components/blog-overview")
	if component == nil {
		logger.Error(nil, "Blog overview component missing", nil)
		return extraSections
	}

	data := gin.H{
		"Posts":      posts,
		"Tags":       tags,
		"Categories": categories,
	}

	if pagination != nil {
		data["Pagination"] = pagination
	}

	if extraSections != "" {
		data["ExtraSections"] = extraSections
	}

	output, execErr := h.executeTemplate(component, data)
	if execErr != nil {
		logger.Error(execErr, "Failed to render blog overview component", nil)
		return extraSections
	}

	return template.HTML(output)
}

func (h *TemplateHandler) RenderSearch(c *gin.Context) {
	if h.searchService == nil {
		h.renderError(c, http.StatusServiceUnavailable, "Search unavailable", "The blog plugin is not active.")
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	searchType := c.DefaultQuery("type", "all")
	limitValue := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitValue)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	var result *blogservice.SearchResult
	if query != "" {
		searchResult, searchErr := h.searchService.Search(query, searchType, limit)
		if searchErr != nil {
			logger.Error(searchErr, "Failed to execute search", map[string]interface{}{"query": query, "type": searchType})
			h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to perform search")
			return
		}
		result = searchResult
	} else {
		result = &blogservice.SearchResult{Posts: []models.Post{}, Total: 0, Query: query}
	}

	hasQuery := query != ""
	title := "Search"
	description := fmt.Sprintf("Search articles on %s to discover tutorials, guides, and engineering stories.", h.config.SiteName)
	if hasQuery {
		title = fmt.Sprintf("Search results for \"%s\"", query)
		description = fmt.Sprintf("Search results for \"%s\" on %s.", query, h.config.SiteName)
	}

	data := gin.H{
		"Query":       query,
		"SearchType":  searchType,
		"Limit":       limit,
		"HasQuery":    hasQuery,
		"Result":      result,
		"SearchQuery": query,
		"NoIndex":     true,
	}

	if result != nil {
		data["Total"] = result.Total
	}

	h.renderTemplate(c, "search", title, description, data)
}

func (h *TemplateHandler) RenderForum(c *gin.Context) {
	if !h.ensureForumAvailable(c) {
		return
	}

	pageNumber, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}

	limitParam := strings.TrimSpace(c.Query("limit"))
	limit := 20
	if limitParam != "" {
		if parsed, parseErr := strconv.Atoi(limitParam); parseErr == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	search := strings.TrimSpace(c.Query("search"))
	categorySlug := strings.TrimSpace(c.Query("category"))

	var categories []models.ForumCategory
	var activeCategory *models.ForumCategory
	if h.forumCategorySvc != nil {
		if list, err := h.forumCategorySvc.GetAll(); err != nil {
			logger.Error(err, "Failed to load forum categories", nil)
		} else {
			categories = list
			if categorySlug != "" {
				for idx := range categories {
					slug := strings.TrimSpace(categories[idx].Slug)
					if strings.EqualFold(slug, categorySlug) {
						activeCategory = &categories[idx]
						break
					}
				}
			}
		}
	}

	var categoryID *uint
	if activeCategory != nil {
		id := activeCategory.ID
		categoryID = &id
	} else if categoryIDParam := strings.TrimSpace(c.Query("category_id")); categoryIDParam != "" {
		if parsed, parseErr := strconv.ParseUint(categoryIDParam, 10, 64); parseErr == nil && parsed > 0 {
			value := uint(parsed)
			categoryID = &value
			for idx := range categories {
				if categories[idx].ID == value {
					activeCategory = &categories[idx]
					break
				}
			}
		}
	}

	options := forumservice.QuestionListOptions{
		Search:       search,
		CategoryID:   categoryID,
		CategorySlug: categorySlug,
	}

	questions, total, listErr := h.forumQuestionSvc.List(pageNumber, limit, options)
	if listErr != nil {
		logger.Error(listErr, "Failed to load forum questions", map[string]interface{}{"page": pageNumber, "search": search, "category": categorySlug})
		h.renderError(c, http.StatusInternalServerError, "Forum unavailable", "We couldn't load the forum questions right now.")
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}
	if totalPages < 1 {
		totalPages = 1
	}

	buildCategoryURL := func(slug string, id *uint) string {
		params := url.Values{}
		if search != "" {
			params.Set("search", search)
		}
		if slug != "" {
			params.Set("category", slug)
		} else if id != nil {
			params.Set("category_id", strconv.FormatUint(uint64(*id), 10))
		}
		if limitParam != "" && limitParam != strconv.Itoa(limit) {
			params.Set("limit", limitParam)
		}
		base := "/forum"
		if len(params) == 0 {
			return base
		}
		return base + "?" + params.Encode()
	}

	pagination := h.buildPagination(pageNumber, totalPages, func(p int) string {
		params := url.Values{}
		if search != "" {
			params.Set("search", search)
		}
		if categorySlug != "" {
			params.Set("category", categorySlug)
		} else if categoryID != nil {
			params.Set("category_id", strconv.FormatUint(uint64(*categoryID), 10))
		}
		if p > 1 {
			params.Set("page", strconv.Itoa(p))
		}
		if limitParam != "" && limitParam != strconv.Itoa(limit) {
			params.Set("limit", limitParam)
		}
		base := "/forum"
		if len(params) == 0 {
			return base
		}
		return base + "?" + params.Encode()
	})

	pageTitle := "Community forum"
	description := "Join the community forum to ask questions, share insights, and collaborate with other members."
	if activeCategory != nil && search == "" {
		pageTitle = fmt.Sprintf("%s discussions", strings.TrimSpace(activeCategory.Name))
		description = fmt.Sprintf("Community conversations in the %s category.", strings.TrimSpace(activeCategory.Name))
	} else if activeCategory != nil && search != "" {
		pageTitle = fmt.Sprintf("Results for \"%s\" in %s", search, strings.TrimSpace(activeCategory.Name))
		description = fmt.Sprintf("Questions matching \"%s\" within the %s forum category.", search, strings.TrimSpace(activeCategory.Name))
	} else if search != "" {
		pageTitle = fmt.Sprintf("Forum results for \"%s\"", search)
		description = fmt.Sprintf("Questions matching \"%s\" from the community discussion board.", search)
	}

	canonicalPath := "/forum"
	params := url.Values{}
	if search != "" {
		params.Set("search", search)
	}
	if categorySlug != "" {
		params.Set("category", categorySlug)
	} else if categoryID != nil {
		params.Set("category_id", strconv.FormatUint(uint64(*categoryID), 10))
	}
	if pageNumber > 1 {
		params.Set("page", strconv.Itoa(pageNumber))
	}
	if limitParam != "" && limitParam != strconv.Itoa(limit) {
		params.Set("limit", limitParam)
	}
	if len(params) > 0 {
		canonicalPath = canonicalPath + "?" + params.Encode()
	}

	categoryFilters := make([]gin.H, 0, len(categories)+1)
	activeFilterName := "All discussions"
	defaultFilter := gin.H{
		"Name":   "All discussions",
		"Slug":   "",
		"URL":    buildCategoryURL("", nil),
		"Active": activeCategory == nil && categorySlug == "" && categoryID == nil,
	}
	if isActive, ok := defaultFilter["Active"].(bool); ok && isActive {
		activeFilterName = defaultFilter["Name"].(string)
	}
	categoryFilters = append(categoryFilters, defaultFilter)
	for i := range categories {
		slug := strings.TrimSpace(categories[i].Slug)
		id := categories[i].ID
		isActive := false
		if activeCategory != nil {
			isActive = activeCategory.ID == categories[i].ID
		} else if categorySlug != "" {
			isActive = strings.EqualFold(slug, categorySlug)
		}
		name := strings.TrimSpace(categories[i].Name)
		filter := gin.H{
			"Name":   name,
			"Slug":   slug,
			"URL":    buildCategoryURL(slug, &id),
			"Active": isActive,
		}
		if isActive && name != "" {
			activeFilterName = name
		}
		categoryFilters = append(categoryFilters, filter)
	}

	extra := gin.H{
		"ForumQuestions": questions,
		"ForumSearch":    search,
		"ForumTotal":     total,
		"ForumPage": gin.H{
			"Current":    pageNumber,
			"Limit":      limit,
			"TotalPages": totalPages,
		},
		"ForumEndpoints": gin.H{
			"Create": "/api/v1/forum/questions",
		},
		"Scripts":                 []string{"/static/js/forum.js"},
		"Canonical":               h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath),
		"ForumPath":               "/forum",
		"ForumCategories":         categories,
		"ForumActiveCategory":     activeCategory,
		"ForumActiveCategorySlug": categorySlug,
		"ForumCategoryFilters":    categoryFilters,
		"ForumActiveCategoryName": activeFilterName,
	}

	if pagination != nil {
		extra["Pagination"] = pagination
	}

	if search != "" {
		extra["NoIndex"] = true
	}
	if activeCategory != nil && search == "" {
		extra["PageType"] = "collection"
	}

	h.renderTemplate(c, "forum", pageTitle, description, extra)
}

func (h *TemplateHandler) RenderForumQuestion(c *gin.Context) {
	if !h.ensureForumAvailable(c) {
		return
	}

	identifier := strings.TrimSpace(c.Param("slug"))
	if identifier == "" {
		h.renderError(c, http.StatusNotFound, "404 - Question not found", "The requested discussion could not be found.")
		return
	}

	question, err := h.forumQuestionSvc.GetBySlug(identifier)
	if err != nil {
		if !errors.Is(err, forumservice.ErrQuestionNotFound) {
			logger.Error(err, "Failed to load forum question", map[string]interface{}{"identifier": identifier})
		}
		if idValue, parseErr := strconv.ParseUint(identifier, 10, 64); parseErr == nil {
			question, err = h.forumQuestionSvc.GetByID(uint(idValue))
		}
	}

	if err != nil {
		if errors.Is(err, forumservice.ErrQuestionNotFound) {
			h.renderError(c, http.StatusNotFound, "404 - Question not found", "The requested discussion could not be found.")
		} else {
			logger.Error(err, "Failed to load forum question", map[string]interface{}{"identifier": identifier})
			h.renderError(c, http.StatusInternalServerError, "Forum unavailable", "We couldn't load this discussion right now.")
		}
		return
	}

	slug := strings.TrimSpace(question.Slug)
	if slug != "" && !strings.EqualFold(slug, identifier) {
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/forum/%s", slug))
		return
	}

	answerCount := len(question.Answers)
	canonicalPath := fmt.Sprintf("/forum/%s", slug)
	if slug == "" {
		canonicalPath = fmt.Sprintf("/forum/%d", question.ID)
	}
	canonicalURL := h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath)

	contentSummary := truncatePlainText(strings.TrimSpace(question.Content), 160)
	authorName := strings.TrimSpace(question.Author.Username)
	description := contentSummary
	if description == "" {
		if authorName != "" {
			description = fmt.Sprintf("Discussion started by %s.", authorName)
		} else {
			description = "Community discussion thread."
		}
	} else if authorName != "" {
		description = fmt.Sprintf("%s — %s", authorName, description)
	}

	site := h.siteSettings()
	structuredData := h.buildForumStructuredData(question, site, canonicalURL)

	var (
		forumCurrentUserID       uint
		forumCanManageAllAnswers bool
		canDeleteQuestion        bool
	)

	if user, ok := h.currentUser(c); ok {
		forumCurrentUserID = user.ID
		forumCanManageAllAnswers = authorization.RoleHasPermission(user.Role, authorization.PermissionManageAllContent)
		if user.ID == question.AuthorID || forumCanManageAllAnswers {
			canDeleteQuestion = true
		}
	}

	loginRedirect := c.Request.URL.RequestURI()
	if loginRedirect == "" {
		loginRedirect = canonicalPath
	}

	extra := gin.H{
		"Question":         question,
		"ForumAnswerCount": answerCount,
		"ForumEndpoints": gin.H{
			"Question":     fmt.Sprintf("/api/v1/forum/questions/%d", question.ID),
			"QuestionVote": fmt.Sprintf("/api/v1/forum/questions/%d/vote", question.ID),
			"AnswerCreate": fmt.Sprintf("/api/v1/forum/questions/%d/answers", question.ID),
			"AnswerBase":   "/api/v1/forum/answers",
			"AnswerVote":   "/api/v1/forum/answers",
		},
		"ForumQuestionCanDelete":   canDeleteQuestion,
		"ForumCanManageAllAnswers": forumCanManageAllAnswers,
		"ForumCurrentUserID":       forumCurrentUserID,
		"ForumPath":                "/forum",
		"Scripts":                  []string{"/static/js/forum.js"},
		"Canonical":                canonicalURL,
		"StructuredData":           structuredData,
		"OGType":                   "article",
		"OGURL":                    canonicalURL,
		"TwitterCard":              "summary_large_image",
		"ForumLoginURL":            fmt.Sprintf("/login?redirect=%s", url.QueryEscape(loginRedirect)),
	}

	h.renderTemplate(c, "forum_question", question.Title, description, extra)
}

func truncatePlainText(value string, limit int) string {
	trimmed := strings.TrimSpace(value)
	if limit <= 0 || trimmed == "" {
		return trimmed
	}
	runes := []rune(trimmed)
	if len(runes) <= limit {
		return trimmed
	}
	truncated := strings.TrimSpace(string(runes[:limit]))
	if truncated == "" {
		truncated = strings.TrimSpace(string(runes[:limit]))
	}
	return truncated + "…"
}

func (h *TemplateHandler) buildForumStructuredData(question *models.ForumQuestion, site models.SiteSettings, canonicalURL string) template.JS {
	if question == nil {
		return ""
	}

	payload := map[string]any{
		"@context": "https://schema.org",
		"@type":    "QAPage",
	}

	questionData := map[string]any{
		"@type":        "Question",
		"name":         strings.TrimSpace(question.Title),
		"text":         strings.TrimSpace(question.Content),
		"answerCount":  question.AnswersCount,
		"upvoteCount":  question.Rating,
		"dateCreated":  question.CreatedAt.UTC().Format(time.RFC3339),
		"dateModified": question.UpdatedAt.UTC().Format(time.RFC3339),
	}

	if canonicalURL != "" {
		payload["url"] = canonicalURL
		questionData["url"] = canonicalURL
	}

	if authorName := strings.TrimSpace(question.Author.Username); authorName != "" {
		questionData["author"] = map[string]any{
			"@type": "Person",
			"name":  authorName,
		}
	}

	answers := make([]map[string]any, 0, len(question.Answers))
	for _, answer := range question.Answers {
		text := strings.TrimSpace(answer.Content)
		if text == "" {
			continue
		}
		answerData := map[string]any{
			"@type":        "Answer",
			"text":         text,
			"dateCreated":  answer.CreatedAt.UTC().Format(time.RFC3339),
			"dateModified": answer.UpdatedAt.UTC().Format(time.RFC3339),
			"upvoteCount":  answer.Rating,
		}
		if author := strings.TrimSpace(answer.Author.Username); author != "" {
			answerData["author"] = map[string]any{
				"@type": "Person",
				"name":  author,
			}
		}
		answers = append(answers, answerData)
	}

	if len(answers) > 0 {
		questionData["suggestedAnswer"] = answers
	}

	payload["mainEntity"] = questionData

	if site.Name != "" {
		payload["name"] = fmt.Sprintf("%s — %s", site.Name, strings.TrimSpace(question.Title))
	}

	data, err := json.Marshal(payload)
	if err != nil {
		logger.Error(err, "Failed to build forum structured data", map[string]interface{}{"question_id": question.ID})
		return ""
	}
	return template.JS(data)
}

func (h *TemplateHandler) RenderCategory(c *gin.Context) {
	if !h.ensureBlogAvailable(c) {
		return
	}

	slug := c.Param("slug")
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "12"))
	if err != nil || limit <= 0 {
		limit = 12
	}
	if limit > 50 {
		limit = 50
	}

	category, posts, total, err := h.postService.GetCategoryWithPosts(slug, page, limit)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested category not found")
		} else {
			logger.Error(err, "Failed to load category posts", map[string]interface{}{"slug": slug})
			h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		}
		return
	}

	var categories []models.Category
	if h.categoryService != nil {
		if loadedCategories, catErr := h.categoryService.GetAll(); catErr != nil {
			logger.Error(catErr, "Failed to load categories", nil)
		} else {
			if len(loadedCategories) > 0 {
				filteredCategories := make([]models.Category, 0, len(loadedCategories))
				for _, category := range loadedCategories {
					if strings.EqualFold(category.Slug, "uncategorized") || strings.EqualFold(category.Name, "uncategorized") {
						continue
					}
					filteredCategories = append(filteredCategories, category)
				}
				categories = filteredCategories
			}
		}
	}

	totalCount := int(total)
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(page, totalPages, func(p int) string {
		return fmt.Sprintf("/category/%s?page=%d", category.Slug, p)
	})

	categoryName := category.Name
	if categoryName == "" {
		categoryName = category.Slug
	}

	description := strings.TrimSpace(category.Description)
	if description == "" {
		description = fmt.Sprintf("Articles in the \"%s\" category on %s.", categoryName, h.config.SiteName)
	}

	data := gin.H{
		"Posts":       posts,
		"Total":       totalCount,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
		"Category":    category,
		"Canonical":   fmt.Sprintf("/category/%s", category.Slug),
	}

	if len(categories) > 0 {
		data["Categories"] = categories
	}

	if category.Name != "" {
		data["Keywords"] = category.Name
	} else if category.Slug != "" {
		data["Keywords"] = category.Slug
	}

	site := h.siteSettings()
	baseURL := site.URL
	if baseURL == "" {
		baseURL = h.config.SiteURL
	}

	itemList := make([]map[string]interface{}, 0, len(posts))
	for idx, post := range posts {
		position := (page-1)*limit + idx + 1
		postURL := fmt.Sprintf("/blog/post/%s", post.Slug)
		if post.Slug == "" {
			postURL = fmt.Sprintf("/blog/post/%d", post.ID)
		}
		absoluteURL := h.ensureAbsoluteURL(baseURL, postURL)

		item := map[string]interface{}{
			"@type":    "ListItem",
			"position": position,
			"name":     post.Title,
		}
		if absoluteURL != "" {
			item["url"] = absoluteURL
		}
		itemList = append(itemList, item)
	}

	structuredData := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        fmt.Sprintf("%s category", categoryName),
		"description": description,
		"mainEntity": map[string]interface{}{
			"@type":           "ItemList",
			"itemListElement": itemList,
		},
	}

	if dataBytes, marshalErr := json.Marshal(structuredData); marshalErr == nil {
		data["StructuredData"] = template.JS(dataBytes)
	} else {
		logger.Error(marshalErr, "Failed to marshal category structured data", map[string]interface{}{"category": category.Slug})
	}

	h.renderTemplate(c, "category", "Category: "+categoryName, description, data)
}

func (h *TemplateHandler) RenderTag(c *gin.Context) {
	if !h.ensureBlogAvailable(c) {
		return
	}

	slug := c.Param("slug")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "12"))

	logger.Debug("Rendering tag page", map[string]interface{}{
		"slug":  slug,
		"page":  page,
		"limit": limit,
	})

	tag, posts, total, err := h.postService.GetTagWithPosts(slug, page, limit)
	if err != nil {
		logger.Error(err, "Failed to render tag", map[string]interface{}{
			"slug":  slug,
			"page":  page,
			"limit": limit,
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			h.renderError(c, http.StatusNotFound, "404 - Page Not Found", "Requested tag not found")
		} else {
			h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to load posts")
		}
		return
	}

	logger.Debug("Tag page data loaded", map[string]interface{}{
		"slug":        slug,
		"page":        page,
		"limit":       limit,
		"posts_count": len(posts),
		"total":       total,
	})

	var tags []models.Tag
	if loadedTags, tagErr := h.postService.GetTagsInUse(); tagErr != nil {
		logger.Error(tagErr, "Failed to load tags", nil)
	} else {
		tags = loadedTags
		logger.Debug("Loaded sidebar tags", map[string]interface{}{
			"slug":       slug,
			"tags_count": len(tags),
		})
	}

	totalCount := int(total)

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	pagination := h.buildPagination(page, totalPages, func(p int) string {
		return fmt.Sprintf("/tag/%s?page=%d", slug, p)
	})

	data := gin.H{
		"Posts":       posts,
		"Total":       totalCount,
		"CurrentPage": page,
		"TotalPages":  totalPages,
		"Pagination":  pagination,
		"Tag":         tag,
	}

	if len(tags) > 0 {
		data["Tags"] = tags
	}

	description := "Articles tagged with \"" + tag.Name + "\" — development topics, code patterns, and Go programming insights on " + h.config.SiteName + "."
	h.renderTemplate(c, "tag", "Tag: "+tag.Name, description, data)
}

func (h *TemplateHandler) RenderLogin(c *gin.Context) {
	if _, ok := h.currentUser(c); ok {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	redirectTo := c.Query("redirect")
	if redirectTo != "" {
		if decoded, err := url.QueryUnescape(redirectTo); err == nil {
			redirectTo = decoded
		}

		if !strings.HasPrefix(redirectTo, "/") {
			redirectTo = "/profile"
		}
	} else {
		redirectTo = "/profile"
	}

	h.renderTemplate(c, "login", "Sign in", "Access your dashboard and manage your content.", gin.H{
		"AuthAction": "/api/v1/login",
		"RedirectTo": redirectTo,
		"NoIndex":    true,
	})
}

func (h *TemplateHandler) RenderRegister(c *gin.Context) {
	if _, ok := h.currentUser(c); ok {
		c.Redirect(http.StatusFound, "/profile")
		return
	}

	h.renderTemplate(c, "register", "Create an account", "Join the community to publish articles and leave comments.", gin.H{
		"RegisterAction": "/api/v1/register",
	})
}

func (h *TemplateHandler) RenderSetup(c *gin.Context) {
	if h.setupService == nil {
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Setup is not available")
		return
	}

	complete, err := h.setupService.IsSetupComplete()
	if err != nil {
		logger.Error(err, "Failed to determine setup status", nil)
		h.renderError(c, http.StatusInternalServerError, "500 - Server Error", "Failed to verify setup status")
		return
	}

	if complete {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}

	// Preserve the setup key in the page context if provided
	setupKey := c.Query("key")

	h.renderTemplate(c, "setup", "Initial setup", "Create the first administrator account and configure the site.", gin.H{
		"Scripts":     []string{"/static/js/setup-config.js", "/static/js/setup.js"},
		"SetupAction": "/api/v1/setup",
		"SetupStatus": "/api/v1/setup/status",
		"SetupKey":    setupKey,
		"HideChrome":  true,
		"NoIndex":     true,
		"Layout":      "setup_base.html",
	})
}

func (h *TemplateHandler) RenderSetupKeyRequired(c *gin.Context) {
	// Render the setup key required page directly
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Header("X-Robots-Tag", "noindex, nofollow")
	c.Status(http.StatusOK)

	// Try to load from theme, fallback to inline HTML
	tmpl, err := h.templateClone()
	if err == nil {
		if t := tmpl.Lookup("setup_key_required.html"); t != nil {
			if err := t.Execute(c.Writer, nil); err == nil {
				return
			}
		}
	}

	// Fallback: serve inline HTML
	c.Writer.Write([]byte(`<!DOCTYPE html>
<html lang="en" data-theme="light">
<head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Setup Access Required</title>
    <style>
        :root {
            --color-background: #f9fafb;
            --color-surface: #ffffff;
            --color-text: #111827;
            --color-text-muted: #6b7280;
            --color-border: #e5e7eb;
            --color-primary: #3b82f6;
            --color-primary-dark: #2563eb;
        }
        @media (prefers-color-scheme: dark) {
            :root {
                --color-background: #111827;
                --color-surface: #1f2937;
                --color-text: #f9fafb;
                --color-text-muted: #9ca3af;
                --color-border: #374151;
            }
        }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            background: var(--color-background);
            color: var(--color-text);
            font-family: system-ui, -apple-system, sans-serif;
            padding: 1rem;
        }
        .container {
            max-width: 500px;
            width: 100%;
            padding: 2rem;
            background: var(--color-surface);
            border-radius: 12px;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
            text-align: center;
        }
        .icon { font-size: 3rem; margin-bottom: 1rem; opacity: 0.8; }
        h1 { font-size: 1.875rem; font-weight: 700; margin-bottom: 0.5rem; }
        p { color: var(--color-text-muted); margin-bottom: 2rem; line-height: 1.5; }
        form { display: flex; flex-direction: column; gap: 1rem; }
        input {
            padding: 0.75rem 1rem;
            font-size: 1rem;
            border: 2px solid var(--color-border);
            border-radius: 8px;
            background: var(--color-background);
            color: var(--color-text);
            font-family: monospace;
        }
        input:focus {
            outline: none;
            border-color: var(--color-primary);
        }
        button {
            padding: 0.75rem 1.5rem;
            font-size: 1rem;
            font-weight: 600;
            background: var(--color-primary);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
        }
        button:hover { background: var(--color-primary-dark); }
        .error {
            padding: 0.75rem;
            background: rgba(239, 68, 68, 0.1);
            border: 1px solid rgba(239, 68, 68, 0.3);
            border-radius: 6px;
            color: #dc2626;
            font-size: 0.875rem;
            display: none;
        }
        .error.show { display: block; }
        .help {
            margin-top: 1.5rem;
            padding-top: 1.5rem;
            border-top: 1px solid var(--color-border);
            font-size: 0.875rem;
        }
        code {
            background: var(--color-background);
            padding: 0.125rem 0.375rem;
            border-radius: 4px;
            font-family: monospace;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">🔐</div>
        <h1>Setup Access Required</h1>
        <p>A security key is required to access the setup page. This key was generated when you ran the deployment script.</p>
        <form id="form">
            <div class="error" id="error"></div>
            <input type="text" id="key" placeholder="Enter your setup key" autocomplete="off" spellcheck="false" required />
            <button type="submit">Continue to Setup</button>
        </form>
        <div class="help">
            <p>💡 The setup key was displayed when you ran <code>./deploy/quickstart.sh</code> or can be found in <code>deploy/.env.production</code> as <code>SETUP_KEY</code>.</p>
        </div>
    </div>
    <script>
        const form = document.getElementById('form');
        const input = document.getElementById('key');
        const errorDiv = document.getElementById('error');
        const params = new URLSearchParams(window.location.search);
        if (params.get('error')) {
            errorDiv.textContent = params.get('error');
            errorDiv.classList.add('show');
        }
        form.addEventListener('submit', (e) => {
            e.preventDefault();
            errorDiv.classList.remove('show');
            const key = input.value.trim();
            if (!key) {
                errorDiv.textContent = 'Please enter a setup key';
                errorDiv.classList.add('show');
                return;
            }
            window.location.href = '/setup?key=' + encodeURIComponent(key);
        });
        input.focus();
    </script>
</body>
</html>`))
}

type profileTab struct {
	ID          string
	Label       string
	Description string
	Content     template.HTML
}

func (h *TemplateHandler) RenderProfile(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		redirectTo := url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, "/login?redirect="+redirectTo)
		return
	}

	courses := make([]models.UserCoursePackage, 0)
	if h.coursePackageSvc != nil && user != nil {
		if loaded, err := h.coursePackageSvc.ListForUser(user.ID); err != nil {
			logger.Error(err, "Failed to load courses for profile", map[string]interface{}{"user_id": user.ID})
		} else {
			courses = loaded
		}
	}

	sections, page := h.profileSectionsForUser(user, courses)
	sectionsHTML, sectionScripts := h.renderSectionsWithPrefix(sections, "page-view")
	scripts := appendScripts(nil, sectionScripts)
	defaultTitle := "Profile"
	defaultDescription := "Manage personal details, account security, and connected devices."

	if page == nil {
		page = &models.Page{}
	}

	pageTitle := strings.TrimSpace(page.Title)
	if pageTitle == "" {
		pageTitle = defaultTitle
	}

	pageDescription := strings.TrimSpace(page.Description)
	if pageDescription == "" {
		pageDescription = defaultDescription
	}

	if strings.TrimSpace(page.Path) == "" {
		page.Path = "/profile"
	}

	templateName := strings.TrimSpace(page.Template)
	if templateName == "" ||
		strings.EqualFold(templateName, "page") ||
		strings.EqualFold(templateName, "profile") {
		templateName = "profile"
	}

	page.Title = pageTitle
	page.Description = pageDescription
	page.Template = templateName

	data := gin.H{
		"Page":               page,
		"UserCourses":        courses,
		"PageViewModifiers":  []string{"profile"},
		"PageViewAttributes": template.HTMLAttr(`data-page="profile"`),
	}

	tabs := buildProfileTabs(h, sections)
	if len(tabs) > 0 {
		data["ProfileTabs"] = tabs
		data["ProfileDefaultTab"] = tabs[0].ID
	}

	if sectionsHTML != "" {
		data["Sections"] = sectionsHTML
	}

	if len(scripts) > 0 {
		data["Scripts"] = scripts
	}

	h.renderTemplate(c, templateName, pageTitle, pageDescription, data)
}

func (h *TemplateHandler) profileSectionsForUser(user *models.User, courses []models.UserCoursePackage) (models.PostSections, *models.Page) {
	if h == nil {
		return applyProfileUserContext(defaultProfileSections(), user, courses), nil
	}

	var sections models.PostSections
	var page *models.Page
	if h.pageService != nil {
		if loaded, err := h.pageService.GetByPathAny("/profile"); err == nil && loaded != nil {
			page = loaded
			sections = cloneSections(page.Sections)
		}
	}

	if len(sections) == 0 {
		sections = defaultProfileSections()
	}

	sections = upgradeProfileCourseSections(sections)

	return applyProfileUserContext(sections, user, courses), page
}

func cloneSections(source models.PostSections) models.PostSections {
	if len(source) == 0 {
		return nil
	}

	cloned := make(models.PostSections, len(source))
	for i := range source {
		section := source[i]
		if len(section.Elements) > 0 {
			elements := make([]models.SectionElement, len(section.Elements))
			for j := range section.Elements {
				elem := section.Elements[j]
				if content, ok := elem.Content.(map[string]interface{}); ok {
					copied := make(map[string]interface{}, len(content))
					for key, value := range content {
						copied[key] = value
					}
					elem.Content = copied
				}
				elements[j] = elem
			}
			section.Elements = elements
		}
		cloned[i] = section
	}
	return cloned
}

func upgradeProfileCourseSections(sections models.PostSections) models.PostSections {
	if len(sections) == 0 {
		return sections
	}

	upgraded := make(models.PostSections, len(sections))
	for i := range sections {
		section := sections[i]
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))

		if sectionType == "courses_list" {
			if strings.TrimSpace(section.Mode) == "" {
				section.Mode = constants.CourseListModeCatalog
			}
			upgraded[i] = section
			continue
		}

		legacyIndex := -1
		legacyEmpty := ""
		for idx, elem := range section.Elements {
			elemType := strings.TrimSpace(strings.ToLower(elem.Type))
			if elemType == "profile_courses" {
				legacyIndex = idx
				if content, ok := elem.Content.(map[string]interface{}); ok {
					if msg, ok := content["empty_message"].(string); ok {
						legacyEmpty = strings.TrimSpace(msg)
					}
				}
				break
			}
		}

		if legacyIndex >= 0 {
			section.Type = "courses_list"
			section.Mode = constants.CourseListModeOwned
			section.Elements = []models.SectionElement{
				{
					ID:      section.ID + "-courses",
					Type:    "courses_list:owned",
					Order:   1,
					Content: ownedCourseSectionData{EmptyMessage: legacyEmpty},
				},
			}
			upgraded[i] = section
			continue
		}

		upgraded[i] = section
	}

	return upgraded
}

func applyProfileUserContext(sections models.PostSections, user *models.User, courses []models.UserCoursePackage) models.PostSections {
	username := ""
	email := ""
	role := "user"
	if user != nil {
		if trimmed := strings.TrimSpace(user.Username); trimmed != "" {
			username = trimmed
		}
		if trimmed := strings.TrimSpace(user.Email); trimmed != "" {
			email = trimmed
		}
		if trimmed := strings.TrimSpace(string(user.Role)); trimmed != "" {
			role = trimmed
		}
	}

	for i := range sections {
		section := &sections[i]
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))

		applyProfileTabSetting(section)

		if sectionType == "courses_list" && strings.EqualFold(strings.TrimSpace(strings.ToLower(section.Mode)), constants.CourseListModeOwned) {
			data := extractOwnedCourseSectionData(*section)
			data.Courses = cloneUserCoursePackages(courses)
			section.Elements = []models.SectionElement{
				{
					ID:      section.ID + "-courses",
					Type:    "courses_list:owned",
					Order:   1,
					Content: data,
				},
			}
			continue
		}

		elements := section.Elements
		for j := range elements {
			element := &elements[j]
			typeKey := strings.TrimSpace(strings.ToLower(element.Type))
			switch typeKey {
			case "profile_account_details":
				content := ensureContentMap(element)
				content["action"] = "/api/v1/profile"
				content["username"] = username
				content["email"] = email
				content["role"] = role
			case "profile_security":
				content := ensureContentMap(element)
				content["action"] = "/api/v1/profile/password"
				content["username"] = username
			}
		}
		section.Elements = elements
	}

	return sections
}

func applyProfileTabSetting(section *models.Section) {
	if section == nil {
		return
	}

	id := strings.TrimSpace(strings.ToLower(section.ID))
	tab := ""
	switch id {
	case "profile-account":
		tab = "account"
	case "profile-courses":
		tab = "courses"
	case "profile-security":
		tab = "security"
	}

	if tab == "" {
		return
	}

	if section.Settings == nil {
		section.Settings = map[string]interface{}{}
	}
	if existing, ok := section.Settings["profile_tab"].(string); ok && strings.TrimSpace(existing) != "" {
		return
	}

	section.Settings["profile_tab"] = tab
}

func buildProfileTabs(h *TemplateHandler, sections models.PostSections) []profileTab {
	if h == nil || len(sections) == 0 {
		return nil
	}

	var accountHTML string
	var securityHTML string
	var coursesHTML string

	for i := range sections {
		section := sections[i]
		sectionType := strings.TrimSpace(strings.ToLower(section.Type))

		if sectionType == "courses_list" && coursesHTML == "" {
			coursesHTML = h.renderCoursesListSection(pageViewClassPrefix, section)
		}

		for j := range section.Elements {
			elem := section.Elements[j]
			elemType := strings.TrimSpace(strings.ToLower(elem.Type))
			switch elemType {
			case "profile_account_details":
				if accountHTML == "" {
					if html, _ := h.renderSectionElement(pageViewClassPrefix, elem); html != "" {
						accountHTML = html
					}
				}
			case "profile_security":
				if securityHTML == "" {
					if html, _ := h.renderSectionElement(pageViewClassPrefix, elem); html != "" {
						securityHTML = html
					}
				}
			}
		}
	}

	tabs := make([]profileTab, 0, 3)

	if accountHTML != "" {
		tabs = append(tabs, profileTab{
			ID:          "account",
			Label:       "Account settings",
			Description: "Update your display name and email for this profile.",
			Content:     template.HTML(accountHTML),
		})
	}

	if coursesHTML != "" {
		tabs = append(tabs, profileTab{
			ID:          "courses",
			Label:       "My courses",
			Description: "Learning packages currently available to your account.",
			Content:     template.HTML(coursesHTML),
		})
	}

	if securityHTML != "" {
		tabs = append(tabs, profileTab{
			ID:          "security",
			Label:       "Account security",
			Description: "Password and sign-in safety controls.",
			Content:     template.HTML(securityHTML),
		})
	}

	return tabs
}

func ensureContentMap(element *models.SectionElement) map[string]interface{} {
	if element == nil {
		return map[string]interface{}{}
	}
	if content, ok := element.Content.(map[string]interface{}); ok && content != nil {
		return content
	}
	content := map[string]interface{}{}
	element.Content = content
	return content
}

func defaultProfileSections() models.PostSections {
	sections := models.PostSections{
		{
			ID:       "profile-account",
			Type:     "profile_account_details",
			Order:    1,
			Settings: map[string]interface{}{"profile_tab": "account"},
			Elements: []models.SectionElement{
				{
					ID:    "profile-account",
					Type:  "profile_account_details",
					Order: 1,
					Content: map[string]interface{}{
						"title":        "Account details",
						"description":  "The information below appears in comments and author bylines.",
						"button_label": "Save changes",
					},
				},
			},
		},
		{
			ID:       "profile-courses",
			Type:     "courses_list",
			Title:    "Courses",
			Order:    2,
			Mode:     constants.CourseListModeOwned,
			Settings: map[string]interface{}{"profile_tab": "courses"},
		},
		{
			ID:       "profile-security",
			Type:     "profile_security",
			Order:    3,
			Settings: map[string]interface{}{"profile_tab": "security"},
			Elements: []models.SectionElement{
				{
					ID:    "profile-security",
					Type:  "profile_security",
					Order: 1,
					Content: map[string]interface{}{
						"title":        "Security",
						"description":  "Change your password regularly and review connected devices.",
						"button_label": "Update password",
					},
				},
			},
		},
	}

	return sections
}

func (h *TemplateHandler) RenderCourse(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		redirectTo := url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, "/login?redirect="+redirectTo)
		return
	}

	if h.coursePackageSvc == nil {
		h.renderError(c, http.StatusServiceUnavailable, "Courses unavailable", "Course access is not configured.")
		return
	}

	identifier := strings.TrimSpace(c.Param("slug"))
	if identifier == "" {
		h.renderError(c, http.StatusNotFound, "Course not found", "Requested course could not be found.")
		return
	}

	course, err := h.coursePackageSvc.GetForUserByIdentifier(identifier, user.ID)
	if err == nil && course == nil {
		err = fmt.Errorf("course package was nil without error")
	}
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			h.renderError(c, http.StatusNotFound, "Course not found", "The course is unavailable or your access has expired.")
			return
		case courseservice.IsValidationError(err):
			h.renderError(c, http.StatusBadRequest, "Course unavailable", err.Error())
			return
		default:
			logger.Error(err, "Failed to load course for user", map[string]interface{}{"course_identifier": identifier, "user_id": user.ID})
			h.renderError(c, http.StatusInternalServerError, "Course unavailable", "We couldn't load this course right now.")
			return
		}
	}

	pkg := course.Package
	slug := strings.TrimSpace(pkg.Slug)
	if _, parseErr := strconv.ParseUint(identifier, 10, 64); parseErr == nil && slug != "" && !strings.EqualFold(slug, identifier) {
		c.Redirect(http.StatusMovedPermanently, fmt.Sprintf("/courses/%s", slug))
		return
	}

	title := strings.TrimSpace(pkg.Title)
	if title == "" {
		title = "Course"
	}
	summary := strings.TrimSpace(pkg.Summary)
	description := summary
	if description == "" {
		description = strings.TrimSpace(pkg.Description)
	}

	var sectionScripts []string
	for topicIndex := range pkg.Topics {
		topic := &pkg.Topics[topicIndex]
		for stepIndex := range topic.Steps {
			step := &topic.Steps[stepIndex]
			switch step.StepType {
			case models.CourseTopicStepTypeVideo:
				if step.Video == nil {
					continue
				}
				sections := step.Video.Sections
				if len(sections) == 0 {
					step.Video.SectionsHTML = ""
					continue
				}
				html, scripts := h.renderSectionsWithPrefix(sections, "course-player")
				if html != "" {
					step.Video.SectionsHTML = string(html)
				} else {
					step.Video.SectionsHTML = ""
				}
				sectionScripts = appendScripts(sectionScripts, scripts)
			case models.CourseTopicStepTypeContent:
				if step.Content == nil {
					continue
				}
				sections := step.Content.Sections
				if len(sections) == 0 {
					step.Content.SectionsHTML = ""
					continue
				}
				html, scripts := h.renderSectionsWithPrefix(sections, "course-player")
				if html != "" {
					step.Content.SectionsHTML = string(html)
				} else {
					step.Content.SectionsHTML = ""
				}
				sectionScripts = appendScripts(sectionScripts, scripts)
			}
		}
	}

	payload, err := json.Marshal(course)
	if err != nil {
		logger.Error(err, "Failed to serialise course", map[string]interface{}{"course_identifier": identifier, "user_id": user.ID})
		h.renderError(c, http.StatusInternalServerError, "Course unavailable", "We couldn't prepare this course for viewing.")
		return
	}
	payload = bytes.ReplaceAll(payload, []byte("</"), []byte("<\\/"))

	canonicalPath := fmt.Sprintf("/courses/%s", slug)
	if slug == "" {
		canonicalPath = fmt.Sprintf("/courses/%d", pkg.ID)
	}
	canonical := h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath)

	lessonCount := 0
	for _, topic := range pkg.Topics {
		lessonCount += len(topic.Steps)
	}

	scripts := appendScripts([]string{"/static/js/course-player.js"}, sectionScripts)

	pageTitle := strings.TrimSpace(pkg.MetaTitle)
	if pageTitle == "" {
		pageTitle = title
	}
	pageDescription := strings.TrimSpace(pkg.MetaDescription)
	if pageDescription == "" {
		pageDescription = description
	}

	courseEndpoint := fmt.Sprintf("/api/v1/courses/packages/%s", slug)
	if slug == "" {
		courseEndpoint = fmt.Sprintf("/api/v1/courses/packages/%d", pkg.ID)
	}

	data := gin.H{
		"Course":              course,
		"CourseJSON":          template.JS(string(payload)),
		"CourseEndpoint":      courseEndpoint,
		"CourseTestEndpoint":  "/api/v1/courses/tests",
		"CourseTopicCount":    len(pkg.Topics),
		"CourseLessonCount":   lessonCount,
		"CourseCanonicalPath": canonicalPath,
		"Scripts":             scripts,
		"Canonical":           canonical,
		"NoIndex":             true,
	}

	h.renderTemplate(c, "course", pageTitle, pageDescription, data)
}

func (h *TemplateHandler) RenderArchive(c *gin.Context) {
	if !h.ensureArchiveAvailable(c) {
		return
	}

	directories, err := h.archiveDirectorySvc.ListPublishedTree()
	if err != nil {
		logger.Error(err, "Failed to load archive tree", nil)
		h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't load the archive directory tree right now.")
		return
	}

	title := "Resource archive"
	description := "Browse shared documents and downloadable files from a single archive."

	data := gin.H{
		"Directories":    directories,
		"ArchiveEnabled": true,
		"Styles":         []string{"/static/css/sections/archive.css"},
	}

	h.renderTemplate(c, "archive", title, description, data)
}

func (h *TemplateHandler) RenderArchivePath(c *gin.Context) {
	if !h.ensureArchiveAvailable(c) {
		return
	}

	pathValue := strings.Trim(c.Param("path"), "/")
	if pathValue == "" {
		h.RenderArchive(c)
		return
	}

	if strings.HasPrefix(pathValue, "files/") {
		filePath := strings.TrimPrefix(pathValue, "files/")
		if strings.TrimSpace(filePath) == "" {
			h.renderError(c, http.StatusNotFound, "File not found", "The requested file could not be located.")
			return
		}

		h.renderArchiveFile(c, filePath)
		return
	}

	h.renderArchiveDirectory(c, pathValue)
}

func (h *TemplateHandler) renderArchiveDirectory(c *gin.Context, pathValue string) {
	directory, err := h.archiveDirectorySvc.GetByPath(pathValue, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			h.renderError(c, http.StatusNotFound, "Directory not found", "The requested directory could not be located.")
			return
		}
		logger.Error(err, "Failed to load archive directory", map[string]interface{}{"path": pathValue})
		h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't load this directory right now.")
		return
	}

	files, err := h.archiveFileSvc.ListByDirectory(directory.ID, false)
	if err != nil {
		logger.Error(err, "Failed to list archive files", map[string]interface{}{"directory": directory.Path})
		h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't list the files for this directory.")
		return
	}

	children, err := h.archiveDirectorySvc.ListByParent(&directory.ID, false)
	if err != nil {
		logger.Error(err, "Failed to list archive subdirectories", map[string]interface{}{"directory": directory.Path})
	}

	breadcrumbs, err := h.archiveDirectorySvc.BuildBreadcrumbs(pathValue, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			h.renderError(c, http.StatusNotFound, "Directory not found", "The requested directory could not be located.")
		} else {
			logger.Error(err, "Failed to build archive breadcrumbs", map[string]interface{}{"path": pathValue})
			h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't prepare navigation for this directory.")
		}
		return
	}

	trimmedName := strings.TrimSpace(directory.Name)
	title := trimmedName
	if title == "" {
		title = "Archive directory"
	}
	description := "Browse files available in this directory."
	if trimmedName != "" {
		description = fmt.Sprintf("Browse files available in %s.", trimmedName)
	}

	canonicalPath := "/archive/" + directory.Path
	canonical := h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath)

	data := gin.H{
		"Directory":      directory,
		"Files":          files,
		"Children":       children,
		"Breadcrumbs":    breadcrumbs,
		"ArchiveEnabled": true,
		"Canonical":      canonical,
		"Styles":         []string{"/static/css/sections/archive.css"},
	}

	h.renderTemplate(c, "archive-directory", title, description, data)
}

func (h *TemplateHandler) renderArchiveFile(c *gin.Context, pathValue string) {
	file, err := h.archiveFileSvc.GetByPath(pathValue, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrFileNotFound) {
			h.renderError(c, http.StatusNotFound, "File not found", "The requested file could not be located.")
			return
		}

		// If file still missing type/size, try to refresh metadata in background (non-blocking)
		if strings.TrimSpace(file.FileType) == "" || file.FileSize == 0 || strings.TrimSpace(file.MimeType) == "" {
			go func(id uint) {
				// Call Update with an empty request to trigger re-inference without changing other fields
				if _, err := h.archiveFileSvc.Update(id, models.UpdateArchiveFileRequest{}); err != nil {
					logger.Warn("Failed to refresh archive file metadata", map[string]interface{}{"id": id, "error": err.Error()})
				}
			}(file.ID)
		}
		logger.Error(err, "Failed to load archive file", map[string]interface{}{"path": pathValue})
		h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't load this file right now.")
		return
	}

	segments := strings.Split(pathValue, "/")
	if len(segments) < 2 {
		h.renderError(c, http.StatusBadRequest, "Invalid file path", "Files must belong to a directory.")
		return
	}

	directoryPath := strings.Join(segments[:len(segments)-1], "/")
	directory, err := h.archiveDirectorySvc.GetByPath(directoryPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			h.renderError(c, http.StatusNotFound, "Directory not found", "The parent directory could not be located.")
			return
		}
		logger.Error(err, "Failed to load archive directory for file", map[string]interface{}{"path": directoryPath})
		h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't load the parent directory for this file.")
		return
	}

	breadcrumbs, err := h.archiveDirectorySvc.BuildBreadcrumbs(directoryPath, false)
	if err != nil {
		if errors.Is(err, archiveservice.ErrDirectoryNotFound) {
			h.renderError(c, http.StatusNotFound, "Directory not found", "The parent directory could not be located.")
		} else {
			logger.Error(err, "Failed to build archive breadcrumbs", map[string]interface{}{"path": directoryPath})
			h.renderError(c, http.StatusInternalServerError, "Archive unavailable", "We couldn't prepare navigation for this file.")
		}
		return
	}
	breadcrumbs = append(breadcrumbs, models.ArchiveBreadcrumb{Name: strings.TrimSpace(file.Name), Path: file.Path})

	title := strings.TrimSpace(file.Name)
	if title == "" {
		title = "Archive file"
	}
	description := strings.TrimSpace(file.Description)
	if description == "" {
		description = fmt.Sprintf("Download or preview %s.", strings.TrimSpace(file.Name))
	}

	canonicalPath := "/archive/files/" + file.Path
	canonical := h.ensureAbsoluteURL(h.config.SiteURL, canonicalPath)

	data := gin.H{
		"File":           file,
		"Directory":      directory,
		"Breadcrumbs":    breadcrumbs,
		"ArchiveEnabled": true,
		"Canonical":      canonical,
		"Styles":         []string{"/static/css/sections/archive.css"},
	}

	h.renderTemplate(c, "archive-file", title, description, data)
}

func (h *TemplateHandler) RenderAdmin(c *gin.Context) {
	user, ok := h.currentUser(c)
	if !ok {
		redirectTo := url.QueryEscape(c.Request.URL.RequestURI())
		c.Redirect(http.StatusFound, "/login?redirect="+redirectTo)
		return
	}

	hasAccess := authorization.RoleHasPermission(user.Role, authorization.PermissionManageAllContent) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionManageSettings) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionManageUsers) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionManageBackups) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionManageThemes) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionManagePlugins) ||
		authorization.RoleHasPermission(user.Role, authorization.PermissionModerateComments)

	if !hasAccess {
		h.renderError(c, http.StatusForbidden, "403 - Forbidden", "Administrator access required")
		return
	}

	sectionJSON, elementJSON := h.builderDefinitionsJSON()
	blogEnabled := h.blogEnabled()
	coursesEnabled := h.coursesEnabled()
	forumEnabled := h.forumEnabled()
	archiveEnabled := h.archiveEnabled()

	adminEndpoints := gin.H{
		"Stats":               "/api/v1/admin/stats",
		"Pages":               "/api/v1/admin/pages",
		"PageSectionsPadding": "/api/v1/admin/pages/sections/padding",
		"PageBuilder":         "/api/v1/admin/pages/:id/builder",
		"DuplicatePage":       "/api/v1/admin/pages/:id/duplicate",
		"ReorderSections":     "/api/v1/admin/pages/:id/sections/reorder",
		"AddSection":          "/api/v1/admin/pages/:id/sections",
		"UpdateSection":       "/api/v1/admin/pages/:id/sections/:sectionId",
		"DeleteSection":       "/api/v1/admin/pages/:id/sections/:sectionId",
		"DuplicateSection":    "/api/v1/admin/pages/:id/sections/:sectionId/duplicate",
		"PageTemplates":       "/api/v1/admin/pages/templates",
		"CreateFromTemplate":  "/api/v1/admin/pages/templates/:templateId",
		"PreviewPage":         "/api/v1/admin/pages/:id/preview",
		"ValidateSlug":        "/api/v1/admin/pages/validate-slug",
		"PageBuilderConfig":   "/api/v1/admin/pages/builder/config",
		"SiteSettings":        "/api/v1/admin/settings/site",
		"Homepage":            "/api/v1/admin/settings/homepage",
		"FaviconUpload":       "/api/v1/admin/settings/favicon",
		"LogoUpload":          "/api/v1/admin/settings/logo",
		"Upload":              "/api/v1/admin/upload",
		"Uploads":             "/api/v1/admin/uploads",
		"UploadDelete":        "/api/v1/admin/uploads",
		"UploadRename":        "/api/v1/admin/uploads/rename",
		"Themes":              "/api/v1/admin/themes",
		"Plugins":             "/api/v1/admin/plugins",
		"SocialLinks":         "/api/v1/admin/social-links",
		"Fonts":               "/api/v1/admin/settings/fonts",
		"MenuItems":           "/api/v1/admin/menu-items",
		"Users":               "/api/v1/admin/users",
		"Advertising":         "/api/v1/admin/settings/advertising",
		"BackupSettings":      "/api/v1/admin/backups/settings",
		"BackupExport":        "/api/v1/admin/backups/export",
		"BackupImport":        "/api/v1/admin/backups/import",
	}

	if coursesEnabled {
		adminEndpoints["CourseVideos"] = "/api/v1/admin/courses/videos"
		adminEndpoints["CourseContents"] = "/api/v1/admin/courses/contents"
		adminEndpoints["CourseTopics"] = "/api/v1/admin/courses/topics"
		adminEndpoints["CourseTests"] = "/api/v1/admin/courses/tests"
		adminEndpoints["CoursePackages"] = "/api/v1/admin/courses/packages"
	}

	if blogEnabled {
		adminEndpoints["Posts"] = "/api/v1/admin/posts"
		adminEndpoints["Categories"] = "/api/v1/admin/categories"
		adminEndpoints["CategoriesIndex"] = "/api/v1/categories"
		adminEndpoints["Comments"] = "/api/v1/admin/comments"
		adminEndpoints["Tags"] = "/api/v1/tags"
		adminEndpoints["TagsAdmin"] = "/api/v1/admin/tags"
	}

	if forumEnabled {
		adminEndpoints["ForumQuestions"] = "/api/v1/forum/questions"
		adminEndpoints["ForumQuestionsManage"] = "/api/v1/admin/forum/questions"
		adminEndpoints["ForumAnswers"] = "/api/v1/forum/answers"
		adminEndpoints["ForumCategories"] = "/api/v1/admin/forum/categories"
	}

	if archiveEnabled {
		adminEndpoints["ArchiveDirectories"] = "/api/v1/admin/archive/directories"
		adminEndpoints["ArchiveFiles"] = "/api/v1/admin/archive/files"
		adminEndpoints["ArchiveTree"] = "/api/v1/admin/archive/directories?tree=1"
	}

	h.renderTemplate(c, "admin", "Admin dashboard", "Monitor site activity, review content performance, and manage published resources in one place.", gin.H{
		"Layout":                 "admin_base.html",
		"Styles":                 []string{"/static/css/admin.css", "/static/css/admin/anchor-picker.css"},
		"Scripts":                h.builderScripts(),
		"SectionDefinitionsJSON": sectionJSON,
		"ElementDefinitionsJSON": elementJSON,
		"BuilderConfigJSON":      h.builderConfigJSON(),
		"AdminEndpoints":         adminEndpoints,
		"BlogEnabled":            blogEnabled,
		"CoursesEnabled":         coursesEnabled,
		"ForumEnabled":           forumEnabled,
		"ArchiveEnabled":         archiveEnabled,
		"LanguageFeatureEnabled": h.languageService != nil,
		"NoIndex":                true,
	})
}

func (h *TemplateHandler) builderScripts() []string {
	scripts := []string{
		"/static/js/admin/utils.js",
		"/static/js/admin/elements/registry.js",
	}

	var assets theme.BuilderAssets
	if h.themeManager != nil {
		if active := h.themeManager.Active(); active != nil {
			assets = active.BuilderAssets()
		}
	}

	if len(assets.ElementScripts) > 0 {
		scripts = append(scripts, assets.ElementScripts...)
	} else {
		scripts = append(scripts,
			"/static/js/admin/elements/paragraph.js",
			"/static/js/admin/elements/image.js",
			"/static/js/admin/elements/image-group.js",
			"/static/js/admin/elements/feature-item.js",
			"/static/js/admin/elements/file-group.js",
			"/static/js/admin/elements/list.js",
			"/static/js/admin/elements/search.js",
			"/static/js/admin/elements/profile-account-details.js",
			"/static/js/admin/elements/profile-security.js",
		)
	}

	scripts = append(scripts, "/static/js/admin/sections/registry.js")

	if len(assets.SectionScripts) > 0 {
		scripts = append(scripts, assets.SectionScripts...)
	}

	scripts = append(scripts,
		"/static/js/admin/builder/section-state.js",
		"/static/js/admin/builder/section-view.js",
		"/static/js/admin/builder/section-events.js",
		"/static/js/admin/builder/section-builder.js",
		// Modular components (must be loaded before admin.js)
		"/static/js/admin/formatters.js",
		"/static/js/admin/api-client.js",
		"/static/js/admin/ui-manager.js",
		"/static/js/admin/analytics.js",
		"/static/js/admin/tables/table-manager.js",
		"/static/js/admin/forms/form-handler.js",
		// Layout and core
		"/static/js/admin/media-library.js",
		"/static/js/admin/layout.js",
		"/static/js/admin/panels/core.js",
		"/static/js/section-builder.js",
		"/static/js/admin.js",
	)

	scripts = append(scripts, "/static/js/admin/forum.js")
	scripts = append(scripts, "/static/js/admin/archive.js")

	return scripts
}

func (h *TemplateHandler) builderDefinitionsJSON() (template.JS, template.JS) {
	sectionDefs := theme.DefaultSectionDefinitions()
	baseSectionDefs := sectionDefs
	elementDefs := theme.DefaultElementDefinitions()

	// Try to use dynamic metadata from section registry first
	if metadata := h.SectionMetadata(); len(metadata) > 0 {
		if h.themeManager != nil {
			if active := h.themeManager.Active(); active != nil {
				if defs := active.SectionDefinitions(); len(defs) > 0 {
					baseSectionDefs = defs
				}
			}
		}

		sectionDefs = make(map[string]theme.SectionDefinition, len(metadata))
		for _, meta := range metadata {
			supportsElements := true
			supportsHeaderImage := false

			// Determine if section supports elements based on schema
			if meta.Schema != nil {
				if _, hasLimit := meta.Schema["limit"]; hasLimit {
					supportsElements = false
				}
			}

			base := baseSectionDefs[strings.TrimSpace(strings.ToLower(meta.Type))]
			def := base
			def.Type = meta.Type
			def.Label = meta.Name
			def.Description = meta.Description
			def.Order = 0
			def.SupportsElements = &supportsElements
			def.SupportsHeaderImage = &supportsHeaderImage

			// Convert schema to settings
			if len(meta.Schema) > 0 {
				if def.Settings == nil {
					def.Settings = make(map[string]theme.SectionSettingDefinition)
				}
				for key, schemaValue := range meta.Schema {
					setting := theme.SectionSettingDefinition{}

					if schemaMap, ok := schemaValue.(map[string]interface{}); ok {
						if label, ok := schemaMap["label"].(string); ok {
							setting.Label = label
						}
						if min, ok := schemaMap["min"].(int); ok {
							setting.Min = &min
						}
						if max, ok := schemaMap["max"].(int); ok {
							setting.Max = &max
						}
						if defaultVal, ok := schemaMap["default"].(int); ok {
							setting.Default = &defaultVal
						}
					}

					def.Settings[key] = setting
				}
			}

			sectionDefs[meta.Type] = def
		}
	} else if h.themeManager != nil {
		if active := h.themeManager.Active(); active != nil {
			if defs := active.SectionDefinitions(); len(defs) > 0 {
				sectionDefs = defs
			}
			if defs := active.ElementDefinitions(); len(defs) > 0 {
				elementDefs = defs
			}
		}
	}

	if !h.blogEnabled() {
		delete(sectionDefs, "posts_list")
		delete(sectionDefs, "categories_list")
	}

	if !h.coursesEnabled() {
		delete(sectionDefs, "courses_list")
	}

	sectionJSON, err := json.Marshal(sectionDefs)
	if err != nil {
		logger.Error(err, "Failed to marshal section definitions", nil)
		sectionJSON = []byte("{}")
	}
	elementJSON, err := json.Marshal(elementDefs)
	if err != nil {
		logger.Error(err, "Failed to marshal element definitions", nil)
		elementJSON = []byte("{}")
	}

	return template.JS(sectionJSON), template.JS(elementJSON)
}

func (h *TemplateHandler) builderConfigJSON() template.JS {
	defaultPadding := constants.DefaultSectionPadding
	if h.themeManager != nil {
		if active := h.themeManager.Active(); active != nil {
			defaultPadding = active.DefaultSectionPadding()
		}
	}

	config := gin.H{
		"paddingOptions":        constants.SectionPaddingOptions(),
		"marginOptions":         constants.SectionMarginOptions(),
		"defaultSectionPadding": defaultPadding,
		"defaultSectionMargin":  constants.DefaultSectionMargin,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		logger.Error(err, "Failed to marshal builder config", nil)
		configJSON = []byte("{}")
	}

	return template.JS(configJSON)
}

func (h *TemplateHandler) buildPagination(current, total int, buildURL func(page int) string) gin.H {
	if total <= 1 || buildURL == nil {
		return nil
	}

	data := gin.H{
		"CurrentPage": current,
		"TotalPages":  total,
	}

	if current > 1 {
		data["PrevURL"] = buildURL(current - 1)
	}

	if current < total {
		data["NextURL"] = buildURL(current + 1)
	}

	return data
}

func (h *TemplateHandler) renderError(c *gin.Context, status int, title, msg string) {
	site := h.siteSettings()

	data := gin.H{
		"Title":      title,
		"error":      msg,
		"StatusCode": status,
		"Site": gin.H{
			"Name":        site.Name,
			"Description": site.Description,
			"URL":         site.URL,
			"Favicon":     site.Favicon,
			"FaviconType": site.FaviconType,
			"Logo":        site.Logo,
		},
	}

	tmpl, err := h.templateClone()
	if err != nil {
		logger.Error(err, "Failed to load error template", nil)
		c.JSON(status, gin.H{"error": msg})
		return
	}

	errorTmpl := tmpl.Lookup("error.html")
	if errorTmpl == nil {
		logger.Error(nil, "Error template missing", nil)
		c.JSON(status, gin.H{"error": msg})
		return
	}

	output, err := h.executeTemplate(errorTmpl, data)
	if err != nil {
		logger.Error(err, "Failed to render error template", nil)
		c.JSON(status, gin.H{"error": msg})
		return
	}

	c.Data(status, "text/html; charset=utf-8", output)
}

func (h *TemplateHandler) RenderErrorPage(c *gin.Context, status int, title, msg string) {
	h.renderError(c, status, title, msg)
}
