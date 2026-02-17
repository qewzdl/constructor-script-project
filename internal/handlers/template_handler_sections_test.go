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
