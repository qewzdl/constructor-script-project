package service

import (
	"encoding/json"
	"testing"

	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"

	"gorm.io/gorm"
)

type memorySettingRepository struct {
	store map[string]string
}

func newMemorySettingRepository() *memorySettingRepository {
	return &memorySettingRepository{store: make(map[string]string)}
}

func (m *memorySettingRepository) Get(key string) (*models.Setting, error) {
	if value, ok := m.store[key]; ok {
		return &models.Setting{Key: key, Value: value}, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *memorySettingRepository) Set(key, value string) error {
	m.store[key] = value
	return nil
}

func (m *memorySettingRepository) Delete(key string) error {
	delete(m.store, key)
	return nil
}

var _ repository.SettingRepository = (*memorySettingRepository)(nil)

func TestFontService_CreateAndList(t *testing.T) {
	repo := newMemorySettingRepository()
	service := NewFontService(repo)
	if service == nil {
		t.Fatal("expected service to be created")
	}

	font, err := service.Create(models.CreateFontAssetRequest{
		Name:        "  Example Font  ",
		Snippet:     "  <link href=\"https://example.com/font.css\" rel=\"stylesheet\">  ",
		Preconnects: []string{"https://fonts.example.com", " https://fonts.example.com ", "https://cdn.example.com"},
		Notes:       "  Primary heading typeface  ",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if font.Name != "Example Font" {
		t.Fatalf("expected trimmed name, got %q", font.Name)
	}
	if !font.Enabled {
		t.Fatalf("new fonts should default to enabled")
	}
	if len(font.Preconnects) != 2 {
		t.Fatalf("expected duplicates removed, got %v", font.Preconnects)
	}

	fonts, err := service.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(fonts) == 0 {
		t.Fatalf("expected at least one font")
	}

	var found bool
	for _, entry := range fonts {
		if entry.ID == font.ID {
			found = true
			if entry.Snippet != "<link href=\"https://example.com/font.css\" rel=\"stylesheet\">" {
				t.Fatalf("expected snippet trimmed, got %q", entry.Snippet)
			}
			if entry.Notes != "Primary heading typeface" {
				t.Fatalf("expected notes trimmed, got %q", entry.Notes)
			}
			break
		}
	}
	if !found {
		t.Fatalf("created font not returned from List")
	}

	// Ensure the configuration was persisted to the repository.
	raw, ok := repo.store[settingKeySiteFonts]
	if !ok {
		t.Fatalf("expected fonts saved to repository")
	}
	var stored []models.FontAsset
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("failed to unmarshal stored fonts: %v", err)
	}
	if len(stored) < len(fonts) {
		t.Fatalf("expected stored fonts to match list")
	}
}

func TestFontService_UpdateAndReorder(t *testing.T) {
	repo := newMemorySettingRepository()
	service := NewFontService(repo)
	if service == nil {
		t.Fatal("expected service to be created")
	}

	first, err := service.Create(models.CreateFontAssetRequest{
		Name:    "First",
		Snippet: "<style>body{}</style>",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	second, err := service.Create(models.CreateFontAssetRequest{
		Name:        "Second",
		Snippet:     "<link href=\"https://cdn.second/font.css\" rel=\"stylesheet\">",
		Enabled:     boolPtr(false),
		Preconnects: []string{"https://cdn.second"},
	})
	if err != nil {
		t.Fatalf("second Create returned error: %v", err)
	}

	updatedName := "Updated Second"
	newNotes := "Use for body copy"
	newPreconnects := []string{"https://cdn.second", "https://static.second"}
	updated, err := service.Update(second.ID, models.UpdateFontAssetRequest{
		Name:        &updatedName,
		Enabled:     boolPtr(true),
		Notes:       &newNotes,
		Preconnects: &newPreconnects,
	})
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if !updated.Enabled {
		t.Fatalf("expected update to enable font")
	}
	if len(updated.Preconnects) != 2 {
		t.Fatalf("expected two preconnects, got %v", updated.Preconnects)
	}

	fonts, err := service.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	orders := []models.FontAssetOrder{
		{ID: updated.ID, Order: 1},
		{ID: first.ID, Order: 2},
	}
	for _, f := range fonts {
		if f.ID != updated.ID && f.ID != first.ID {
			orders = append(orders, models.FontAssetOrder{ID: f.ID, Order: len(orders) + 1})
		}
	}

	if err := service.Reorder(orders); err != nil {
		t.Fatalf("Reorder returned error: %v", err)
	}

	reordered, err := service.List()
	if err != nil {
		t.Fatalf("List after reorder returned error: %v", err)
	}
	if len(reordered) == 0 || reordered[0].ID != updated.ID {
		t.Fatalf("expected updated font to be first after reorder")
	}
}

func TestFontService_Delete(t *testing.T) {
	repo := newMemorySettingRepository()
	service := NewFontService(repo)
	if service == nil {
		t.Fatal("expected service to be created")
	}

	font, err := service.Create(models.CreateFontAssetRequest{
		Name:    "Deletable",
		Snippet: "<style>@font-face{}</style>",
	})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	if err := service.Delete(font.ID); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	fonts, err := service.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	for _, entry := range fonts {
		if entry.ID == font.ID {
			t.Fatalf("expected font %s to be removed", font.ID)
		}
	}
}

func TestCollectFontPreconnects(t *testing.T) {
	fonts := []models.FontAsset{
		{
			Preconnects: []string{"https://fonts.example.com", "https://cdn.example.com"},
		},
		{
			Preconnects: []string{"https://fonts.example.com", " https://fonts.gstatic.com "},
		},
	}
	preconnects := CollectFontPreconnects(fonts)
	if len(preconnects) != 3 {
		t.Fatalf("expected three unique preconnects, got %v", preconnects)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
