package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostService struct {
	postRepo     repository.PostRepository
	tagRepo      repository.TagRepository
	categoryRepo repository.CategoryRepository
	commentRepo  repository.CommentRepository
	cache        *cache.Cache
	settingRepo  repository.SettingRepository
}

var ErrPostNotPublished = errors.New("post is not published")

func (s *PostService) invalidateTagCaches() {
	if s.cache == nil {
		return
	}
	s.cache.Delete("tags:all")
	s.cache.Delete("tags:used")
}

func (s *PostService) handleTagChanges() {
	s.invalidateTagCaches()
	s.scheduleUnusedTagCleanup()
}

func (s *PostService) scheduleUnusedTagCleanup() {
	if s.tagRepo == nil {
		return
	}

	go func() {
		now := time.Now().UTC()
		if err := s.tagRepo.MarkUnused(now); err != nil {
			logger.Error(err, "Failed to mark unused tags", nil)
			return
		}

		retention := s.unusedTagRetentionDuration()
		cutoff := now.Add(-retention)
		if deleted, err := s.tagRepo.DeleteUnusedBefore(cutoff); err != nil {
			logger.Error(err, "Failed to delete unused tags", nil)
		} else if deleted > 0 {
			logger.Info("Removed unused tags", map[string]interface{}{"count": deleted})
		}
	}()
}

func (s *PostService) unusedTagRetentionDuration() time.Duration {
	hours := DefaultUnusedTagRetentionHours

	if s.settingRepo != nil {
		setting, err := s.settingRepo.Get(settingKeyTagRetentionHours)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Error(err, "Failed to load unused tag retention setting", nil)
			}
		} else {
			value := strings.TrimSpace(setting.Value)
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed > 0 {
				hours = parsed
			} else if parseErr != nil {
				logger.Error(parseErr, "Invalid unused tag retention value", map[string]interface{}{"value": value})
			}
		}
	}

	if hours <= 0 {
		hours = DefaultUnusedTagRetentionHours
	}

	return time.Duration(hours) * time.Hour
}

func NewPostService(
	postRepo repository.PostRepository,
	tagRepo repository.TagRepository,
	categoryRepo repository.CategoryRepository,
	commentRepo repository.CommentRepository,
	cacheService *cache.Cache,
	settingRepo repository.SettingRepository,
) *PostService {
	return &PostService{
		postRepo:     postRepo,
		tagRepo:      tagRepo,
		categoryRepo: categoryRepo,
		commentRepo:  commentRepo,
		cache:        cacheService,
		settingRepo:  settingRepo,
	}
}

func (s *PostService) Create(req models.CreatePostRequest, authorID uint) (*models.Post, error) {
	if req.Title == "" {
		return nil, errors.New("post title is required")
	}

	slug := utils.GenerateSlug(req.Title)

	exists, err := s.postRepo.ExistsBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to check post existence: %w", err)
	}
	if exists {
		return nil, errors.New("post with this title already exists")
	}

	sections, err := s.prepareSections(req.Sections)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare sections: %w", err)
	}

	content := req.Content
	if len(sections) > 0 && content == "" {
		content = s.generateContentFromSections(sections)
	}

	categoryID := req.CategoryID
	if categoryID == 0 && s.categoryRepo != nil {
		defaultCategory, err := s.categoryRepo.GetBySlug(defaultCategorySlug)
		if err != nil {
			return nil, fmt.Errorf("failed to assign default category: %w", err)
		}
		categoryID = defaultCategory.ID
	}

	post := &models.Post{
		Title:       req.Title,
		Slug:        slug,
		Description: req.Description,
		Content:     content,
		Excerpt:     req.Excerpt,
		FeaturedImg: req.FeaturedImg,
		Published:   req.Published,
		AuthorID:    authorID,
		CategoryID:  categoryID,
		Sections:    sections,
		Template:    s.getTemplate(req.Template),
	}

	now := time.Now().UTC()
	post.Published, post.PublishAt, post.PublishedAt = normalizePublicationState(post.Published, req.PublishAt.Or(nil), now)

	if req.TagNames != nil {
		if len(req.TagNames) == 0 {
			post.Tags = []models.Tag{}
		} else {
			tags, err := s.getOrCreateTags(req.TagNames)
			if err != nil {
				return nil, err
			}
			post.Tags = tags
		}
	}

	if err := s.postRepo.Create(post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	s.handleTagChanges()
	if s.cache != nil {
		s.cache.InvalidatePostsCache()
	}

	return s.postRepo.GetByID(post.ID)
}

func (s *PostService) ExistsBySlug(slug string) (bool, error) {
	cleaned := strings.TrimSpace(slug)
	if cleaned == "" {
		return false, errors.New("slug is required")
	}
	if s.postRepo == nil {
		return false, errors.New("post repository not configured")
	}

	return s.postRepo.ExistsBySlug(cleaned)
}

func (s *PostService) Update(id uint, req models.UpdatePostRequest, userID uint, canManageAll bool) (*models.Post, error) {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !canManageAll && post.AuthorID != userID {
		return nil, errors.New("unauthorized")
	}

	if req.Title != nil {
		post.Title = *req.Title
		post.Slug = utils.GenerateSlug(*req.Title)
	}
	if req.Description != nil {
		post.Description = *req.Description
	}
	if req.Content != nil {
		post.Content = *req.Content
	}
	if req.Excerpt != nil {
		post.Excerpt = *req.Excerpt
	}
	if req.FeaturedImg != nil {
		post.FeaturedImg = *req.FeaturedImg
	}
	if req.Published != nil {
		post.Published = *req.Published
	}
	if req.CategoryID != nil {
		post.CategoryID = *req.CategoryID
	}
	if req.Template != nil {
		post.Template = s.getTemplate(*req.Template)
	}

	publishAtCandidate := req.PublishAt.Or(post.PublishAt)
	now := time.Now().UTC()
	post.Published, post.PublishAt, post.PublishedAt = normalizePublicationState(post.Published, publishAtCandidate, now)

	if req.Sections != nil {
		sections, err := s.prepareSections(*req.Sections)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare sections: %w", err)
		}
		post.Sections = sections

		if post.Content == "" || len(sections) > 0 {
			post.Content = s.generateContentFromSections(sections)
		}
	}

	if req.TagNames != nil {
		if len(req.TagNames) == 0 {
			post.Tags = []models.Tag{}
		} else {
			tags, err := s.getOrCreateTags(req.TagNames)
			if err != nil {
				return nil, err
			}
			post.Tags = tags
		}
	}

	if err := s.postRepo.Update(post); err != nil {
		return nil, err
	}

	s.handleTagChanges()
	if s.cache != nil {
		s.cache.InvalidatePost(id)
		s.cache.InvalidatePostsCache()
	}

	return s.postRepo.GetByID(post.ID)
}

func (s *PostService) prepareSections(sections []models.Section) (models.PostSections, error) {
	if len(sections) == 0 {
		return models.PostSections{}, nil
	}

	prepared := make(models.PostSections, 0, len(sections))

	for i, section := range sections {

		sectionType := strings.TrimSpace(section.Type)
		sectionType = strings.ToLower(sectionType)
		if sectionType == "" {
			sectionType = "standard"
		}
		switch sectionType {
		case "standard", "grid":
			if len(section.Elements) > 0 {
				preparedElements, err := s.prepareSectionElements(section.Elements)
				if err != nil {
					return nil, fmt.Errorf("section %d: %w", i, err)
				}
				section.Elements = preparedElements
			}
		case "hero":
			section.Elements = nil
		case "posts_list":
			section.Elements = nil
			limit := section.Limit
			if limit <= 0 {
				limit = constants.DefaultPostListSectionLimit
			}
			if limit > constants.MaxPostListSectionLimit {
				limit = constants.MaxPostListSectionLimit
			}
			section.Limit = limit
		default:
			return nil, fmt.Errorf("section %d: unknown type '%s'", i, sectionType)
		}

		if section.ID == "" {
			section.ID = uuid.New().String()
		}

		if section.Order == 0 {
			section.Order = i + 1
		}

		section.Type = sectionType

		prepared = append(prepared, section)
	}

	return prepared, nil
}

func (s *PostService) prepareSectionElements(elements []models.SectionElement) ([]models.SectionElement, error) {
	prepared := make([]models.SectionElement, 0, len(elements))

	for i, elem := range elements {

		if elem.ID == "" {
			elem.ID = uuid.New().String()
		}

		if elem.Order == 0 {
			elem.Order = i + 1
		}

		switch elem.Type {
		case "paragraph", "image", "image_group", "list", "search":

		default:
			return nil, fmt.Errorf("element %d: unknown type '%s'", i, elem.Type)
		}

		if elem.Content == nil {
			return nil, fmt.Errorf("element %d: content is required", i)
		}

		prepared = append(prepared, elem)
	}

	return prepared, nil
}

func (s *PostService) generateContentFromSections(sections models.PostSections) string {
	var content strings.Builder

	for _, section := range sections {
		title := strings.TrimSpace(section.Title)
		if title != "" {
			content.WriteString(title)
			content.WriteString("\n\n")
		}

		for _, elem := range section.Elements {
			if elem.Type == "paragraph" {
				if contentMap, ok := elem.Content.(map[string]interface{}); ok {
					if text, ok := contentMap["text"].(string); ok {
						content.WriteString(text)
						content.WriteString("\n\n")
					}
				}
				continue
			}

			if elem.Type == "list" {
				if contentMap, ok := elem.Content.(map[string]interface{}); ok {
					if items, ok := contentMap["items"].([]interface{}); ok {
						for _, item := range items {
							if text, ok := item.(string); ok && text != "" {
								content.WriteString(text)
								content.WriteString("\n")
							}
						}
						content.WriteString("\n")
					} else if strItems, ok := contentMap["items"].([]string); ok {
						for _, text := range strItems {
							if text != "" {
								content.WriteString(text)
								content.WriteString("\n")
							}
						}
						content.WriteString("\n")
					}
				}
			}
		}
	}

	return content.String()
}

func (s *PostService) getTemplate(template string) string {
	if template == "" {
		return "post"
	}
	return template
}

func (s *PostService) getOrCreateTags(tagNames []string) ([]models.Tag, error) {
	var tags []models.Tag
	seen := make(map[string]struct{})

	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		slug := utils.GenerateSlug(name)
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}

		tag, err := s.tagRepo.GetBySlug(slug)

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				tag = &models.Tag{
					Name: name,
					Slug: slug,
				}
				if err := s.tagRepo.Create(tag); err != nil {
					return nil, err
				}

				s.handleTagChanges()
			} else {
				return nil, err
			}
		}

		tags = append(tags, *tag)
	}

	return tags, nil
}

func (s *PostService) Delete(id uint, userID uint, canManageAll bool) error {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return err
	}

	if !canManageAll && post.AuthorID != userID {
		return errors.New("unauthorized")
	}

	if err := s.postRepo.Delete(id); err != nil {
		return err
	}

	s.handleTagChanges()
	if s.cache != nil {
		s.cache.InvalidatePost(id)
		s.cache.InvalidatePostsCache()
	}

	return nil
}

func (s *PostService) GetByID(id uint) (*models.Post, error) {

	if s.cache != nil {
		var post models.Post
		if err := s.cache.GetCachedPost(id, &post); err == nil {
			s.trackPostView(&post)
			return &post, nil
		}
	}

	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	s.trackPostView(post)

	return post, nil
}

func (s *PostService) GetBySlug(slug string) (*models.Post, error) {

	if s.cache != nil {
		var post models.Post
		cacheKey := fmt.Sprintf("post:slug:%s", slug)
		if err := s.cache.Get(cacheKey, &post); err == nil {
			s.trackPostView(&post)
			return &post, nil
		}
	}

	post, err := s.postRepo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	s.trackPostView(post)

	return post, nil
}

func (s *PostService) trackPostView(post *models.Post) {
	if post == nil {
		return
	}

	now := time.Now().UTC()
	if !post.Published {
		return
	}
	if post.PublishAt != nil && post.PublishAt.After(now) {
		return
	}

	post.Views++

	go func(postID uint) {
		if err := s.postRepo.IncrementViews(postID); err != nil {
			logger.Error(err, "Failed to increment post views", map[string]interface{}{"post_id": postID})
		}
	}(post.ID)

	if s.cache != nil {
		s.cache.CachePost(post.ID, post)
		if slug := strings.TrimSpace(post.Slug); slug != "" {
			cacheKey := fmt.Sprintf("post:slug:%s", slug)
			s.cache.Set(cacheKey, post, time.Hour)
		}
	}
}

func (s *PostService) GetAll(page, limit int, categoryID *uint, tagName *string, authorID *uint) ([]models.Post, int64, error) {
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("posts:page:%d:limit:%d", page, limit)
	if categoryID != nil {
		cacheKey += fmt.Sprintf(":cat:%d", *categoryID)
	}
	if tagName != nil {
		cacheKey += fmt.Sprintf(":tag:%s", *tagName)
	}
	if authorID != nil {
		cacheKey += fmt.Sprintf(":author:%d", *authorID)
	}

	if s.cache != nil {
		var result struct {
			Posts []models.Post
			Total int64
		}
		if err := s.cache.Get(cacheKey, &result); err == nil {
			return result.Posts, result.Total, nil
		}
	}

	published := true

	posts, total, err := s.postRepo.GetAll(offset, limit, categoryID, tagName, authorID, &published)
	if err != nil {
		return nil, 0, err
	}

	if s.cache != nil {
		result := struct {
			Posts []models.Post
			Total int64
		}{posts, total}
		s.cache.Set(cacheKey, result, 5*time.Minute)
	}

	return posts, total, nil
}

func (s *PostService) ListPublishedForSitemap() ([]models.Post, error) {
	if s.postRepo == nil {
		return nil, errors.New("post repository not configured")
	}

	return s.postRepo.GetAllPublished()
}

func (s *PostService) GetAllAdmin(page, limit int) ([]models.Post, int64, error) {
	offset := (page - 1) * limit
	return s.postRepo.GetAll(offset, limit, nil, nil, nil, nil)
}

func (s *PostService) fetchPostsByTag(tagSlug string, page, limit int) ([]models.Post, int64, error) {
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("posts:tag:%s:page:%d:limit:%d", tagSlug, page, limit)

	logger.Debug("Fetching posts by tag", map[string]interface{}{
		"tag_slug": tagSlug,
		"page":     page,
		"limit":    limit,
		"offset":   offset,
	})

	if s.cache != nil {
		var result struct {
			Posts []models.Post
			Total int64
		}
		if err := s.cache.Get(cacheKey, &result); err == nil {
			logger.Debug("Loaded posts by tag from cache", map[string]interface{}{
				"tag_slug": tagSlug,
				"page":     page,
				"limit":    limit,
				"total":    result.Total,
			})
			return result.Posts, result.Total, nil
		}
	}

	published := true

	posts, total, err := s.postRepo.GetAll(offset, limit, nil, &tagSlug, nil, &published)
	if err != nil {
		logger.Error(err, "Failed to load posts by tag", map[string]interface{}{
			"tag_slug": tagSlug,
			"page":     page,
			"limit":    limit,
			"offset":   offset,
		})
		return nil, 0, err
	}

	logger.Debug("Fetched posts by tag from repository", map[string]interface{}{
		"tag_slug": tagSlug,
		"page":     page,
		"limit":    limit,
		"total":    total,
	})

	if s.cache != nil {
		result := struct {
			Posts []models.Post
			Total int64
		}{posts, total}
		s.cache.Set(cacheKey, result, 5*time.Minute)
	}

	return posts, total, nil
}

func (s *PostService) GetPostsByTag(tagSlug string, page, limit int) ([]models.Post, int64, error) {
	_, posts, total, err := s.GetTagWithPosts(tagSlug, page, limit)
	return posts, total, err
}

func (s *PostService) GetTagWithPosts(tagSlug string, page, limit int) (*models.Tag, []models.Post, int64, error) {
	logger.Debug("Loading tag with posts", map[string]interface{}{
		"tag_slug": tagSlug,
		"page":     page,
		"limit":    limit,
	})

	tag, err := s.tagRepo.GetBySlug(tagSlug)
	if err != nil {
		logger.Error(err, "Failed to load tag by slug", map[string]interface{}{
			"tag_slug": tagSlug,
		})
		return nil, nil, 0, err
	}

	logger.Debug("Tag loaded", map[string]interface{}{
		"tag_id":   tag.ID,
		"tag_slug": tag.Slug,
	})

	posts, total, err := s.fetchPostsByTag(tag.Slug, page, limit)
	if err != nil {
		logger.Error(err, "Failed to load posts for tag", map[string]interface{}{
			"tag_slug": tag.Slug,
			"page":     page,
			"limit":    limit,
		})
		return nil, nil, 0, err
	}

	logger.Debug("Successfully loaded tag with posts", map[string]interface{}{
		"tag_id":   tag.ID,
		"tag_slug": tag.Slug,
		"total":    total,
	})

	return tag, posts, total, nil
}

func (s *PostService) GetTagBySlug(tagSlug string) (*models.Tag, error) {
	return s.tagRepo.GetBySlug(tagSlug)
}

func (s *PostService) GetAllTags() ([]models.Tag, error) {

	if s.cache != nil {
		var tags []models.Tag
		if err := s.cache.Get("tags:all", &tags); err == nil {
			return tags, nil
		}
	}

	tags, err := s.tagRepo.GetAll()
	if err != nil {
		return nil, err
	}

	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	if s.cache != nil {
		s.cache.Set("tags:all", tags, 2*time.Hour)
	}

	return tags, nil
}

func (s *PostService) GetTagsInUse() ([]models.Tag, error) {

	if s.cache != nil {
		var tags []models.Tag
		if err := s.cache.Get("tags:used", &tags); err == nil {
			return tags, nil
		}
	}

	tags, err := s.tagRepo.GetUsed()
	if err != nil {
		return nil, err
	}

	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i].Name) < strings.ToLower(tags[j].Name)
	})

	if s.cache != nil {
		s.cache.Set("tags:used", tags, 2*time.Hour)
	}

	return tags, nil
}

func (s *PostService) DeleteTag(id uint) error {
	if err := s.tagRepo.Delete(id); err != nil {
		return err
	}

	s.handleTagChanges()

	return nil
}

func (s *PostService) GetPopularPosts(limit int) ([]models.Post, error) {

	cacheKey := fmt.Sprintf("posts:popular:%d", limit)
	if s.cache != nil {
		var posts []models.Post
		if err := s.cache.Get(cacheKey, &posts); err == nil {
			return posts, nil
		}
	}

	posts, err := s.postRepo.GetPopular(limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, posts, 10*time.Minute)
	}

	return posts, nil
}

func (s *PostService) GetRecentPosts(limit int) ([]models.Post, error) {

	cacheKey := fmt.Sprintf("posts:recent:%d", limit)
	if s.cache != nil {
		var posts []models.Post
		if err := s.cache.Get(cacheKey, &posts); err == nil {
			return posts, nil
		}
	}

	posts, err := s.postRepo.GetRecent(limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, posts, 5*time.Minute)
	}

	return posts, nil
}

func (s *PostService) GetRelatedPosts(postID uint, limit int) ([]models.Post, error) {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("posts:related:%d:%d", postID, limit)
	if s.cache != nil {
		var posts []models.Post
		if err := s.cache.Get(cacheKey, &posts); err == nil {
			return posts, nil
		}
	}

	posts, err := s.postRepo.GetRelated(postID, post.CategoryID, limit)
	if err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.Set(cacheKey, posts, 30*time.Minute)
	}

	return posts, nil
}

func (s *PostService) PublishPost(postID uint) error {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	post.Published, post.PublishAt, post.PublishedAt = normalizePublicationState(true, &now, now)

	if err := s.postRepo.Update(post); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePost(postID)
		s.cache.InvalidatePostsCache()
	}

	return nil
}

func (s *PostService) UnpublishPost(postID uint) error {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	post.Published, post.PublishAt, post.PublishedAt = normalizePublicationState(false, nil, now)

	if err := s.postRepo.Update(post); err != nil {
		return err
	}

	if s.cache != nil {
		s.cache.InvalidatePost(postID)
		s.cache.InvalidatePostsCache()
	}

	return nil
}

func (s *PostService) IncrementViews(postID uint) error {
	return s.postRepo.IncrementViews(postID)
}

type PostAnalyticsPoint struct {
	Period   time.Time `json:"period"`
	Views    int64     `json:"views"`
	Comments int64     `json:"comments"`
}

type PostAnalyticsMetrics struct {
	TotalViews            int64   `json:"total_views"`
	TotalComments         int64   `json:"total_comments"`
	ViewsLast7Days        int64   `json:"views_last_7_days"`
	ViewsPrevious7Days    int64   `json:"views_previous_7_days"`
	CommentsLast7Days     int64   `json:"comments_last_7_days"`
	CommentsPrevious7Days int64   `json:"comments_previous_7_days"`
	ViewsChangePercent    float64 `json:"views_change_percent"`
	CommentsChangePercent float64 `json:"comments_change_percent"`
	EngagementRate        float64 `json:"engagement_rate"`
}

type PostAnalyticsComparisons struct {
	AverageViews                float64 `json:"average_views"`
	AverageComments             float64 `json:"average_comments"`
	ViewsVsAverageDifference    float64 `json:"views_vs_average_difference"`
	ViewsVsAveragePercent       float64 `json:"views_vs_average_percent"`
	CommentsVsAverageDifference float64 `json:"comments_vs_average_difference"`
	CommentsVsAveragePercent    float64 `json:"comments_vs_average_percent"`
	ViewsRankPosition           int64   `json:"views_rank_position"`
	ViewsRankTotal              int64   `json:"views_rank_total"`
	CommentsRankPosition        int64   `json:"comments_rank_position"`
	CommentsRankTotal           int64   `json:"comments_rank_total"`
}

type PostAnalytics struct {
	Metrics     PostAnalyticsMetrics     `json:"metrics"`
	Trend       []PostAnalyticsPoint     `json:"trend"`
	Comparisons PostAnalyticsComparisons `json:"comparisons"`
}

func (s *PostService) GetAnalytics(postID uint, days int) (*PostAnalytics, error) {
	if s.postRepo == nil {
		return nil, errors.New("post repository not configured")
	}

	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if !post.Published || (post.PublishAt != nil && post.PublishAt.After(now)) {
		return nil, ErrPostNotPublished
	}

	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	start := today.AddDate(0, 0, -(days - 1))

	viewStats, err := s.postRepo.GetViewStats(postID, start)
	if err != nil {
		return nil, err
	}

	var commentStats []repository.DailyCount
	if s.commentRepo != nil {
		commentStats, err = s.commentRepo.DailyCountsByPostID(postID, start)
		if err != nil {
			return nil, err
		}
	}

	viewMap := make(map[string]int64, len(viewStats))
	for _, entry := range viewStats {
		key := entry.Period.UTC().Format("2006-01-02")
		viewMap[key] = entry.Count
	}

	commentMap := make(map[string]int64, len(commentStats))
	for _, entry := range commentStats {
		key := entry.Period.UTC().Format("2006-01-02")
		commentMap[key] = entry.Count
	}

	trend := make([]PostAnalyticsPoint, 0, days)
	for index := 0; index < days; index++ {
		day := start.AddDate(0, 0, index)
		key := day.Format("2006-01-02")
		trend = append(trend, PostAnalyticsPoint{
			Period:   day,
			Views:    viewMap[key],
			Comments: commentMap[key],
		})
	}

	window := 7
	if days < window {
		window = days
	}

	var viewsLast, viewsPrev, commentsLast, commentsPrev int64
	if window > 0 {
		startLast := len(trend) - window
		if startLast < 0 {
			startLast = 0
		}
		for i := startLast; i < len(trend); i++ {
			viewsLast += trend[i].Views
			commentsLast += trend[i].Comments
		}

		startPrev := startLast - window
		if startPrev < 0 {
			startPrev = 0
		}
		for i := startPrev; i < startLast; i++ {
			viewsPrev += trend[i].Views
			commentsPrev += trend[i].Comments
		}
	}

	var totalComments int64
	if s.commentRepo != nil {
		totalComments, err = s.commentRepo.CountByPostID(postID)
		if err != nil {
			return nil, err
		}
	}

	totalViews := int64(post.Views)

	metrics := PostAnalyticsMetrics{
		TotalViews:            totalViews,
		TotalComments:         totalComments,
		ViewsLast7Days:        viewsLast,
		ViewsPrevious7Days:    viewsPrev,
		CommentsLast7Days:     commentsLast,
		CommentsPrevious7Days: commentsPrev,
		ViewsChangePercent:    calculatePercentChange(viewsLast, viewsPrev),
		CommentsChangePercent: calculatePercentChange(commentsLast, commentsPrev),
	}

	if totalViews > 0 {
		metrics.EngagementRate = (float64(totalComments) / float64(totalViews)) * 100
	}

	avgViews, err := s.postRepo.GetAverageViews()
	if err != nil {
		return nil, err
	}

	avgComments, err := s.postRepo.GetAverageComments()
	if err != nil {
		return nil, err
	}

	comparisons := PostAnalyticsComparisons{
		AverageViews:                avgViews,
		AverageComments:             avgComments,
		ViewsVsAverageDifference:    float64(totalViews) - avgViews,
		CommentsVsAverageDifference: float64(totalComments) - avgComments,
	}

	if avgViews > 0 {
		comparisons.ViewsVsAveragePercent = ((float64(totalViews) - avgViews) / avgViews) * 100
	}
	if avgComments > 0 {
		comparisons.CommentsVsAveragePercent = ((float64(totalComments) - avgComments) / avgComments) * 100
	}

	if rank, total, err := s.postRepo.GetViewRank(postID); err == nil {
		comparisons.ViewsRankPosition = rank
		comparisons.ViewsRankTotal = total
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if rank, total, err := s.postRepo.GetCommentRank(postID); err == nil {
		comparisons.CommentsRankPosition = rank
		comparisons.CommentsRankTotal = total
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	analytics := &PostAnalytics{
		Metrics:     metrics,
		Trend:       trend,
		Comparisons: comparisons,
	}

	return analytics, nil
}

func calculatePercentChange(current, previous int64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	return (float64(current-previous) / float64(previous)) * 100
}

func (s *PostService) fetchPostsByCategory(categorySlug string, categoryID uint, page, limit int) ([]models.Post, int64, error) {
	offset := (page - 1) * limit

	cacheKey := fmt.Sprintf("posts:category:%s:page:%d:limit:%d", categorySlug, page, limit)

	if s.cache != nil {
		var result struct {
			Posts []models.Post
			Total int64
		}
		if err := s.cache.Get(cacheKey, &result); err == nil {
			return result.Posts, result.Total, nil
		}
	}

	published := true

	posts, total, err := s.postRepo.GetAll(offset, limit, &categoryID, nil, nil, &published)
	if err != nil {
		return nil, 0, err
	}

	if s.cache != nil {
		result := struct {
			Posts []models.Post
			Total int64
		}{posts, total}
		s.cache.Set(cacheKey, result, 5*time.Minute)
	}

	return posts, total, nil
}

func (s *PostService) GetPostsByCategory(categorySlug string, page, limit int) ([]models.Post, int64, error) {
	_, posts, total, err := s.GetCategoryWithPosts(categorySlug, page, limit)
	return posts, total, err
}

func (s *PostService) GetCategoryWithPosts(categorySlug string, page, limit int) (*models.Category, []models.Post, int64, error) {
	if s.categoryRepo == nil {
		return nil, nil, 0, fmt.Errorf("category repository is not configured")
	}

	category, err := s.categoryRepo.GetBySlug(categorySlug)
	if err != nil {
		return nil, nil, 0, err
	}

	posts, total, err := s.fetchPostsByCategory(category.Slug, category.ID, page, limit)
	if err != nil {
		return nil, nil, 0, err
	}

	return category, posts, total, nil
}
