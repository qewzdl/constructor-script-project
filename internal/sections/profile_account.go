package sections

import (
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

const (
	profileAccountDefaultTitle       = "Account details"
	profileAccountDefaultDescription = "The information below appears in comments and author bylines."
	profileAccountDefaultButton      = "Save changes"
)

// RegisterProfileAccount registers the profile account details form renderer.
func RegisterProfileAccount(reg *Registry) {
	if reg == nil {
		return
	}

	reg.RegisterSafe("profile_account_details", renderProfileAccount)
}

func renderProfileAccount(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	content := sectionContent(elem)

	title := strings.TrimSpace(getString(content, "title"))
	if title == "" {
		title = profileAccountDefaultTitle
	}

	description := strings.TrimSpace(getString(content, "description"))
	if description == "" {
		description = profileAccountDefaultDescription
	}

	buttonLabel := strings.TrimSpace(getString(content, "button_label"))
	if buttonLabel == "" {
		buttonLabel = profileAccountDefaultButton
	}

	action := strings.TrimSpace(getString(content, "action"))
	if action == "" {
		action = "/api/v1/profile"
	}

	username := getString(content, "username")
	email := getString(content, "email")
	role := getString(content, "role")
	if strings.TrimSpace(role) == "" {
		role = "user"
	}

	var sb strings.Builder
	sb.WriteString(`<section class="profile-card" aria-labelledby="profile-details-title">`)
	sb.WriteString(`<header class="profile-card__header">`)
	sb.WriteString(`<h2 id="profile-details-title" class="profile-card__title">`)
	sb.WriteString(template.HTMLEscapeString(title))
	sb.WriteString(`</h2>`)
	sb.WriteString(`<p class="profile-card__description">`)
	sb.WriteString(template.HTMLEscapeString(description))
	sb.WriteString(`</p>`)
	sb.WriteString(`</header>`)

	sb.WriteString(`<div class="profile__alert" id="profile-details-alert" role="alert" hidden></div>`)

	sb.WriteString(`<form id="profile-form" class="profile-form" method="post" data-action="`)
	sb.WriteString(template.HTMLEscapeString(action))
	sb.WriteString(`" novalidate>`)

	sb.WriteString(`<div class="form-grid">`)
	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="profile-username">Username</label>`)
	sb.WriteString(`<input id="profile-username" name="username" type="text" class="form-field__input" placeholder="Display name" autocomplete="username" value="`)
	sb.WriteString(template.HTMLEscapeString(username))
	sb.WriteString(`" required />`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="profile-email">Email</label>`)
	sb.WriteString(`<input id="profile-email" name="email" type="email" class="form-field__input" placeholder="name@example.com" autocomplete="email" value="`)
	sb.WriteString(template.HTMLEscapeString(email))
	sb.WriteString(`" required />`)
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<div class="form-field">`)
	sb.WriteString(`<label class="form-field__label" for="profile-role">Role</label>`)
	sb.WriteString(`<input id="profile-role" name="role" type="text" class="form-field__input" value="`)
	sb.WriteString(template.HTMLEscapeString(role))
	sb.WriteString(`" readonly aria-readonly="true" />`)
	sb.WriteString(`</div>`)

	sb.WriteString(`<button type="submit" class="button button--primary">`)
	sb.WriteString(template.HTMLEscapeString(buttonLabel))
	sb.WriteString(`</button>`)

	sb.WriteString(`</form>`)
	sb.WriteString(`</section>`)

	return sb.String(), nil
}
