package service

import (
	"strings"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/validator"
)

type SearchService struct {
	searchRepo repository.SearchRepository
}

type SearchResult struct {
	Posts []models.Post `json:"posts"`
	Total int           `json:"total"`
	Query string        `json:"query"`
}

func NewSearchService(searchRepo repository.SearchRepository) *SearchService {
	return &SearchService{searchRepo: searchRepo}
}

func (s *SearchService) Search(query string, searchType string, limit int) (*SearchResult, error) {

	query = validator.TrimSpaces(query)
	query = validator.NormalizeSpaces(query)

	if query == "" {
		return &SearchResult{
			Posts: []models.Post{},
			Total: 0,
			Query: query,
		}, nil
	}

	var posts []models.Post
	var err error

	switch searchType {
	case "title":
		posts, err = s.searchRepo.SearchByTitle(query, limit)
	case "content":
		posts, err = s.searchRepo.SearchByContent(query, limit)
	case "tag":
		posts, err = s.searchRepo.SearchByTag(query, limit)
	case "author":
		posts, err = s.searchRepo.SearchByAuthor(query, limit)
	default:
		posts, err = s.searchRepo.SearchPosts(query, limit)
	}

	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Posts: posts,
		Total: len(posts),
		Query: query,
	}, nil
}

func (s *SearchService) SearchMultiple(query string, limit int) (*SearchResult, error) {
	query = validator.TrimSpaces(query)

	terms := strings.Fields(query)
	if len(terms) == 0 {
		return &SearchResult{Posts: []models.Post{}, Total: 0, Query: query}, nil
	}

	posts, err := s.searchRepo.SearchPosts(query, limit)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Posts: posts,
		Total: len(posts),
		Query: query,
	}, nil
}

func (s *SearchService) SuggestTags(query string, limit int) ([]string, error) {
	posts, err := s.searchRepo.SearchByTag(query, limit)
	if err != nil {
		return nil, err
	}

	tagSet := make(map[string]bool)
	var tags []string

	for _, post := range posts {
		for _, tag := range post.Tags {
			if !tagSet[tag.Name] {
				tagSet[tag.Name] = true
				tags = append(tags, tag.Name)
			}
		}
	}

	return tags, nil
}
