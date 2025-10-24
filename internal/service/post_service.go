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
	cache        *cache.Cache
	settingRepo  repository.SettingRepository
}

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
	cacheService *cache.Cache,
	settingRepo repository.SettingRepository,
) *PostService {
	return &PostService{
		postRepo:     postRepo,
		tagRepo:      tagRepo,
		categoryRepo: categoryRepo,
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

func (s *PostService) Update(id uint, req models.UpdatePostRequest, userID uint, isAdmin bool) (*models.Post, error) {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !isAdmin && post.AuthorID != userID {
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
		case "standard":
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

		if section.Title == "" {
			return nil, fmt.Errorf("section %d: title is required", i)
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
		content.WriteString(section.Title)
		content.WriteString("\n\n")

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

func (s *PostService) Delete(id uint, userID uint, isAdmin bool) error {
	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return err
	}

	if !isAdmin && post.AuthorID != userID {
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

			s.cache.IncrementViews(id)
			return &post, nil
		}
	}

	post, err := s.postRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	go func() {
		post.Views++
		s.postRepo.Update(post)
	}()

	if s.cache != nil {
		s.cache.CachePost(id, post)
	}

	return post, nil
}

func (s *PostService) GetBySlug(slug string) (*models.Post, error) {

	if s.cache != nil {
		var post models.Post
		cacheKey := fmt.Sprintf("post:slug:%s", slug)
		if err := s.cache.Get(cacheKey, &post); err == nil {
			return &post, nil
		}
	}

	post, err := s.postRepo.GetBySlug(slug)
	if err != nil {
		return nil, err
	}

	go func() {
		post.Views++
		s.postRepo.Update(post)
	}()

	if s.cache != nil {
		cacheKey := fmt.Sprintf("post:slug:%s", slug)
		s.cache.Set(cacheKey, post, 1*time.Hour)
		s.cache.CachePost(post.ID, post)
	}

	return post, nil
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

	posts, total, err := s.postRepo.GetAll(offset, limit, nil, &tagSlug, nil, &published)
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

func (s *PostService) GetPostsByTag(tagSlug string, page, limit int) ([]models.Post, int64, error) {
	_, posts, total, err := s.GetTagWithPosts(tagSlug, page, limit)
	return posts, total, err
}

func (s *PostService) GetTagWithPosts(tagSlug string, page, limit int) (*models.Tag, []models.Post, int64, error) {
	tag, err := s.tagRepo.GetBySlug(tagSlug)
	if err != nil {
		return nil, nil, 0, err
	}

	posts, total, err := s.fetchPostsByTag(tag.Slug, page, limit)
	if err != nil {
		return nil, nil, 0, err
	}

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

	post.Published = true

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

	post.Published = false

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
