package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/url"
	"path"
	"strings"
	"time"

	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
)

type User struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Username string                 `gorm:"uniqueIndex;not null" json:"username"`
	Email    string                 `gorm:"uniqueIndex;not null" json:"email"`
	Password string                 `gorm:"not null" json:"-"`
	Role     authorization.UserRole `gorm:"type:varchar(32);default:'user'" json:"role"`

	Status string `gorm:"default:'active'" json:"status"`

	Posts    []Post    `gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Comments []Comment `gorm:"foreignKey:AuthorID" json:"comments,omitempty"`
}

type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string `gorm:"uniqueIndex;not null" json:"slug"`
	Description string `json:"description"`

	Order int `gorm:"default:0" json:"order"`

	Posts []Post `gorm:"foreignKey:CategoryID" json:"posts,omitempty"`
}

type Post struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string     `gorm:"not null" json:"title"`
	Slug        string     `gorm:"uniqueIndex;not null" json:"slug"`
	Description string     `json:"description"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	Excerpt     string     `json:"excerpt"`
	FeaturedImg string     `json:"featured_img"`
	Published   bool       `gorm:"default:false" json:"published"`
	PublishAt   *time.Time `gorm:"index" json:"publish_at,omitempty"`
	PublishedAt *time.Time `gorm:"index" json:"published_at,omitempty"`
	Views       int        `gorm:"default:0" json:"views"`

	Sections PostSections `gorm:"type:jsonb" json:"sections"`
	Template string       `gorm:"default:'post'" json:"template"`

	AuthorID   uint     `gorm:"not null" json:"author_id"`
	Author     User     `gorm:"foreignKey:AuthorID" json:"author"`
	CategoryID uint     `json:"category_id"`
	Category   Category `gorm:"foreignKey:CategoryID" json:"category"`

	Tags     []Tag     `gorm:"many2many:post_tags;" json:"tags,omitempty"`
	Comments []Comment `gorm:"foreignKey:PostID" json:"comments,omitempty"`
}

type PostViewStat struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PostID uint      `gorm:"not null;index:idx_post_view_stats_post_date,priority:1" json:"post_id"`
	Date   time.Time `gorm:"type:date;not null;index:idx_post_view_stats_post_date,priority:2" json:"date"`
	Views  int64     `gorm:"not null;default:0" json:"views"`

	Post Post `gorm:"foreignKey:PostID" json:"-"`
}

type Tag struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string     `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string     `gorm:"uniqueIndex;not null" json:"slug"`
	UnusedSince *time.Time `gorm:"index" json:"unused_since,omitempty"`
	Posts       []Post     `gorm:"many2many:post_tags;" json:"posts,omitempty"`
}

type Comment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Content  string `gorm:"type:text;not null" json:"content"`
	Approved bool   `gorm:"default:true" json:"approved"`

	PostID uint `gorm:"not null" json:"post_id"`
	Post   Post `gorm:"foreignKey:PostID" json:"post,omitempty"`

	AuthorID uint `gorm:"not null" json:"author_id"`
	Author   User `gorm:"foreignKey:AuthorID" json:"author"`

	ParentID *uint      `json:"parent_id"`
	Parent   *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Replies  []*Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

type CourseVideo struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title           string `gorm:"not null" json:"title"`
	Description     string `json:"description"`
	FileURL         string `gorm:"not null" json:"file_url"`
	Filename        string `gorm:"not null" json:"filename"`
	DurationSeconds int    `gorm:"not null" json:"duration_seconds"`
}

type CourseTopic struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`

	Videos []CourseVideo `gorm:"-" json:"videos"`
}

type CoursePackage struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description"`
	PriceCents  int64  `gorm:"not null" json:"price_cents"`
	ImageURL    string `json:"image_url"`

	Topics []CourseTopic `gorm:"-" json:"topics"`
}

type CourseTopicVideo struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	TopicID  uint `gorm:"not null;index" json:"topic_id"`
	VideoID  uint `gorm:"not null;index" json:"video_id"`
	Position int  `gorm:"not null;default:0" json:"position"`

	Topic CourseTopic `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Video CourseVideo `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type CoursePackageTopic struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	PackageID uint `gorm:"not null;index" json:"package_id"`
	TopicID   uint `gorm:"not null;index" json:"topic_id"`
	Position  int  `gorm:"not null;default:0" json:"position"`

	Package CoursePackage `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
	Topic   CourseTopic   `gorm:"constraint:OnDelete:CASCADE;" json:"-"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=12,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type CreateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

type UpdateCommentRequest struct {
	Content  string `json:"content" binding:"required"`
	Approved *bool  `json:"approved"`
}

type CreateCourseVideoRequest struct {
	Title       string `form:"title" binding:"required"`
	Description string `form:"description"`
	Preferred   string `form:"preferred_name"`
}

type UpdateCourseVideoRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type CreateCourseTopicRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	VideoIDs    []uint `json:"video_ids"`
}

type UpdateCourseTopicRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type ReorderCourseTopicVideosRequest struct {
	VideoIDs []uint `json:"video_ids" binding:"required"`
}

type CreateCoursePackageRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents" binding:"required"`
	ImageURL    string `json:"image_url"`
	TopicIDs    []uint `json:"topic_ids"`
}

type UpdateCoursePackageRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents" binding:"required"`
	ImageURL    string `json:"image_url"`
}

type ReorderCoursePackageTopicsRequest struct {
	TopicIDs []uint `json:"topic_ids" binding:"required"`
}

type AuthResponse struct {
	Token     string `json:"token"`
	User      User   `json:"user"`
	CSRFToken string `json:"csrf_token,omitempty"`
}

type PostSections []Section

type Section struct {
	ID             string           `json:"id"`
	Type           string           `json:"type"`
	Title          string           `json:"title"`
	Image          string           `json:"image"`
	Limit          int              `json:"limit"`
	Order          int              `json:"order"`
	StyleGridItems *bool            `json:"style_grid_items,omitempty"`
	Elements       []SectionElement `json:"elements"`
}

type SectionElement struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Order   int         `json:"order"`
	Content interface{} `json:"content"`
}

type ParagraphContent struct {
	Text string `json:"text"`
}

type ImageContent struct {
	URL     string `json:"url"`
	Alt     string `json:"alt"`
	Caption string `json:"caption"`
}

// ContentSecurityPolicyDirectives contains additional CSP directive values keyed by directive name.
// Each directive maps to a slice of allowed source expressions that will be merged into the base policy.
type ContentSecurityPolicyDirectives map[string][]string

type JSONMap map[string]interface{}

func (m JSONMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = JSONMap{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan JSONMap")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		return err
	}

	*m = decoded
	return nil
}

type ImageGroupContent struct {
	Images []ImageContent `json:"images"`
	Layout string         `json:"layout"`
}

type ListContent struct {
	Items   []string `json:"items"`
	Ordered bool     `json:"ordered"`
}

func (ps *PostSections) Scan(value interface{}) error {
	if value == nil {
		*ps = PostSections{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to scan PostSections")
	}

	return json.Unmarshal(bytes, ps)
}

func (ps PostSections) Value() (driver.Value, error) {
	if len(ps) == 0 {
		return nil, nil
	}
	return json.Marshal(ps)
}

type CreatePostRequest struct {
	Title       string       `json:"title" binding:"required"`
	Description string       `json:"description"`
	Content     string       `json:"content"`
	Excerpt     string       `json:"excerpt"`
	FeaturedImg string       `json:"featured_img"`
	Published   bool         `json:"published"`
	CategoryID  uint         `json:"category_id"`
	TagNames    []string     `json:"tags"`
	Sections    []Section    `json:"sections"`
	Template    string       `json:"template"`
	PublishAt   OptionalTime `json:"publish_at"`
}

type UpdatePostRequest struct {
	Title       *string      `json:"title"`
	Description *string      `json:"description"`
	Content     *string      `json:"content"`
	Excerpt     *string      `json:"excerpt"`
	FeaturedImg *string      `json:"featured_img"`
	Published   *bool        `json:"published"`
	CategoryID  *uint        `json:"category_id"`
	TagNames    []string     `json:"tags"`
	Sections    *[]Section   `json:"sections"`
	Template    *string      `json:"template"`
	PublishAt   OptionalTime `json:"publish_at"`
}

type Page struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title       string       `gorm:"not null" json:"title"`
	Slug        string       `gorm:"uniqueIndex;not null" json:"slug"`
	Path        string       `gorm:"uniqueIndex;not null" json:"path"`
	Description string       `json:"description"`
	FeaturedImg string       `json:"featured_img"`
	Published   bool         `gorm:"default:false" json:"published"`
	PublishAt   *time.Time   `gorm:"index" json:"publish_at,omitempty"`
	PublishedAt *time.Time   `gorm:"index" json:"published_at,omitempty"`
	Content     string       `gorm:"type:text" json:"content"`
	Sections    PostSections `gorm:"type:jsonb" json:"sections"`
	Template    string       `gorm:"default:'page'" json:"template"`
	HideHeader  bool         `gorm:"default:false" json:"hide_header"`

	Order int `gorm:"default:0" json:"order"`
}

type CreatePageRequest struct {
	Title       string       `json:"title" binding:"required"`
	Slug        string       `json:"slug"`
	Path        string       `json:"path"`
	Description string       `json:"description"`
	FeaturedImg string       `json:"featured_img"`
	Published   bool         `json:"published"`
	Content     string       `json:"content"`
	Sections    []Section    `json:"sections"`
	Template    string       `json:"template"`
	HideHeader  bool         `json:"hide_header"`
	Order       int          `json:"order"`
	PublishAt   OptionalTime `json:"publish_at"`
}

type UpdatePageRequest struct {
	Title       *string      `json:"title"`
	Path        *string      `json:"path"`
	Description *string      `json:"description"`
	FeaturedImg *string      `json:"featured_img"`
	Published   *bool        `json:"published"`
	Content     *string      `json:"content"`
	Sections    *[]Section   `json:"sections"`
	Template    *string      `json:"template"`
	HideHeader  *bool        `json:"hide_header"`
	Order       *int         `json:"order"`
	PublishAt   OptionalTime `json:"publish_at"`
}

type Setting struct {
	Key       string    `gorm:"primaryKey;size:191" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Plugin struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Slug            string     `gorm:"uniqueIndex;not null" json:"slug"`
	Name            string     `gorm:"not null" json:"name"`
	Version         string     `json:"version"`
	Description     string     `gorm:"type:text" json:"description"`
	Author          string     `json:"author"`
	Homepage        string     `json:"homepage"`
	Active          bool       `gorm:"default:false" json:"active"`
	InstalledAt     time.Time  `gorm:"not null" json:"installed_at"`
	Metadata        JSONMap    `gorm:"type:jsonb" json:"metadata"`
	LastActivatedAt *time.Time `json:"last_activated_at"`
}

type PluginInfo struct {
	Slug           string     `json:"slug"`
	Name           string     `json:"name"`
	Description    string     `json:"description,omitempty"`
	Version        string     `json:"version,omitempty"`
	Author         string     `json:"author,omitempty"`
	Homepage       string     `json:"homepage,omitempty"`
	Active         bool       `json:"active"`
	Installed      bool       `json:"installed"`
	InstalledAt    *time.Time `json:"installed_at,omitempty"`
	LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
	MissingFiles   bool       `json:"missing_files"`
	AdditionalData JSONMap    `json:"metadata,omitempty"`
}

type SiteSettings struct {
	Name                    string       `json:"name"`
	Description             string       `json:"description"`
	URL                     string       `json:"url"`
	Favicon                 string       `json:"favicon"`
	FaviconType             string       `json:"favicon_type"`
	Logo                    string       `json:"logo"`
	UnusedTagRetentionHours int          `json:"unused_tag_retention_hours"`
	SocialLinks             []SocialLink `json:"social_links"`
	MenuItems               []MenuItem   `json:"menu_items"`
	DefaultLanguage         string       `json:"default_language"`
	SupportedLanguages      []string     `json:"supported_languages"`
	Fonts                   []FontAsset  `json:"fonts"`
	FontPreconnects         []string     `json:"font_preconnects"`
}

type BackupSettings struct {
	Enabled       bool       `json:"enabled"`
	IntervalHours int        `json:"interval_hours"`
	LastRun       *time.Time `json:"last_run,omitempty"`
	NextRun       *time.Time `json:"next_run,omitempty"`
}

type UpdateBackupSettingsRequest struct {
	Enabled       bool `json:"enabled"`
	IntervalHours int  `json:"interval_hours" binding:"required,min=1,max=168"`
}

type ThemeInfo struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Version      string `json:"version,omitempty"`
	Author       string `json:"author,omitempty"`
	PreviewImage string `json:"preview_image,omitempty"`
	Active       bool   `json:"active"`
}

type UpdateSiteSettingsRequest struct {
	Name                    string   `json:"name" binding:"required"`
	Description             string   `json:"description"`
	URL                     string   `json:"url" binding:"required"`
	Favicon                 string   `json:"favicon"`
	Logo                    string   `json:"logo"`
	UnusedTagRetentionHours int      `json:"unused_tag_retention_hours" binding:"required,min=1"`
	DefaultLanguage         string   `json:"default_language"`
	SupportedLanguages      []string `json:"supported_languages"`
}

type FontAsset struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Snippet     string   `json:"snippet"`
	Preconnects []string `json:"preconnects,omitempty"`
	Order       int      `json:"order"`
	Enabled     bool     `json:"enabled"`
	Notes       string   `json:"notes,omitempty"`
}

type CreateFontAssetRequest struct {
	Name        string   `json:"name" binding:"required"`
	Snippet     string   `json:"snippet" binding:"required"`
	Preconnects []string `json:"preconnects"`
	Enabled     *bool    `json:"enabled"`
	Notes       string   `json:"notes"`
}

type UpdateFontAssetRequest struct {
	Name        *string   `json:"name"`
	Snippet     *string   `json:"snippet"`
	Preconnects *[]string `json:"preconnects"`
	Enabled     *bool     `json:"enabled"`
	Notes       *string   `json:"notes"`
}

type FontAssetOrder struct {
	ID    string `json:"id"`
	Order int    `json:"order"`
}

type ReorderFontAssetsRequest struct {
	Items []FontAssetOrder `json:"items"`
}

type HomepagePage struct {
	ID        uint       `json:"id"`
	Title     string     `json:"title"`
	Slug      string     `json:"slug"`
	Path      string     `json:"path"`
	Published bool       `json:"published"`
	PublishAt *time.Time `json:"publish_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type UpdateHomepageRequest struct {
	PageID *uint `json:"page_id"`
}

type AdvertisingSettings struct {
	Enabled   bool               `json:"enabled"`
	Provider  string             `json:"provider"`
	GoogleAds *GoogleAdsSettings `json:"google_ads,omitempty"`
}

type GoogleAdsSettings struct {
	PublisherID string          `json:"publisher_id"`
	AutoAds     bool            `json:"auto_ads"`
	Slots       []GoogleAdsSlot `json:"slots"`
}

type GoogleAdsSlot struct {
	Placement           string `json:"placement"`
	SlotID              string `json:"slot_id"`
	Format              string `json:"format"`
	FullWidthResponsive bool   `json:"full_width_responsive"`
}

type AdvertisingProviderMetadata struct {
	Key             string                 `json:"key"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	SupportsAutoAds bool                   `json:"supports_auto_ads"`
	Placements      []AdvertisingPlacement `json:"placements"`
	Formats         []AdvertisingFormat    `json:"formats"`
}

type AdvertisingPlacement struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Recommended bool   `json:"recommended,omitempty"`
}

type AdvertisingFormat struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type UpdateAdvertisingSettingsRequest struct {
	Enabled   bool               `json:"enabled"`
	Provider  string             `json:"provider"`
	GoogleAds *GoogleAdsSettings `json:"google_ads"`
}

func DetectFaviconType(favicon string) string {
	const defaultType = "image/x-icon"

	trimmed := strings.TrimSpace(favicon)
	if trimmed == "" {
		return defaultType
	}

	value := trimmed
	if parsed, err := url.Parse(trimmed); err == nil {
		if parsed.Path != "" {
			value = parsed.Path
		}
	}

	ext := strings.TrimPrefix(strings.ToLower(path.Ext(value)), ".")
	switch ext {
	case "png":
		return "image/png"
	case "svg":
		return "image/svg+xml"
	case "jpg", "jpeg":
		return "image/jpeg"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "ico":
		return "image/x-icon"
	default:
		return defaultType
	}
}

type SetupRequest struct {
	AdminUsername string `json:"admin_username" binding:"required,min=3,max=50"`
	AdminEmail    string `json:"admin_email" binding:"required,email"`
	AdminPassword string `json:"admin_password" binding:"required,min=8"`

	SiteName               string   `json:"site_name" binding:"required"`
	SiteDescription        string   `json:"site_description"`
	SiteURL                string   `json:"site_url" binding:"required"`
	SiteFavicon            string   `json:"site_favicon"`
	SiteLogo               string   `json:"site_logo"`
	SiteDefaultLanguage    string   `json:"site_default_language"`
	SiteSupportedLanguages []string `json:"site_supported_languages"`
}

type SocialLink struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name  string `gorm:"not null" json:"name"`
	URL   string `gorm:"not null" json:"url"`
	Icon  string `json:"icon"`
	Order int    `gorm:"default:0" json:"order"`
}

type CreateSocialLinkRequest struct {
	Name  string `json:"name" binding:"required"`
	URL   string `json:"url" binding:"required"`
	Icon  string `json:"icon"`
	Order *int   `json:"order"`
}

type UpdateSocialLinkRequest struct {
	Name  string `json:"name" binding:"required"`
	URL   string `json:"url" binding:"required"`
	Icon  string `json:"icon"`
	Order *int   `json:"order"`
}

type MenuItem struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Title    string `gorm:"not null" json:"title"`
	Label    string `gorm:"column:label;not null" json:"-"`
	URL      string `gorm:"not null" json:"url"`
	Location string `gorm:"type:varchar(32);not null;default:'header'" json:"location"`
	Order    int    `gorm:"default:0" json:"order"`
}

func (m *MenuItem) EnsureTextFields() {
	if m == nil {
		return
	}
	m.Title = strings.TrimSpace(m.Title)
	m.Label = strings.TrimSpace(m.Label)
	if m.Title == "" && m.Label != "" {
		m.Title = m.Label
	}
	if m.Label == "" && m.Title != "" {
		m.Label = m.Title
	}
}

func NormalizeMenuItems(items []MenuItem) []MenuItem {
	for i := range items {
		items[i].EnsureTextFields()
	}
	return items
}

type CreateMenuItemRequest struct {
	Title    string `json:"title" binding:"required"`
	URL      string `json:"url" binding:"required"`
	Location string `json:"location"`
	Order    *int   `json:"order"`
}

type UpdateMenuItemRequest struct {
	Title    string  `json:"title" binding:"required"`
	URL      string  `json:"url" binding:"required"`
	Location *string `json:"location"`
	Order    *int    `json:"order"`
}

type MenuOrder struct {
	ID    uint `json:"id"`
	Order int  `json:"order"`
}

type ReorderMenuItemsRequest struct {
	Orders []MenuOrder `json:"orders"`
}
