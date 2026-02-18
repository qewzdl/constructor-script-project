package handlers

import "testing"

func TestResolveSectionTextPosition(t *testing.T) {
	cases := []struct {
		name       string
		settings   map[string]interface{}
		settingKey string
		want       string
	}{
		{
			name:       "default when settings are missing",
			settingKey: "title_position",
			want:       "left",
		},
		{
			name: "default when setting key is missing",
			settings: map[string]interface{}{
				"title_position": "right",
			},
			settingKey: "description_position",
			want:       "left",
		},
		{
			name: "center normalised",
			settings: map[string]interface{}{
				"description_position": "CENTER",
			},
			settingKey: "description_position",
			want:       "center",
		},
		{
			name: "right from alternative token",
			settings: map[string]interface{}{
				"description_position": "end",
			},
			settingKey: "description_position",
			want:       "right",
		},
		{
			name: "invalid falls back to left",
			settings: map[string]interface{}{
				"description_position": "diagonal",
			},
			settingKey: "description_position",
			want:       "left",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSectionTextPosition(tc.settings, tc.settingKey)
			if got != tc.want {
				t.Fatalf("resolveSectionTextPosition() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildSectionTextClass(t *testing.T) {
	cases := []struct {
		name      string
		baseClass string
		position  string
		want      string
	}{
		{
			name:      "title left",
			baseClass: "page-view__section-title",
			position:  "left",
			want:      "page-view__section-title page-view__section-title--align-left",
		},
		{
			name:      "description center",
			baseClass: "page-view__section-description",
			position:  "center",
			want:      "page-view__section-description page-view__section-description--align-center",
		},
		{
			name:      "description right",
			baseClass: "page-view__section-description",
			position:  "right",
			want:      "page-view__section-description page-view__section-description--align-right",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSectionTextClass(tc.baseClass, tc.position)
			if got != tc.want {
				t.Fatalf("buildSectionTextClass() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSectionBackgroundGroup(t *testing.T) {
	cases := []struct {
		name     string
		settings map[string]interface{}
		want     string
	}{
		{
			name: "missing settings",
			want: "",
		},
		{
			name: "snake case key",
			settings: map[string]interface{}{
				"background_group": "Hero Surface",
			},
			want: "hero-surface",
		},
		{
			name: "camel case key",
			settings: map[string]interface{}{
				"backgroundGroup": "Landing_One",
			},
			want: "landing-one",
		},
		{
			name: "invalid token",
			settings: map[string]interface{}{
				"background_group": "###",
			},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSectionBackgroundGroup(tc.settings)
			if got != tc.want {
				t.Fatalf("resolveSectionBackgroundGroup() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestResolveSectionBackgroundStyle(t *testing.T) {
	cases := []struct {
		name     string
		settings map[string]interface{}
		want     string
	}{
		{
			name: "missing settings",
			want: "",
		},
		{
			name: "snake case key",
			settings: map[string]interface{}{
				"background_style": "Primary Gradient",
			},
			want: "primary-gradient",
		},
		{
			name: "camel case key",
			settings: map[string]interface{}{
				"backgroundStyle": "Accent_Soft",
			},
			want: "accent-soft",
		},
		{
			name: "invalid token",
			settings: map[string]interface{}{
				"background_style": "###",
			},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSectionBackgroundStyle(tc.settings)
			if got != tc.want {
				t.Fatalf("resolveSectionBackgroundStyle() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildSectionSideSpacingClasses(t *testing.T) {
	value := 33
	gotPaddingTop := buildSectionPaddingTopClass("page-view", &value)
	if gotPaddingTop != "page-view__section--pt-32" {
		t.Fatalf("buildSectionPaddingTopClass() = %q, want %q", gotPaddingTop, "page-view__section--pt-32")
	}

	gotPaddingBottom := buildSectionPaddingBottomClass("page-view", &value)
	if gotPaddingBottom != "page-view__section--pb-32" {
		t.Fatalf("buildSectionPaddingBottomClass() = %q, want %q", gotPaddingBottom, "page-view__section--pb-32")
	}

	gotMarginTop := buildSectionMarginTopClass("page-view", &value)
	if gotMarginTop != "page-view__section--mt-32" {
		t.Fatalf("buildSectionMarginTopClass() = %q, want %q", gotMarginTop, "page-view__section--mt-32")
	}

	gotMarginBottom := buildSectionMarginBottomClass("page-view", &value)
	if gotMarginBottom != "page-view__section--mb-32" {
		t.Fatalf("buildSectionMarginBottomClass() = %q, want %q", gotMarginBottom, "page-view__section--mb-32")
	}

	if got := buildSectionPaddingTopClass("page-view", nil); got != "" {
		t.Fatalf("buildSectionPaddingTopClass() with nil value = %q, want empty", got)
	}
	if got := buildSectionMarginBottomClass("page-view", nil); got != "" {
		t.Fatalf("buildSectionMarginBottomClass() with nil value = %q, want empty", got)
	}
}
