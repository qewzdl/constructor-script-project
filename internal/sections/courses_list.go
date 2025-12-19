package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/constants"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/pkg/logger"
	courseservice "constructor-script-backend/plugins/courses/service"
)

// RegisterCoursesList registers the courses list section renderer.
func RegisterCoursesList(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("courses_list", renderCoursesList)
}

// RegisterCoursesListWithMetadata registers courses list with full metadata.
func RegisterCoursesListWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	desc := &SectionDescriptor{
		Renderer: renderCoursesList,
		Metadata: SectionMetadata{
			Type:        "courses_list",
			Name:        "Courses List",
			Description: "Displays available course packages",
			Category:    "content",
			Icon:        "book",
			Schema: map[string]interface{}{
				"limit": map[string]interface{}{
					"type":    "number",
					"default": constants.DefaultCourseListSectionLimit,
					"min":     1,
					"max":     constants.MaxCourseListSectionLimit,
				},
			},
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderCoursesList(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	mode := strings.TrimSpace(strings.ToLower(section.Mode))
	if mode == "" {
		mode = constants.CourseListModeCatalog
	}

	scripts := []string{"/static/js/courses-modal.js"}

	services := ctx.Services()
	if services == nil {
		return renderCoursesEmpty(prefix), scripts
	}

	coursePackageSvc, ok := services.CoursePackageService().(*courseservice.PackageService)
	if !ok || coursePackageSvc == nil {
		return renderCoursesEmpty(prefix), scripts
	}

	if mode == constants.CourseListModeOwned {
		return renderOwnedCourses(ctx, prefix, section), scripts
	}

	// Catalog mode
	checkoutSvc, _ := services.CourseCheckoutService().(*courseservice.CheckoutService)
	if checkoutSvc != nil {
		scripts = append(scripts, "/static/js/courses-checkout.js")
	}

	return renderCatalogCourses(ctx, prefix, section, coursePackageSvc), scripts
}

func renderCoursesEmpty(prefix string) string {
	emptyClass := fmt.Sprintf("%s__course-list-empty", prefix)
	return `<p class="` + emptyClass + `">Courses are not available right now.</p>`
}

func renderCatalogCourses(ctx RenderContext, prefix string, section models.Section, svc *courseservice.PackageService) string {
	emptyClass := fmt.Sprintf("%s__course-list-empty", prefix)

	limit := section.Limit
	if limit <= 0 {
		limit = constants.DefaultCourseListSectionLimit
	}
	if limit > constants.MaxCourseListSectionLimit {
		limit = constants.MaxCourseListSectionLimit
	}

	packages, err := svc.List()
	if err != nil {
		logger.Error(err, "Failed to load course packages for section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to load courses at the moment. Please try again later.</p>`
	}

	// Limit the results
	if len(packages) > limit {
		packages = packages[:limit]
	}

	if len(packages) == 0 {
		return `<p class="` + emptyClass + `">No courses available yet. Check back soon!</p>`
	}

	tmpl, err := ctx.CloneTemplates()
	if err != nil {
		logger.Error(err, "Failed to clone templates for course list section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to display courses at the moment.</p>`
	}

	var sb strings.Builder
	listClass := fmt.Sprintf("%s__course-list", prefix)
	sb.WriteString(`<div class="` + listClass + `">`)

	for _, pkg := range packages {
		card, renderErr := renderCourseCard(tmpl, &pkg, prefix)
		if renderErr != nil {
			logger.Error(renderErr, "Failed to render course card", map[string]interface{}{"package_id": pkg.ID, "section_id": section.ID})
			continue
		}
		sb.WriteString(card)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func renderOwnedCourses(ctx RenderContext, prefix string, section models.Section) string {
	emptyClass := fmt.Sprintf("%s__course-list-empty", prefix)

	// Extract owned courses data from element
	data := extractOwnedCourseSectionData(section)

	if len(data.Courses) == 0 {
		emptyMsg := data.EmptyMessage
		if emptyMsg == "" {
			emptyMsg = "You don't have any courses yet."
		}
		return `<p class="` + emptyClass + `">` + template.HTMLEscapeString(emptyMsg) + `</p>`
	}

	tmpl, err := ctx.CloneTemplates()
	if err != nil {
		logger.Error(err, "Failed to clone templates for owned courses section", map[string]interface{}{"section_id": section.ID})
		return `<p class="` + emptyClass + `">Unable to display your courses at the moment.</p>`
	}

	var sb strings.Builder
	listClass := fmt.Sprintf("%s__course-list", prefix)
	sb.WriteString(`<div class="` + listClass + `">`)

	for _, userPkg := range data.Courses {
		card, renderErr := renderUserCourseCard(tmpl, &userPkg, prefix)
		if renderErr != nil {
			logger.Error(renderErr, "Failed to render user course card", map[string]interface{}{"package_id": userPkg.Package.ID})
			continue
		}
		sb.WriteString(card)
	}

	sb.WriteString(`</div>`)
	return sb.String()
}

func renderCourseCard(tmpl *template.Template, pkg *models.CoursePackage, prefix string) (string, error) {
	data := map[string]interface{}{
		"Package": pkg,
		"Prefix":  prefix,
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "course-card", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func renderUserCourseCard(tmpl *template.Template, userPkg *models.UserCoursePackage, prefix string) (string, error) {
	data := map[string]interface{}{
		"UserPackage": userPkg,
		"Prefix":      prefix,
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "user-course-card", data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type ownedCourseSectionData struct {
	Courses      []models.UserCoursePackage
	EmptyMessage string
}

func extractOwnedCourseSectionData(section models.Section) ownedCourseSectionData {
	var result ownedCourseSectionData

	for _, sectionElem := range section.Elements {
		switch data := sectionElem.Content.(type) {
		case ownedCourseSectionData:
			return data
		case *ownedCourseSectionData:
			if data != nil {
				return *data
			}
		case []models.UserCoursePackage:
			result.Courses = data
		case *[]models.UserCoursePackage:
			if data != nil {
				result.Courses = *data
			}
		case map[string]interface{}:
			if message, ok := data["empty_message"].(string); ok {
				trimmed := strings.TrimSpace(message)
				if trimmed != "" {
					result.EmptyMessage = trimmed
				}
			}
		}
	}

	return result
}
