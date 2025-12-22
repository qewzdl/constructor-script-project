package models

// AddSectionRequest represents a request to add a new section to a page.
type AddSectionRequest struct {
	Type            string `json:"type" binding:"required"`
	Title           string `json:"title"`
	Description     string `json:"description"`
	PaddingVertical *int   `json:"padding_vertical,omitempty"`
	MarginVertical  *int   `json:"margin_vertical,omitempty"`
	Disabled        *bool  `json:"disabled,omitempty"`
}

// UpdateSectionRequest represents a request to update an existing section.
type UpdateSectionRequest struct {
	Title           *string           `json:"title,omitempty"`
	Description     *string           `json:"description,omitempty"`
	Type            *string           `json:"type,omitempty"`
	Elements        *[]SectionElement `json:"elements,omitempty"`
	PaddingVertical *int              `json:"padding_vertical,omitempty"`
	MarginVertical  *int              `json:"margin_vertical,omitempty"`
	Limit           *int              `json:"limit,omitempty"`
	Mode            *string           `json:"mode,omitempty"`
	StyleGridItems  *bool             `json:"style_grid_items,omitempty"`
	Disabled        *bool             `json:"disabled,omitempty"`
}

// PageTemplate represents a predefined page layout template.
type PageTemplate struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Thumbnail   string    `json:"thumbnail,omitempty"`
	Sections    []Section `json:"sections"`
}

// PageBuilderConfig contains configuration for the page builder UI.
type PageBuilderConfig struct {
	AvailableSections []SectionTypeConfig `json:"available_sections"`
	DefaultPadding    int                 `json:"default_padding"`
	DefaultMargin     int                 `json:"default_margin"`
	PaddingOptions    []int               `json:"padding_options"`
	MarginOptions     []int               `json:"margin_options"`
}

// SectionTypeConfig describes a section type available in the builder.
type SectionTypeConfig struct {
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Icon        string                 `json:"icon"`
	Preview     string                 `json:"preview,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	AllowedIn   []string               `json:"allowed_in,omitempty"` // e.g., ["page", "post", "homepage"]
}
