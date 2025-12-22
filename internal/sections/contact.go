package sections

import (
	"fmt"
	"html/template"
	"strings"

	"constructor-script-backend/internal/models"
)

// RegisterContact registers the contact section renderer.
func RegisterContact(reg *Registry) {
	if reg == nil {
		return
	}
	reg.RegisterSafe("contact", renderContact)
}

// RegisterContactWithMetadata registers the contact section with metadata support.
func RegisterContactWithMetadata(reg *RegistryWithMetadata) {
	if reg == nil {
		return
	}

	desc := &SectionDescriptor{
		Renderer: renderContact,
		Metadata: SectionMetadata{
			Type:        "contact",
			Name:        "Contact",
			Description: "Displays contact methods alongside a short inquiry form.",
			Category:    "support",
			Icon:        "phone",
			Schema: map[string]interface{}{
				"email": map[string]interface{}{
					"type":        "string",
					"label":       "Contact email",
					"placeholder": "team@example.com",
				},
				"phone": map[string]interface{}{
					"type":        "string",
					"label":       "Phone number",
					"placeholder": "+1 (555) 123-4567",
				},
				"location": map[string]interface{}{
					"type":        "string",
					"label":       "Location",
					"placeholder": "City, country or timezone",
				},
				"hours": map[string]interface{}{
					"type":        "string",
					"label":       "Availability",
					"placeholder": "Mon-Fri, 9am-6pm",
				},
				"response_time": map[string]interface{}{
					"type":        "string",
					"label":       "Response time",
					"placeholder": "We respond within one business day",
				},
				"kicker": map[string]interface{}{
					"type":        "string",
					"label":       "Eyebrow label",
					"placeholder": "Prefer email or chat?",
				},
				"note": map[string]interface{}{
					"type":        "textarea",
					"label":       "Helper text",
					"placeholder": "Tell visitors what to expect when they contact you.",
				},
				"form_title": map[string]interface{}{
					"type":        "string",
					"label":       "Form title",
					"placeholder": "Send us a note",
				},
				"form_action": map[string]interface{}{
					"type":              "url",
					"label":             "Form action (URL or mailto)",
					"placeholder":       "mailto:team@example.com",
					"allowAnchorPicker": true,
				},
				"form_submit_label": map[string]interface{}{
					"type":        "string",
					"label":       "Submit button label",
					"placeholder": "Send message",
				},
				"privacy_note": map[string]interface{}{
					"type":        "string",
					"label":       "Privacy note",
					"placeholder": "We only use your details to reply.",
				},
			},
		},
	}

	reg.RegisterWithMetadata(desc)
}

func renderContact(ctx RenderContext, prefix string, elem models.SectionElement) (string, []string) {
	section, ok := extractSection(elem)
	if !ok {
		return "", nil
	}

	settings := section.Settings
	if settings == nil {
		settings = map[string]interface{}{}
	}

	get := func(key string) string {
		return strings.TrimSpace(getString(settings, key))
	}

	email := get("email")
	if email == "" {
		if provider, ok := ctx.(interface{ ContactEmail() string }); ok {
			email = strings.TrimSpace(provider.ContactEmail())
		}
	}
	phone := get("phone")
	location := get("location")
	hours := get("hours")
	responseTime := get("response_time")
	formTitle := get("form_title")
	if formTitle == "" {
		formTitle = "Send us a note"
	}
	formSubmitLabel := get("form_submit_label")
	if formSubmitLabel == "" {
		formSubmitLabel = "Send message"
	}
	formAction := strings.TrimSpace(get("form_action"))
	if formAction == "" && email != "" {
		formAction = "mailto:" + email
	} else if looksLikeEmail(formAction) {
		formAction = "mailto:" + formAction
	}
	formSubject := strings.TrimSpace(get("form_subject"))
	if formSubject == "" {
		formSubject = "New inquiry from your site"
	}
	privacyNote := get("privacy_note")
	if privacyNote == "" {
		privacyNote = "We only use your details to reply."
	}

	containerClass := fmt.Sprintf("%s__contact", prefix)
	gridClass := fmt.Sprintf("%s__contact-grid", prefix)
	infoClass := fmt.Sprintf("%s__contact-info", prefix)
	detailsClass := fmt.Sprintf("%s__contact-details", prefix)
	cardClass := fmt.Sprintf("%s__contact-card", prefix)
	labelClass := fmt.Sprintf("%s__contact-label", prefix)
	valueClass := fmt.Sprintf("%s__contact-value", prefix)
	hintClass := fmt.Sprintf("%s__contact-hint", prefix)
	formClass := fmt.Sprintf("%s__contact-form", prefix)
	formHeaderClass := fmt.Sprintf("%s__contact-form-header", prefix)
	formTitleClass := fmt.Sprintf("%s__contact-form-title", prefix)
	formFooterClass := fmt.Sprintf("%s__contact-form-footer", prefix)
	privacyClass := fmt.Sprintf("%s__contact-privacy", prefix)
	pillClass := fmt.Sprintf("%s__contact-pill", prefix)

	var detailBlocks []string

	if email != "" {
		mailHref := "mailto:" + email
		hint := ""
		if responseTime != "" {
			hint = `<p class="` + hintClass + `">` + template.HTMLEscapeString(responseTime) + `</p>`
		}
		detailBlocks = append(detailBlocks, fmt.Sprintf(
			`<article class="%s"><span class="%s">Email</span><a class="%s" href="%s">%s</a>%s</article>`,
			cardClass,
			labelClass,
			valueClass,
			template.HTMLEscapeString(mailHref),
			template.HTMLEscapeString(email),
			hint,
		))
	}

	if phone != "" {
		phoneHref := normalisePhoneLink(phone)
		if phoneHref != "" {
			phoneHref = "tel:" + phoneHref
		}
		linkOpen := ""
		linkClose := ""
		if phoneHref != "" {
			linkOpen = `<a class="` + valueClass + `" href="` + template.HTMLEscapeString(phoneHref) + `">`
			linkClose = `</a>`
		}
		detailBlocks = append(detailBlocks, fmt.Sprintf(
			`<article class="%s"><span class="%s">Phone</span>%s%s%s</article>`,
			cardClass,
			labelClass,
			linkOpen,
			template.HTMLEscapeString(phone),
			linkClose,
		))
	}

	if location != "" {
		detailBlocks = append(detailBlocks, fmt.Sprintf(
			`<article class="%s"><span class="%s">Location</span><p class="%s">%s</p></article>`,
			cardClass,
			labelClass,
			valueClass,
			template.HTMLEscapeString(location),
		))
	}

	if hours != "" {
		detailBlocks = append(detailBlocks, fmt.Sprintf(
			`<article class="%s"><span class="%s">Hours</span><p class="%s">%s</p></article>`,
			cardClass,
			labelClass,
			valueClass,
			template.HTMLEscapeString(hours),
		))
	}

	var infoBuilder strings.Builder
	infoBuilder.WriteString(`<div class="` + infoClass + `">`)
	if len(detailBlocks) > 0 {
		infoBuilder.WriteString(`<div class="` + detailsClass + `">` + strings.Join(detailBlocks, "") + `</div>`)
	}
	infoBuilder.WriteString(`</div>`)

	formID := template.HTMLEscapeString(section.ID)
	nameFieldID := fmt.Sprintf("contact-name-%s", formID)
	emailFieldID := fmt.Sprintf("contact-email-%s", formID)
	topicFieldID := fmt.Sprintf("contact-topic-%s", formID)
	messageFieldID := fmt.Sprintf("contact-message-%s", formID)

	actionAttr := ""
	mailtoTarget := ""
	if strings.HasPrefix(strings.ToLower(formAction), "mailto:") {
		mailtoTarget = formAction
	} else if formAction != "" {
		actionAttr = ` action="` + template.HTMLEscapeString(formAction) + `"`
	}
	if mailtoTarget == "" && email != "" {
		mailtoTarget = "mailto:" + email
	}
	mailtoAttr := ""
	if mailtoTarget != "" {
		mailtoAttr = ` data-contact-mailto="` + template.HTMLEscapeString(mailtoTarget) + `"`
	}
	subjectAttr := ` data-contact-subject="` + template.HTMLEscapeString(formSubject) + `"`

	var formBuilder strings.Builder
	formBuilder.WriteString(`<form class="` + formClass + `" method="post"` + actionAttr + mailtoAttr + subjectAttr + ` enctype="text/plain" data-contact-form>`)
	formBuilder.WriteString(`<div class="` + formHeaderClass + `">`)
	formBuilder.WriteString(`<span class="` + pillClass + `">Contact form</span>`)
	formBuilder.WriteString(`<h3 class="` + formTitleClass + `">` + template.HTMLEscapeString(formTitle) + `</h3>`)
	formBuilder.WriteString(`</div>`)

	formBuilder.WriteString(`<div class="form-grid">`)
	formBuilder.WriteString(
		`<div class="form-field"><label class="form-field__label" for="` + nameFieldID + `">Your name</label>` +
			`<input id="` + nameFieldID + `" name="name" type="text" class="form-field__input" autocomplete="name" required /></div>`,
	)
	formBuilder.WriteString(
		`<div class="form-field"><label class="form-field__label" for="` + emailFieldID + `">Work email</label>` +
			`<input id="` + emailFieldID + `" name="email" type="email" class="form-field__input" autocomplete="email" required /></div>`,
	)
	formBuilder.WriteString(`</div>`)

	formBuilder.WriteString(`<div class="form-grid">`)
	formBuilder.WriteString(
		`<div class="form-field"><label class="form-field__label" for="` + topicFieldID + `">What do you need?</label>` +
			`<select id="` + topicFieldID + `" name="topic" class="form-field__input">` +
			`<option value="support">Support</option>` +
			`<option value="partnership">Partnership</option>` +
			`<option value="demo">Product demo</option>` +
			`<option value="other">Something else</option>` +
			`</select></div>`,
	)
	formBuilder.WriteString(
		`<div class="form-field"><label class="form-field__label" for="` + messageFieldID + `">Project details</label>` +
			`<textarea id="` + messageFieldID + `" name="message" class="form-field__input" rows="4" required placeholder="Share a few words about your goals"></textarea></div>`,
	)
	formBuilder.WriteString(`</div>`)

	formBuilder.WriteString(`<div class="` + formFooterClass + `">`)
	formBuilder.WriteString(`<p class="` + privacyClass + `">` + template.HTMLEscapeString(privacyNote) + `</p>`)
	formBuilder.WriteString(`<button type="submit" class="button button--primary">` + template.HTMLEscapeString(formSubmitLabel) + `</button>`)
	formBuilder.WriteString(`</div>`)
	formBuilder.WriteString(`</form>`)

	var sb strings.Builder
	sb.WriteString(`<div class="` + containerClass + `">`)
	sb.WriteString(`<div class="` + gridClass + `">`)
	sb.WriteString(infoBuilder.String())
	sb.WriteString(formBuilder.String())
	sb.WriteString(`</div>`)
	sb.WriteString(`</div>`)

	return sb.String(), []string{"/static/js/contact-form.js"}
}

func normalisePhoneLink(phone string) string {
	if phone == "" {
		return ""
	}

	var builder strings.Builder
	for _, r := range phone {
		if (r >= '0' && r <= '9') || r == '+' {
			builder.WriteRune(r)
		}
	}

	return builder.String()
}

func looksLikeEmail(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value != "" &&
		strings.Contains(value, "@") &&
		!strings.Contains(value, "://") &&
		!strings.HasPrefix(value, "mailto:")
}
