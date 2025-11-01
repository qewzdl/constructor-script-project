package host

import (
	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	bloghandlers "constructor-script-backend/plugins/blog/handlers"
	blogservice "constructor-script-backend/plugins/blog/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

type Host interface {
	Config() *config.Config
	Cache() *cache.Cache
	Scheduler() *background.Scheduler
	ThemeManager() *theme.Manager

	Repositories() RepositoryAccess
	CoreServices() CoreServiceAccess
	BlogServices() BlogServiceAccess
	BlogHandlers() BlogHandlerAccess

	TemplateHandler() *handlers.TemplateHandler
	SEOHandler() *handlers.SEOHandler
	ThemeHandler() *handlers.ThemeHandler
}

type RepositoryAccess interface {
	Category() repository.CategoryRepository
	Post() repository.PostRepository
	Tag() repository.TagRepository
	Comment() repository.CommentRepository
	Search() repository.SearchRepository
	Setting() repository.SettingRepository
	User() repository.UserRepository
}

type CoreServiceAccess interface {
	Auth() *service.AuthService
	Setup() *service.SetupService
	Theme() *service.ThemeService
	SocialLink() *service.SocialLinkService
	Menu() *service.MenuService
	Advertising() *service.AdvertisingService
	Language() *languageservice.LanguageService
	SetLanguage(*languageservice.LanguageService)
}

type BlogServiceAccess interface {
	Category() *blogservice.CategoryService
	SetCategory(*blogservice.CategoryService)
	Post() *blogservice.PostService
	SetPost(*blogservice.PostService)
	Comment() *blogservice.CommentService
	SetComment(*blogservice.CommentService)
	Search() *blogservice.SearchService
	SetSearch(*blogservice.SearchService)
}

type BlogHandlerAccess interface {
	Post() *bloghandlers.PostHandler
	SetPost(*bloghandlers.PostHandler)
	Category() *bloghandlers.CategoryHandler
	SetCategory(*bloghandlers.CategoryHandler)
	Comment() *bloghandlers.CommentHandler
	SetComment(*bloghandlers.CommentHandler)
	Search() *bloghandlers.SearchHandler
	SetSearch(*bloghandlers.SearchHandler)
}
