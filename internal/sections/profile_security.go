package sections

import (
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

const (
	profileSecurityDefaultTitle       = "Security"
	profileSecurityDefaultDescription = "Change your password regularly and review connected devices."
	profileSecurityDefaultButton      = "Update password"
)

// RegisterProfileSecurity registers the profile security form renderer.
func RegisterProfileSecurity(reg *Registry) {
	if reg == nil {
		return
	}

	reg.MustRegister("profile_security", renderProfileSecurity)
}

func renderProfileSecurity(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	if title == "" {
		title = profileSecurityDefaultTitle
	}

	description := strings.TrimSpace(getString(content, "description"))
	if description == "" {
		description = profileSecurityDefaultDescription
	}

	buttonLabel := strings.TrimSpace(getString(content, "button_label"))
	if buttonLabel == "" {
		buttonLabel = profileSecurityDefaultButton
	}

	action := strings.TrimSpace(getString(content, "action"))
	if action == "" {
		action = "/api/v1/profile/password"
	}

	username := getString(content, "username")

	var sb strings.Builder
	sb.WriteString(`<section class="profile-card" aria-labelledby="security-title">`)
	sb.WriteString(`<header class="profile-card__header">`)
	sb.WriteString(`<h2 id="security-title" class="profile-card__title">`)
	sb.WriteString(template.HTMLEscapeString(title))
	sb.WriteString(`</h2>`)
	sb.WriteString(`<p class="profile-card__description">`)
	sb.WriteString(template.HTMLEscapeString(description))
	sb.WriteString(`</p>`)
	sb.WriteString(`</header>`)

	sb.WriteString(`<div class="profile__alert" id="profile-password-alert" role="alert" hidden></div>`)

	sb.WriteString(`<form id="password-form" class="profile-form" method="post" data-action="`)
	sb.WriteString(template.HTMLEscapeString(action))
	sb.WriteString(`" novalidate>`)

	sb.WriteString(`<input type="text" name="username" value="`)
	sb.WriteString(template.HTMLEscapeString(username))
	sb.WriteString(`" autocomplete="username" hidden />`)

	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="current-password">Current password</label>`)
	sb.WriteString(`<input id="current-password" name="old_password" type="password" class="form-field__input" autocomplete="current-password" required />`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<div class="form-grid">`)
	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="new-password">New password</label>`)
	sb.WriteString(`<input id="new-password" name="new_password" type="password" class="form-field__input" autocomplete="new-password" required />`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="confirm-password">Confirm password</label>`)
	sb.WriteString(`<input id="confirm-password" name="confirm_password" type="password" class="form-field__input" autocomplete="new-password" required />`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<button type="submit" class="button button--secondary">`)
	sb.WriteString(template.HTMLEscapeString(buttonLabel))
	sb.WriteString(`</button>`)

	sb.WriteString(`</form>`)
	sb.WriteString(`</section>`)

	return sb.String(), nil
}
