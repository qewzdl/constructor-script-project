package postseed

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	postservice "constructor-script-backend/plugins/posts/service"
)

type postDefinition struct {
	models.CreatePostRequest `json:",inline"`
	AuthorEmail              string `json:"author_email"`
	AuthorUsername           string `json:"author_username"`
}

var errNoDefaultPostAuthor = errors.New("no author available for default posts")

func EnsureDefaultPosts(postService *postservice.PostService, userRepo repository.UserRepository, dataFS fs.FS) {
	if postService == nil || userRepo == nil || dataFS == nil {
		return
	}

	entries, err := fs.ReadDir(dataFS, ".")
	if err != nil {
		logger.Error(err, "Failed to read post definitions", nil)
		return
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}

		data, readErr := fs.ReadFile(dataFS, name)
		if readErr != nil {
			logger.Error(readErr, "Failed to read post definition", map[string]interface{}{"file": name})
			continue
		}

		definitions, parseErr := parsePostDefinitions(data)
		if parseErr != nil {
			logger.Error(parseErr, "Failed to parse post definition", map[string]interface{}{"file": name})
			continue
		}

		for _, definition := range definitions {
			ensureDefaultPost(postService, userRepo, definition, name)
		}
	}
}

func parsePostDefinitions(data []byte) ([]postDefinition, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, nil
	}

	if trimmed[0] == '[' {
		var definitions []postDefinition
		if err := json.Unmarshal(trimmed, &definitions); err != nil {
			return nil, err
		}
		return definitions, nil
	}

	var definition postDefinition
	if err := json.Unmarshal(trimmed, &definition); err != nil {
		return nil, err
	}

	return []postDefinition{definition}, nil
}

func ensureDefaultPost(postService *postservice.PostService, userRepo repository.UserRepository, definition postDefinition, source string) {
	title := strings.TrimSpace(definition.Title)
	if title == "" {
		return
	}

	slug := utils.GenerateSlug(title)
	if slug == "" {
		return
	}

	exists, err := postService.ExistsBySlug(slug)
	if err != nil {
		logger.Error(err, "Failed to verify default post", map[string]interface{}{"slug": slug, "source": source})
		return
	}
	if exists {
		logger.Info("Default post already present", map[string]interface{}{"slug": slug, "source": source})
		return
	}

	authorID, authorErr := resolvePostAuthor(userRepo, definition)
	if authorErr != nil {
		if errors.Is(authorErr, errNoDefaultPostAuthor) || errors.Is(authorErr, gorm.ErrRecordNotFound) {
			logger.Warn("Skipping default post; no author available", map[string]interface{}{"slug": slug, "source": source})
		} else {
			logger.Error(authorErr, "Failed to resolve default post author", map[string]interface{}{"slug": slug, "source": source})
		}
		return
	}

	req := definition.CreatePostRequest
	if req.Template == "" {
		req.Template = "post"
	}
	if !req.Published {
		req.Published = true
	}
	req.TagNames = normalizeTagNames(req.TagNames)

	if _, err := postService.Create(req, authorID); err != nil {
		logger.Error(err, "Failed to create default post", map[string]interface{}{"slug": slug, "source": source})
		return
	}

	logger.Info("Ensured default post", map[string]interface{}{"slug": slug, "source": source})
}

func normalizeTagNames(tags []string) []string {
	if len(tags) == 0 {
		return tags
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))

	for _, tag := range tags {
		cleaned := strings.TrimSpace(tag)
		if cleaned == "" {
			continue
		}

		slug := utils.GenerateSlug(cleaned)
		if slug == "" {
			continue
		}

		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		normalized = append(normalized, cleaned)
	}

	return normalized
}

func resolvePostAuthor(userRepo repository.UserRepository, definition postDefinition) (uint, error) {
	if userRepo == nil {
		return 0, errors.New("user repository not configured")
	}

	if email := strings.TrimSpace(definition.AuthorEmail); email != "" {
		user, err := userRepo.GetByEmail(email)
		if err == nil {
			return user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("lookup author by email: %w", err)
		}
	}

	if username := strings.TrimSpace(definition.AuthorUsername); username != "" {
		user, err := userRepo.GetByUsername(username)
		if err == nil {
			return user.ID, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("lookup author by username: %w", err)
		}
	}

	users, err := userRepo.GetAll()
	if err != nil {
		return 0, fmt.Errorf("list users: %w", err)
	}
	if len(users) == 0 {
		return 0, errNoDefaultPostAuthor
	}

	return users[0].ID, nil
}
