package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
)

const (
	settingKeySiteFonts    = "site.fonts"
	defaultFontOutfitID    = "default-outfit"
	defaultFontOutfitNotes = "Default font applied to new sites."
)

var (
	// ErrFontNotFound is returned when a font asset could not be located.
	ErrFontNotFound = errors.New("font asset not found")
	// ErrInvalidFontSnippet indicates the provided embed code was empty.
	ErrInvalidFontSnippet = errors.New("font snippet is required")
)

// FontService manages the set of external font resources used by the site.
type FontService struct {
	repo repository.SettingRepository
}

// NewFontService creates a new font service backed by the provided settings repository.
func NewFontService(repo repository.SettingRepository) *FontService {
	if repo == nil {
		return nil
	}
	return &FontService{repo: repo}
}

// DefaultFontAssets returns a copy of the bundled default font configuration.
func DefaultFontAssets() []models.FontAsset {
	return []models.FontAsset{
		{
			ID:          defaultFontOutfitID,
			Name:        "Outfit (Google Fonts)",
			Snippet:     "<link href=\"https://fonts.googleapis.com/css2?family=Outfit:wght@100..900&display=swap\" rel=\"stylesheet\">",
			Preconnects: []string{"https://fonts.googleapis.com", "https://fonts.gstatic.com"},
			Order:       1,
			Enabled:     true,
			Notes:       defaultFontOutfitNotes,
		},
	}
}

// List returns all configured font assets ordered by their configured order.
func (s *FontService) List() ([]models.FontAsset, error) {
	fonts, err := s.load()
	if err != nil {
		return nil, err
	}
	return cloneFonts(fonts), nil
}

// ListActive returns only the enabled font assets.
func (s *FontService) ListActive() ([]models.FontAsset, error) {
	fonts, err := s.List()
	if err != nil {
		return nil, err
	}
	active := make([]models.FontAsset, 0, len(fonts))
	for _, font := range fonts {
		if font.Enabled {
			active = append(active, font)
		}
	}
	return active, nil
}

// Create stores a new font asset and returns the persisted version.
func (s *FontService) Create(req models.CreateFontAssetRequest) (*models.FontAsset, error) {
	fonts, err := s.load()
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	snippet := strings.TrimSpace(req.Snippet)
	if snippet == "" {
		return nil, ErrInvalidFontSnippet
	}
	if name == "" {
		name = "Custom font"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	preconnects := normalizePreconnects(req.Preconnects)
	notes := strings.TrimSpace(req.Notes)

	order := nextFontOrder(fonts)
	font := models.FontAsset{
		ID:          uuid.NewString(),
		Name:        name,
		Snippet:     snippet,
		Preconnects: preconnects,
		Order:       order,
		Enabled:     enabled,
		Notes:       notes,
	}

	fonts = append(fonts, font)
	if err := s.save(fonts); err != nil {
		return nil, err
	}

	return &font, nil
}

// Update modifies an existing font asset by ID.
func (s *FontService) Update(id string, req models.UpdateFontAssetRequest) (*models.FontAsset, error) {
	fonts, err := s.load()
	if err != nil {
		return nil, err
	}

	index := indexOfFont(fonts, id)
	if index == -1 {
		return nil, ErrFontNotFound
	}

	font := fonts[index]

	if req.Name != nil {
		if trimmed := strings.TrimSpace(*req.Name); trimmed != "" {
			font.Name = trimmed
		}
	}
	if req.Snippet != nil {
		if trimmed := strings.TrimSpace(*req.Snippet); trimmed != "" {
			font.Snippet = trimmed
		}
	}
	if req.Preconnects != nil {
		font.Preconnects = normalizePreconnects(*req.Preconnects)
	}
	if req.Enabled != nil {
		font.Enabled = *req.Enabled
	}
	if req.Notes != nil {
		font.Notes = strings.TrimSpace(*req.Notes)
	}

	fonts[index] = font
	if err := s.save(fonts); err != nil {
		return nil, err
	}

	return &font, nil
}

// Delete removes the specified font asset.
func (s *FontService) Delete(id string) error {
	fonts, err := s.load()
	if err != nil {
		return err
	}

	index := indexOfFont(fonts, id)
	if index == -1 {
		return ErrFontNotFound
	}

	fonts = append(fonts[:index], fonts[index+1:]...)
	if err := s.save(fonts); err != nil {
		return err
	}

	return nil
}

// Reorder updates the display order for the provided font assets.
func (s *FontService) Reorder(orders []models.FontAssetOrder) error {
	if len(orders) == 0 {
		return errors.New("no font order provided")
	}

	fonts, err := s.load()
	if err != nil {
		return err
	}

	orderMap := make(map[string]int, len(orders))
	for _, entry := range orders {
		if entry.ID == "" {
			continue
		}
		orderMap[entry.ID] = entry.Order
	}

	for idx, font := range fonts {
		if order, ok := orderMap[font.ID]; ok {
			fonts[idx].Order = order
		}
	}

	if err := s.save(fonts); err != nil {
		return err
	}

	return nil
}

func (s *FontService) load() ([]models.FontAsset, error) {
	defaults := DefaultFontAssets()
	if s == nil || s.repo == nil {
		return defaults, nil
	}

	setting, err := s.repo.Get(settingKeySiteFonts)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return cloneFonts(defaults), nil
		}
		return nil, err
	}

	value := strings.TrimSpace(setting.Value)
	if value == "" {
		return []models.FontAsset{}, nil
	}

	var fonts []models.FontAsset
	if err := json.Unmarshal([]byte(value), &fonts); err != nil {
		return nil, err
	}

	normalized := make([]models.FontAsset, 0, len(fonts))
	seen := make(map[string]struct{}, len(fonts))
	for _, font := range fonts {
		normaliseFont(&font)
		if font.ID == "" {
			font.ID = uuid.NewString()
		}
		if _, exists := seen[font.ID]; exists {
			font.ID = uuid.NewString()
		}
		seen[font.ID] = struct{}{}
		normalized = append(normalized, font)
	}

	sortFonts(normalized)
	return normalized, nil
}

func (s *FontService) save(fonts []models.FontAsset) error {
	if s == nil || s.repo == nil {
		return errors.New("font repository not configured")
	}

	cleaned := make([]models.FontAsset, 0, len(fonts))
	seen := make(map[string]struct{}, len(fonts))
	for _, font := range fonts {
		normaliseFont(&font)
		if font.ID == "" {
			font.ID = uuid.NewString()
		}
		if _, exists := seen[font.ID]; exists {
			font.ID = uuid.NewString()
		}
		seen[font.ID] = struct{}{}
		cleaned = append(cleaned, font)
	}

	sortFonts(cleaned)
	payload, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}

	return s.repo.Set(settingKeySiteFonts, string(payload))
}

func cloneFonts(fonts []models.FontAsset) []models.FontAsset {
	if len(fonts) == 0 {
		return []models.FontAsset{}
	}
	result := make([]models.FontAsset, len(fonts))
	copy(result, fonts)
	return result
}

func normaliseFont(font *models.FontAsset) {
	if font == nil {
		return
	}
	font.Name = strings.TrimSpace(font.Name)
	font.Snippet = strings.TrimSpace(font.Snippet)
	font.Notes = strings.TrimSpace(font.Notes)
	font.Preconnects = normalizePreconnects(font.Preconnects)
	if font.Name == "" {
		font.Name = "Custom font"
	}
}

func normalizePreconnects(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

// CollectFontPreconnects deduplicates and normalizes preconnect URLs across all fonts.
func CollectFontPreconnects(fonts []models.FontAsset) []string {
	if len(fonts) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, font := range fonts {
		for _, entry := range font.Preconnects {
			trimmed := strings.TrimSpace(entry)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, trimmed)
		}
	}
	return result
}

func sortFonts(fonts []models.FontAsset) {
	sort.SliceStable(fonts, func(i, j int) bool {
		if fonts[i].Order == fonts[j].Order {
			return strings.ToLower(fonts[i].Name) < strings.ToLower(fonts[j].Name)
		}
		return fonts[i].Order < fonts[j].Order
	})
	for idx := range fonts {
		fonts[idx].Order = idx + 1
	}
}

func nextFontOrder(fonts []models.FontAsset) int {
	maxOrder := 0
	for _, font := range fonts {
		if font.Order > maxOrder {
			maxOrder = font.Order
		}
	}
	return maxOrder + 1
}

func indexOfFont(fonts []models.FontAsset, id string) int {
	for idx, font := range fonts {
		if font.ID == id {
			return idx
		}
	}
	return -1
}
