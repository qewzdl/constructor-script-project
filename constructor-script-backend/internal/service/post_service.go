package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/utils"
)

type PostService struct {
	postRepo repository.PostRepository
	tagRepo  repository.TagRepository
	cache    *cache.Cache
}

func NewPostService(postRepo repository.PostRepository, tagRepo repository.TagRepository, cacheService *cache.Cache) *PostService {
	return &PostService{
		postRepo: postRepo,
		tagRepo:  tagRepo,
		cache:    cacheService,
	}
}

func (s *PostService) Create(req models.CreatePostRequest, authorID uint) (*models.Post, error) {
	post := &models.Post{
		Title:       req.Title,
		Slug:        utils.GenerateSlug(req.Title),
		Content:     req.Content,
		Excerpt:     req.Excerpt,
		FeaturedImg: req.FeaturedImg,
		Published:   req.Published,
		AuthorID:    authorID,
		CategoryID:  req.CategoryID,
	}

	if len(req.TagNames) > 0 {
		tags, err := s.getOrCreateTags(req.TagNames)
		if err != nil {
			return nil, err
		}
		post.Tags = tags
	}

	if err := s.postRepo.Create(post); err != nil {
		return nil, err
	}

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

	if len(req.TagNames) > 0 {
		tags, err := s.getOrCreateTags(req.TagNames)
		if err != nil {
			return nil, err
		}
		post.Tags = tags
	}

	if err := s.postRepo.Update(post); err != nil {
		return nil, err
	}

	if s.cache != nil {
		s.cache.InvalidatePost(id)
		s.cache.InvalidatePostsCache()
	}

	return s.postRepo.GetByID(post.ID)
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

	posts, total, err := s.postRepo.GetAll(offset, limit, categoryID, tagName, authorID)
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

func (s *PostService) GetAllAdmin(page, limit int) ([]models.Post, int64, error) {
	offset := (page - 1) * limit
	return s.postRepo.GetAll(offset, limit, nil, nil, nil)
}

func (s *PostService) GetPostsByTag(tagSlug string, page, limit int) ([]models.Post, int64, error) {
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

	posts, total, err := s.postRepo.GetAll(offset, limit, nil, &tagSlug, nil)
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

	if s.cache != nil {
		s.cache.Set("tags:all", tags, 2*time.Hour)
	}

	return tags, nil
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

func (s *PostService) getOrCreateTags(tagNames []string) ([]models.Tag, error) {
	var tags []models.Tag

	for _, name := range tagNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		slug := utils.GenerateSlug(name)
		tag, err := s.tagRepo.GetBySlug(slug)

		if err != nil {

			tag = &models.Tag{
				Name: name,
				Slug: slug,
			}
			if err := s.tagRepo.Create(tag); err != nil {
				return nil, err
			}

			if s.cache != nil {
				s.cache.Delete("tags:all")
			}
		}

		tags = append(tags, *tag)
	}

	return tags, nil
}
