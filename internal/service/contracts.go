package service

import (
	"mime/multipart"

	"github.com/golang-jwt/jwt/v5"

	"constructor-script-backend/internal/models"
)

type AuthUseCase interface {
	Register(models.RegisterRequest) (*models.User, error)
	Login(models.LoginRequest) (string, *models.User, error)
	ValidateToken(string) (*jwt.Token, error)
	GetAllUsers() ([]models.User, error)
	DeleteUser(uint) error
	UpdateUserRole(uint, string) error
	GetUserByID(uint) (*models.User, error)
	UpdateProfile(uint, string, string) (*models.User, error)
	ChangePassword(uint, string, string) error
	RefreshToken(string) (string, *models.User, error)
	UpdateUserStatus(uint, string) error
}

type CategoryUseCase interface {
	EnsureDefaultCategory() (*models.Category, bool, error)
	Create(models.CreateCategoryRequest) (*models.Category, error)
	GetAll() ([]models.Category, error)
	GetByID(uint) (*models.Category, error)
	Update(uint, models.CreateCategoryRequest) (*models.Category, error)
	Delete(uint) error
}

type CommentUseCase interface {
	Create(uint, uint, models.CreateCommentRequest) (*models.Comment, error)
	GetByPostID(uint) ([]models.Comment, error)
	GetAll() ([]models.Comment, error)
	Update(uint, uint, bool, models.UpdateCommentRequest) (*models.Comment, error)
	Delete(uint, uint, bool) error
	ApproveComment(uint) error
	RejectComment(uint) error
}

type PageUseCase interface {
	Create(models.CreatePageRequest) (*models.Page, error)
	Update(uint, models.UpdatePageRequest) (*models.Page, error)
	Delete(uint) error
	GetByID(uint) (*models.Page, error)
	GetBySlug(string) (*models.Page, error)
	GetAll() ([]models.Page, error)
	GetAllAdmin() ([]models.Page, error)
	PublishPage(uint) error
	UnpublishPage(uint) error
}

type PostUseCase interface {
	Create(models.CreatePostRequest, uint) (*models.Post, error)
	Update(uint, models.UpdatePostRequest, uint, bool) (*models.Post, error)
	Delete(uint, uint, bool) error
	GetByID(uint) (*models.Post, error)
	GetBySlug(string) (*models.Post, error)
	GetAll(int, int, *uint, *string, *uint) ([]models.Post, int64, error)
	GetAllAdmin(int, int) ([]models.Post, int64, error)
	GetPostsByTag(string, int, int) ([]models.Post, int64, error)
	GetTagWithPosts(string, int, int) (*models.Tag, []models.Post, int64, error)
	GetAllTags() ([]models.Tag, error)
	GetRelatedPosts(uint, int) ([]models.Post, error)
	PublishPost(uint) error
	UnpublishPost(uint) error
}

type SearchUseCase interface {
	Search(string, string, int) (*SearchResult, error)
	SuggestTags(string, int) ([]string, error)
}

type SetupUseCase interface {
	IsSetupComplete() (bool, error)
	GetSiteSettings(models.SiteSettings) (models.SiteSettings, error)
	CompleteSetup(models.SetupRequest) (*models.User, error)
	UpdateSiteSettings(models.UpdateSiteSettingsRequest) error
}

type UploadUseCase interface {
	ValidateImage(*multipart.FileHeader) error
	UploadImage(*multipart.FileHeader) (string, error)
	UploadMultipleImages([]*multipart.FileHeader) ([]string, error)
}
