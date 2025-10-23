package service

import (
	"fmt"
	"html"
	"sort"
	"strings"
	"unicode"

	"constructor-script-backend/internal/models"
)

const googleAdsProviderKey = "google_ads"

var (
	googleAdsFormats = map[string]models.AdvertisingFormat{
		"auto": {
			Key:         "auto",
			Label:       "Responsive",
			Description: "Let Google optimise the format for the available space.",
		},
		"horizontal": {
			Key:   "horizontal",
			Label: "Horizontal banner",
		},
		"vertical": {
			Key:   "vertical",
			Label: "Vertical skyscraper",
		},
		"rectangle": {
			Key:   "rectangle",
			Label: "Rectangle",
		},
	}

	googleAdsPlacements = []models.AdvertisingPlacement{
		{Key: "layout_top", Label: "Top of page", Description: "Displayed immediately after the opening <body> tag.", Recommended: true},
		{Key: "main_top", Label: "Above main content", Description: "Shown before the primary page content."},
		{Key: "post_content_top", Label: "Post content (top)", Description: "Appears before the article body on post pages."},
		{Key: "post_content_bottom", Label: "Post content (bottom)", Description: "Appears after the article body on post pages."},
		{Key: "post_sidebar", Label: "Post sidebar", Description: "Displayed inside the article sidebar."},
		{Key: "footer", Label: "Footer", Description: "Shown above the footer section."},
		{Key: "layout_bottom", Label: "Bottom of page", Description: "Rendered before closing scripts and </body>."},
	}

	allowedGooglePlacements = func() map[string]struct{} {
		allowed := make(map[string]struct{}, len(googleAdsPlacements))
		for _, placement := range googleAdsPlacements {
			allowed[placement.Key] = struct{}{}
		}
		return allowed
	}()
)

func newGoogleAdsProvider() AdvertisingProvider {
	return &googleAdsProvider{}
}

type googleAdsProvider struct{}

func (p *googleAdsProvider) Key() string {
	return googleAdsProviderKey
}

func (p *googleAdsProvider) Metadata() models.AdvertisingProviderMetadata {
	formats := make([]models.AdvertisingFormat, 0, len(googleAdsFormats))
	for _, format := range googleAdsFormats {
		formats = append(formats, format)
	}
	sort.Slice(formats, func(i, j int) bool {
		return formats[i].Key < formats[j].Key
	})

	placements := make([]models.AdvertisingPlacement, len(googleAdsPlacements))
	copy(placements, googleAdsPlacements)

	return models.AdvertisingProviderMetadata{
		Key:             googleAdsProviderKey,
		Name:            "Google AdSense",
		Description:     "Serve contextual display ads using Google AdSense units.",
		SupportsAutoAds: true,
		Placements:      placements,
		Formats:         formats,
	}
}

func (p *googleAdsProvider) Normalize(settings models.AdvertisingSettings) (models.AdvertisingSettings, error) {
	cfg := settings.GoogleAds
	if cfg == nil {
		return models.AdvertisingSettings{}, validationErrorf("google ads configuration is required")
	}

	normalized := *cfg
	normalized.PublisherID = strings.TrimSpace(normalized.PublisherID)
	if normalized.PublisherID == "" {
		return models.AdvertisingSettings{}, validationErrorf("google ads publisher id is required")
	}
	if !strings.HasPrefix(normalized.PublisherID, "ca-pub-") {
		return models.AdvertisingSettings{}, validationErrorf("google ads publisher id must start with 'ca-pub-'")
	}
	if len(normalized.PublisherID) != len("ca-pub-")+16 {
		return models.AdvertisingSettings{}, validationErrorf("google ads publisher id must include 16 digits")
	}
	for _, r := range normalized.PublisherID[len("ca-pub-"):] {
		if r < '0' || r > '9' {
			return models.AdvertisingSettings{}, validationErrorf("google ads publisher id must include only digits after 'ca-pub-'")
		}
	}

	slots := make([]models.GoogleAdsSlot, 0, len(normalized.Slots))
	for _, slot := range normalized.Slots {
		placement := sanitizePlacement(slot.Placement)
		if placement == "" {
			continue
		}
		if _, ok := allowedGooglePlacements[placement]; !ok {
			continue
		}
		slotID := strings.TrimSpace(slot.SlotID)
		if slotID == "" {
			continue
		}
		format := normalizeGoogleFormat(slot.Format)
		slots = append(slots, models.GoogleAdsSlot{
			Placement:           placement,
			SlotID:              slotID,
			Format:              format,
			FullWidthResponsive: slot.FullWidthResponsive,
		})
	}

	normalized.Slots = slots
	settings.Provider = googleAdsProviderKey
	settings.GoogleAds = &normalized

	if !settings.Enabled {
		return settings, nil
	}

	if !normalized.AutoAds && len(normalized.Slots) == 0 {
		return models.AdvertisingSettings{}, validationErrorf("at least one ad slot is required when auto ads are disabled")
	}

	return settings, nil
}

func (p *googleAdsProvider) SecurityDirectives(settings models.AdvertisingSettings) models.ContentSecurityPolicyDirectives {
	directives := make(models.ContentSecurityPolicyDirectives)

	if !settings.Enabled {
		return directives
	}

	cfg := settings.GoogleAds
	if cfg == nil {
		return directives
	}

	directives["script-src"] = []string{
		"https://pagead2.googlesyndication.com",
		"https://securepubads.g.doubleclick.net",
		"https://www.googletagservices.com",
	}

	directives["frame-src"] = []string{
		"https://googleads.g.doubleclick.net",
		"https://tpc.googlesyndication.com",
		"https://adservice.google.com",
	}

	directives["child-src"] = []string{
		"https://googleads.g.doubleclick.net",
		"https://tpc.googlesyndication.com",
	}

	directives["connect-src"] = []string{
		"https://googleads.g.doubleclick.net",
		"https://pagead2.googlesyndication.com",
		"https://*.adtrafficquality.google",
	}

	return directives
}

func (p *googleAdsProvider) Render(settings models.AdvertisingSettings) (RenderedAdvertising, error) {
	cfg := settings.GoogleAds
	if cfg == nil {
		return RenderedAdvertising{Enabled: false}, nil
	}

	publisherID := html.EscapeString(cfg.PublisherID)

	placements := make(map[string][]string)
	for _, slot := range cfg.Slots {
		if slot.SlotID == "" || slot.Placement == "" {
			continue
		}

		format := normalizeGoogleFormat(slot.Format)
		escapedSlotID := html.EscapeString(slot.SlotID)
		cssClass := html.EscapeString(cssClassFromPlacement(slot.Placement))

		attrs := []string{fmt.Sprintf(`data-ad-client="%s"`, publisherID), fmt.Sprintf(`data-ad-slot="%s"`, escapedSlotID)}
		if format != "" {
			attrs = append(attrs, fmt.Sprintf(`data-ad-format="%s"`, html.EscapeString(format)))
		}
		if slot.FullWidthResponsive {
			attrs = append(attrs, `data-full-width-responsive="true"`)
		}

		snippet := fmt.Sprintf(`<div class="ad-slot ad-slot--%s">
    <ins class="adsbygoogle" style="display:block" %s></ins>
    <script>(adsbygoogle = window.adsbygoogle || []).push({});</script>
</div>`, cssClass, strings.Join(attrs, " "))

		placements[slot.Placement] = append(placements[slot.Placement], snippet)
	}

	enabled := cfg.AutoAds || len(placements) > 0

	if !enabled {
		return RenderedAdvertising{Enabled: false}, nil
	}

	head := []string{
		fmt.Sprintf(`<script async src="https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js?client=%s" crossorigin="anonymous"></script>`, publisherID),
	}
	if cfg.AutoAds {
		head = append(head, fmt.Sprintf(`<meta name="google-adsense-account" content="%s">`, publisherID))
	}

	return RenderedAdvertising{
		Enabled:      enabled,
		HeadSnippets: head,
		Placements:   placements,
	}, nil
}

func normalizeGoogleFormat(input string) string {
	key := strings.TrimSpace(strings.ToLower(input))
	if key == "" {
		return "auto"
	}
	if _, ok := googleAdsFormats[key]; ok {
		return key
	}
	return "auto"
}

func sanitizePlacement(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastSeparator := false

	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastSeparator = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastSeparator = false
		case r == '-' || r == '_':
			if lastSeparator {
				continue
			}
			builder.WriteRune(r)
			lastSeparator = true
		case unicode.IsSpace(r) || r == ':' || r == '/' || r == '.':
			if lastSeparator {
				continue
			}
			builder.WriteRune('-')
			lastSeparator = true
		default:
			// Skip unsupported characters entirely.
			continue
		}
	}

	result := strings.Trim(builder.String(), "-_")
	return result
}

func cssClassFromPlacement(placement string) string {
	cleaned := sanitizePlacement(placement)
	if cleaned == "" {
		return "placement"
	}
	return cleaned
}
