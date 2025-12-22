package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"constructor-script-backend/internal/models"
)

const (
	AssetTypeVideo      = "video"
	AssetTypeAttachment = "attachment"

	defaultMaterialTokenTTL = 4 * time.Hour
	courseAssetBasePath     = "/api/v1/courses/assets/"
)

// MaterialProtection issues short-lived, user-bound URLs for course assets so
// raw upload links are not exposed to end users.
type MaterialProtection struct {
	signingKey []byte
	tokenTTL   time.Duration
}

// NewMaterialProtection creates a signer for course assets.
func NewMaterialProtection(signingKey string) *MaterialProtection {
	key := strings.TrimSpace(signingKey)
	if key == "" {
		return nil
	}
	return &MaterialProtection{
		signingKey: []byte(key),
		tokenTTL:   defaultMaterialTokenTTL,
	}
}

// SetTokenTTL overrides the default token lifetime.
func (p *MaterialProtection) SetTokenTTL(ttl time.Duration) {
	if p == nil {
		return
	}
	if ttl <= 0 {
		p.tokenTTL = defaultMaterialTokenTTL
		return
	}
	p.tokenTTL = ttl
}

// Enabled reports whether protection can sign and validate URLs.
func (p *MaterialProtection) Enabled() bool {
	return p != nil && len(p.signingKey) > 0
}

// ProtectCourseForUser replaces file URLs in the provided course with
// user-bound, time-limited URLs. The original struct is mutated in place.
func (p *MaterialProtection) ProtectCourseForUser(course *models.UserCoursePackage, userID uint) *models.UserCoursePackage {
	if course == nil || userID == 0 || !p.Enabled() {
		return course
	}

	packageID := course.Package.ID
	for topicIndex := range course.Package.Topics {
		topic := &course.Package.Topics[topicIndex]
		for stepIndex := range topic.Steps {
			step := &topic.Steps[stepIndex]
			if step.StepType != models.CourseTopicStepTypeVideo || step.Video == nil {
				continue
			}
			p.protectVideoAssets(userID, packageID, step.Video)
		}
	}

	return course
}

func (p *MaterialProtection) protectVideoAssets(userID, packageID uint, video *models.CourseVideo) {
	if video == nil {
		return
	}

	if p.isManagedUpload(video.FileURL) {
		if signed, err := p.signVideoURL(userID, packageID, video.ID); err == nil && signed != "" {
			video.FileURL = signed
		}
	}

	// Avoid leaking the raw filename to clients.
	video.Filename = ""

	for idx := range video.Attachments {
		attachment := &video.Attachments[idx]
		if !p.isManagedUpload(attachment.URL) {
			continue
		}
		if signed, err := p.signAttachmentURL(userID, packageID, video.ID, idx); err == nil && signed != "" {
			attachment.URL = signed
		}
	}
}

func (p *MaterialProtection) signVideoURL(userID, packageID, videoID uint) (string, error) {
	if !p.Enabled() || userID == 0 || packageID == 0 || videoID == 0 {
		return "", errors.New("material protection not configured")
	}

	token, err := p.buildToken(AssetTokenClaims{
		UserID:    userID,
		PackageID: packageID,
		VideoID:   videoID,
		Type:      AssetTypeVideo,
	})
	if err != nil {
		return "", err
	}

	return courseAssetBasePath + url.PathEscape(token), nil
}

func (p *MaterialProtection) signAttachmentURL(userID, packageID, videoID uint, attachmentIndex int) (string, error) {
	if !p.Enabled() || userID == 0 || packageID == 0 || videoID == 0 || attachmentIndex < 0 {
		return "", errors.New("material protection not configured")
	}

	idx := attachmentIndex
	token, err := p.buildToken(AssetTokenClaims{
		UserID:          userID,
		PackageID:       packageID,
		VideoID:         videoID,
		Type:            AssetTypeAttachment,
		AttachmentIndex: &idx,
	})
	if err != nil {
		return "", err
	}

	return courseAssetBasePath + url.PathEscape(token), nil
}

func (p *MaterialProtection) buildToken(claims AssetTokenClaims) (string, error) {
	if !p.Enabled() {
		return "", errors.New("material protection not configured")
	}
	if claims.UserID == 0 || claims.PackageID == 0 || claims.VideoID == 0 || strings.TrimSpace(claims.Type) == "" {
		return "", errors.New("invalid asset claims")
	}
	claims.Type = strings.ToLower(strings.TrimSpace(claims.Type))
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(p.tokenTTL))
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.signingKey)
}

// ParseToken validates and parses a signed asset token.
func (p *MaterialProtection) ParseToken(raw string) (*AssetTokenClaims, error) {
	if !p.Enabled() {
		return nil, errors.New("material protection not configured")
	}

	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("asset token is required")
	}

	token, err := jwt.ParseWithClaims(trimmed, &AssetTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return p.signingKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AssetTokenClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid asset token")
	}

	return claims, nil
}

func (p *MaterialProtection) isManagedUpload(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed == nil {
		return false
	}

	path := parsed.Path
	if path == "" {
		path = trimmed
	}

	return strings.HasPrefix(path, "/uploads/")
}

type AssetTokenClaims struct {
	jwt.RegisteredClaims
	UserID          uint   `json:"uid"`
	PackageID       uint   `json:"pkg"`
	VideoID         uint   `json:"vid"`
	AttachmentIndex *int   `json:"att,omitempty"`
	Type            string `json:"typ"`
}
