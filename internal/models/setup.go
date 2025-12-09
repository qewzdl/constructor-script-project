package models

import "time"

// SetupStep represents the current step in the setup process
type SetupStep string

const (
	SetupStepSiteInfo  SetupStep = "site_info"
	SetupStepAdmin     SetupStep = "admin"
	SetupStepLanguages SetupStep = "languages"
	SetupStepComplete  SetupStep = "complete"
)

// AllSetupSteps returns all setup steps in order
func AllSetupSteps() []SetupStep {
	return []SetupStep{
		SetupStepSiteInfo,
		SetupStepAdmin,
		SetupStepLanguages,
	}
}

// String returns the string representation of the step
func (s SetupStep) String() string {
	return string(s)
}

// IsValid checks if the step is valid
func (s SetupStep) IsValid() bool {
	switch s {
	case SetupStepSiteInfo, SetupStepAdmin, SetupStepLanguages, SetupStepComplete:
		return true
	default:
		return false
	}
}

// Next returns the next step, or empty if this is the last step
func (s SetupStep) Next() SetupStep {
	switch s {
	case SetupStepSiteInfo:
		return SetupStepAdmin
	case SetupStepAdmin:
		return SetupStepLanguages
	case SetupStepLanguages:
		return SetupStepComplete
	default:
		return ""
	}
}

// Previous returns the previous step, or empty if this is the first step
func (s SetupStep) Previous() SetupStep {
	switch s {
	case SetupStepAdmin:
		return SetupStepSiteInfo
	case SetupStepLanguages:
		return SetupStepAdmin
	default:
		return ""
	}
}

// SetupProgress stores the progress of the setup wizard
type SetupProgress struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CurrentStep string    `gorm:"type:varchar(50);not null;default:'site_info'" json:"current_step"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	// Step completion flags
	SiteInfoComplete  bool `gorm:"default:false" json:"site_info_complete"`
	AdminComplete     bool `gorm:"default:false" json:"admin_complete"`
	LanguagesComplete bool `gorm:"default:false" json:"languages_complete"`

	// Step data
	SiteInfo  SiteInfoData  `gorm:"embedded;embeddedPrefix:site_" json:"site_info,omitempty"`
	Admin     AdminData     `gorm:"embedded;embeddedPrefix:admin_" json:"admin,omitempty"`
	Languages LanguagesData `gorm:"embedded;embeddedPrefix:lang_" json:"languages,omitempty"`
}

// SiteInfoData contains site information fields
type SiteInfoData struct {
	Name        string `gorm:"type:varchar(255)" json:"name,omitempty"`
	Description string `gorm:"type:text" json:"description,omitempty"`
	URL         string `gorm:"type:varchar(500)" json:"url,omitempty"`
	Favicon     string `gorm:"type:varchar(500)" json:"favicon,omitempty"`
	Logo        string `gorm:"type:varchar(500)" json:"logo,omitempty"`
}

// AdminData contains admin account fields
type AdminData struct {
	Username string `gorm:"type:varchar(50)" json:"username,omitempty"`
	Email    string `gorm:"type:varchar(255)" json:"email,omitempty"`
	Password string `gorm:"type:varchar(255)" json:"-"` // hashed password
}

// LanguagesData contains language configuration fields
type LanguagesData struct {
	DefaultLanguage    string `gorm:"type:varchar(10)" json:"default_language,omitempty"`
	SupportedLanguages string `gorm:"type:text" json:"supported_languages,omitempty"` // comma-separated
}

// IsStepComplete checks if a specific step is completed
func (p *SetupProgress) IsStepComplete(step SetupStep) bool {
	if p == nil {
		return false
	}

	switch step {
	case SetupStepSiteInfo:
		return p.SiteInfoComplete
	case SetupStepAdmin:
		return p.AdminComplete
	case SetupStepLanguages:
		return p.LanguagesComplete
	default:
		return false
	}
}

// MarkStepComplete marks a step as completed and advances to next step
func (p *SetupProgress) MarkStepComplete(step SetupStep) {
	if p == nil {
		return
	}

	switch step {
	case SetupStepSiteInfo:
		p.SiteInfoComplete = true
		p.CurrentStep = SetupStepAdmin.String()
	case SetupStepAdmin:
		p.AdminComplete = true
		p.CurrentStep = SetupStepLanguages.String()
	case SetupStepLanguages:
		p.LanguagesComplete = true
		p.CurrentStep = SetupStepComplete.String()
	}
}

// AllStepsComplete checks if all steps are completed
func (p *SetupProgress) AllStepsComplete() bool {
	return p != nil && p.SiteInfoComplete && p.AdminComplete && p.LanguagesComplete
}

// SetupStepRequest represents a request to save data for a specific step
type SetupStepRequest struct {
	Step string `json:"step" binding:"required"`

	// Site info step fields
	SiteName        string `json:"site_name,omitempty"`
	SiteDescription string `json:"site_description,omitempty"`
	SiteURL         string `json:"site_url,omitempty"`
	SiteFavicon     string `json:"site_favicon,omitempty"`
	SiteLogo        string `json:"site_logo,omitempty"`

	// Admin step fields
	AdminUsername string `json:"admin_username,omitempty"`
	AdminEmail    string `json:"admin_email,omitempty"`
	AdminPassword string `json:"admin_password,omitempty"`

	// Languages step fields
	DefaultLanguage    string   `json:"default_language,omitempty"`
	SupportedLanguages []string `json:"supported_languages,omitempty"`
}

// ToSiteInfoData converts request to SiteInfoData
func (r *SetupStepRequest) ToSiteInfoData() SiteInfoData {
	return SiteInfoData{
		Name:        r.SiteName,
		Description: r.SiteDescription,
		URL:         r.SiteURL,
		Favicon:     r.SiteFavicon,
		Logo:        r.SiteLogo,
	}
}

// ToAdminData converts request to AdminData (password should be hashed separately)
func (r *SetupStepRequest) ToAdminData() AdminData {
	return AdminData{
		Username: r.AdminUsername,
		Email:    r.AdminEmail,
		Password: r.AdminPassword, // Will be hashed by service
	}
}

// ToLanguagesData converts request to LanguagesData
func (r *SetupStepRequest) ToLanguagesData() LanguagesData {
	supportedLangs := ""
	if len(r.SupportedLanguages) > 0 {
		supportedLangs = joinStrings(r.SupportedLanguages, ",")
	}

	return LanguagesData{
		DefaultLanguage:    r.DefaultLanguage,
		SupportedLanguages: supportedLangs,
	}
}

// SetupStatusResponse represents the response with setup status and progress
type SetupStatusResponse struct {
	SetupRequired bool           `json:"setup_required"`
	CurrentStep   string         `json:"current_step,omitempty"`
	Progress      *SetupProgress `json:"progress,omitempty"`
	Site          SiteSettings   `json:"site"`
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
