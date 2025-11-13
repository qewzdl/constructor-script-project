package host

import (
	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	languageservice "constructor-script-backend/plugins/language/service"
)

type Host interface {
	Config() *config.Config
	Cache() *cache.Cache
	Scheduler() *background.Scheduler
	ThemeManager() *theme.Manager

	Repositories() RepositoryAccess
	CoreServices() CoreServiceAccess
	Services(namespace string) Registry
	Handlers(namespace string) Registry

	TemplateHandler() *handlers.TemplateHandler
	SEOHandler() *handlers.SEOHandler
	ThemeHandler() *handlers.ThemeHandler
	AuthHandler() *handlers.AuthHandler
}

// Registry provides access to values managed by the host for a specific plugin
// namespace. Implementations must be safe for concurrent use.
type Registry interface {
	Get(key string) any
	Set(key string, value any)
	Delete(key string)
}

type RepositoryAccess interface {
	Category() repository.CategoryRepository
	Post() repository.PostRepository
	Tag() repository.TagRepository
	Comment() repository.CommentRepository
	Search() repository.SearchRepository
	Setting() repository.SettingRepository
	User() repository.UserRepository
	CourseVideo() repository.CourseVideoRepository
	CourseTopic() repository.CourseTopicRepository
	CoursePackage() repository.CoursePackageRepository
	CoursePackageAccess() repository.CoursePackageAccessRepository
	CourseTest() repository.CourseTestRepository
	ForumCategory() repository.ForumCategoryRepository
	ForumQuestion() repository.ForumQuestionRepository
	ForumAnswer() repository.ForumAnswerRepository
	ForumQuestionVote() repository.ForumQuestionVoteRepository
	ForumAnswerVote() repository.ForumAnswerVoteRepository
	ArchiveDirectory() repository.ArchiveDirectoryRepository
	ArchiveFile() repository.ArchiveFileRepository
}

type CoreServiceAccess interface {
	Auth() *service.AuthService
	Setup() *service.SetupService
	Theme() *service.ThemeService
	SocialLink() *service.SocialLinkService
	Menu() *service.MenuService
	Advertising() *service.AdvertisingService
	Upload() *service.UploadService
	Language() *languageservice.LanguageService
	SetLanguage(*languageservice.LanguageService)
}
