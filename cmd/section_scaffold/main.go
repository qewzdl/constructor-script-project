package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

type options struct {
	themePath            string
	sectionType          string
	variationValue       string
	sectionLabel         string
	sectionDescription   string
	variationLabel       string
	variationDescription string
	order                int
	supportsElements     bool
	force                bool
}

type sectionDefinition struct {
	Type                string                       `json:"type"`
	Label               string                       `json:"label,omitempty"`
	Order               int                          `json:"order,omitempty"`
	Category            string                       `json:"category,omitempty"`
	Icon                string                       `json:"icon,omitempty"`
	Description         string                       `json:"description,omitempty"`
	AllowedIn           []string                     `json:"allowed_in,omitempty"`
	Variations          []sectionVariationDefinition `json:"variations,omitempty"`
	AllowedElements     []string                     `json:"allowed_elements,omitempty"`
	SupportsElements    *bool                        `json:"supports_elements,omitempty"`
	SupportsHeaderImage *bool                        `json:"supports_header_image,omitempty"`
	Settings            map[string]interface{}       `json:"settings,omitempty"`
}

type sectionVariationDefinition struct {
	ID          string `json:"id,omitempty"`
	Value       string `json:"value"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	IsDefault   bool   `json:"is_default,omitempty"`
}

func main() {
	opts := options{}

	flag.StringVar(&opts.themePath, "theme", filepath.Join("themes", "default"), "Theme root directory")
	flag.StringVar(&opts.sectionType, "section", "", "Section type (for example: features, posts_list)")
	flag.StringVar(&opts.variationValue, "variation", "", "Variation value (for example: timeline, compact)")
	flag.StringVar(&opts.sectionLabel, "section-label", "", "Section label (used when creating a new section definition)")
	flag.StringVar(&opts.sectionDescription, "section-description", "", "Section description (used when creating a new section definition)")
	flag.StringVar(&opts.variationLabel, "variation-label", "", "Variation label")
	flag.StringVar(&opts.variationDescription, "variation-description", "", "Variation description")
	flag.IntVar(&opts.order, "order", 100, "Section order used when creating a new section definition")
	flag.BoolVar(&opts.supportsElements, "supports-elements", true, "Supports nested section elements (used when creating a new section definition)")
	flag.BoolVar(&opts.force, "force", false, "Overwrite existing CSS file and update an existing variation")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "section_scaffold: %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	themePath := filepath.Clean(strings.TrimSpace(opts.themePath))
	sectionType := normaliseSectionType(opts.sectionType)
	variationValue := normaliseVariationValue(opts.variationValue)

	if themePath == "" {
		return errors.New("theme path is required")
	}
	if sectionType == "" {
		return errors.New("section type is required")
	}
	if variationValue == "" {
		return errors.New("variation value is required")
	}

	sectionPath := filepath.Join(themePath, "data", "admin", "sections", sectionType+".json")
	cssFileName := fmt.Sprintf(
		"%s-variation-%s.css",
		strings.ReplaceAll(sectionType, "_", "-"),
		variationValue,
	)
	cssPath := filepath.Join(themePath, "static", "css", "sections", cssFileName)

	definition, exists, err := loadSectionDefinition(sectionPath)
	if err != nil {
		return err
	}

	if !exists {
		definition = sectionDefinition{
			Type:             sectionType,
			Order:            opts.order,
			SupportsElements: boolPointer(opts.supportsElements),
		}
	}

	if existingType := normaliseSectionType(definition.Type); existingType != "" && existingType != sectionType {
		return fmt.Errorf(
			"section definition %s contains type %q, expected %q",
			sectionPath,
			definition.Type,
			sectionType,
		)
	}
	definition.Type = sectionType

	sectionLabel := strings.TrimSpace(opts.sectionLabel)
	if sectionLabel == "" {
		sectionLabel = strings.TrimSpace(definition.Label)
	}
	if sectionLabel == "" {
		sectionLabel = humaniseToken(sectionType)
	}
	definition.Label = sectionLabel

	if !exists {
		definition.Description = strings.TrimSpace(opts.sectionDescription)
	} else if description := strings.TrimSpace(opts.sectionDescription); description != "" {
		definition.Description = description
	}

	variationLabel := strings.TrimSpace(opts.variationLabel)
	if variationLabel == "" {
		variationLabel = humaniseToken(variationValue)
	}

	variation := sectionVariationDefinition{
		ID:          variationID(sectionType, variationValue),
		Value:       variationValue,
		Label:       variationLabel,
		Description: strings.TrimSpace(opts.variationDescription),
	}

	addedVariation, err := upsertVariation(&definition, variation, opts.force)
	if err != nil {
		return err
	}
	ensureDefaultVariation(&definition)

	if err := writeSectionDefinition(sectionPath, definition); err != nil {
		return err
	}

	cssExists, err := fileExists(cssPath)
	if err != nil {
		return fmt.Errorf("inspect CSS scaffold %s: %w", cssPath, err)
	}

	cssWritten := false
	if !cssExists || opts.force {
		if err := writeVariationCSS(
			cssPath,
			buildVariationCSSTemplate(sectionType, variationValue, sectionLabel, variationLabel),
		); err != nil {
			return err
		}
		cssWritten = true
	}

	fmt.Printf("Section definition: %s\n", sectionPath)
	if exists {
		fmt.Println("Section definition status: updated")
	} else {
		fmt.Println("Section definition status: created")
	}
	if addedVariation {
		fmt.Printf("Variation: added %q\n", variationValue)
	} else {
		fmt.Printf("Variation: updated %q\n", variationValue)
	}

	if cssWritten {
		fmt.Printf("CSS scaffold: written %s\n", cssPath)
	} else {
		fmt.Printf("CSS scaffold: kept existing file %s (use --force to overwrite)\n", cssPath)
	}

	fmt.Println("Next step: add your styles to the generated CSS file.")

	return nil
}

func loadSectionDefinition(path string) (sectionDefinition, bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return sectionDefinition{}, false, nil
		}
		return sectionDefinition{}, false, fmt.Errorf("read section definition %s: %w", path, err)
	}

	definition := sectionDefinition{}
	if err := json.Unmarshal(content, &definition); err != nil {
		return sectionDefinition{}, true, fmt.Errorf("parse section definition %s: %w", path, err)
	}

	return definition, true, nil
}

func writeSectionDefinition(path string, definition sectionDefinition) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create section directory: %w", err)
	}

	content, err := json.MarshalIndent(definition, "", "    ")
	if err != nil {
		return fmt.Errorf("encode section definition %s: %w", path, err)
	}

	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write section definition %s: %w", path, err)
	}

	return nil
}

func writeVariationCSS(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create CSS directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write CSS scaffold %s: %w", path, err)
	}

	return nil
}

func upsertVariation(
	definition *sectionDefinition,
	variation sectionVariationDefinition,
	force bool,
) (bool, error) {
	if definition == nil {
		return false, errors.New("section definition is nil")
	}

	for index, existing := range definition.Variations {
		if normaliseVariationValue(existing.Value) != variation.Value {
			continue
		}

		if !force {
			return false, fmt.Errorf(
				"variation %q already exists in section %q (use --force to update)",
				variation.Value,
				definition.Type,
			)
		}

		variation.IsDefault = existing.IsDefault
		definition.Variations[index] = variation
		return false, nil
	}

	definition.Variations = append(definition.Variations, variation)
	return true, nil
}

func ensureDefaultVariation(definition *sectionDefinition) {
	if definition == nil || len(definition.Variations) == 0 {
		return
	}

	for _, variation := range definition.Variations {
		if variation.IsDefault {
			return
		}
	}

	definition.Variations[0].IsDefault = true
}

func buildVariationCSSTemplate(
	sectionType string,
	variationValue string,
	sectionLabel string,
	variationLabel string,
) string {
	sectionClass := strings.ReplaceAll(sectionType, "_", "-")

	return fmt.Sprintf(
		`/* Auto-generated section variation scaffold.
 * Section: %s (%s)
 * Variation: %s (%s)
 */

.page-view__section--%s.page-view__section--variation-%s {
    /* Wrapper-level overrides for this variation. */
}

.page-view__section--%s.page-view__section--variation-%s .page-view__section-container {
    /* Optional container/layout overrides. */
}
`,
		sectionLabel,
		sectionType,
		variationLabel,
		variationValue,
		sectionClass,
		variationValue,
		sectionClass,
		variationValue,
	)
}

func variationID(sectionType string, variationValue string) string {
	return strings.ReplaceAll(sectionType, "_", "-") + "-" + variationValue
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func boolPointer(value bool) *bool {
	v := value
	return &v
}

func humaniseToken(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return ""
	}

	replaced := strings.NewReplacer("_", " ", "-", " ").Replace(cleaned)
	parts := strings.Fields(replaced)
	if len(parts) == 0 {
		return ""
	}

	for index, part := range parts {
		lower := strings.ToLower(part)
		runes := []rune(lower)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[index] = string(runes)
	}

	return strings.Join(parts, " ")
}

func normaliseSectionType(value string) string {
	return normaliseIdentifier(value, '_')
}

func normaliseVariationValue(value string) string {
	return normaliseIdentifier(value, '-')
}

func normaliseIdentifier(value string, separator rune) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	builder.Grow(len(trimmed))

	lastSeparator := false
	for _, r := range trimmed {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastSeparator = false
		case r == '_' || r == '-' || unicode.IsSpace(r):
			if builder.Len() == 0 || lastSeparator {
				continue
			}
			builder.WriteRune(separator)
			lastSeparator = true
		default:
			if builder.Len() == 0 || lastSeparator {
				continue
			}
			builder.WriteRune(separator)
			lastSeparator = true
		}
	}

	return strings.Trim(builder.String(), string(separator))
}
