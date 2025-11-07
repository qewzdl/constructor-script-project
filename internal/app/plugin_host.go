package app

import (
	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	bloghandlers "constructor-script-backend/plugins/blog/handlers"
	blogservice "constructor-script-backend/plugins/blog/service"
	coursehandlers "constructor-script-backend/plugins/courses/handlers"
	courseservice "constructor-script-backend/plugins/courses/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

type applicationRepositoryAccess struct {
	app *Application
}

type applicationCoreServices struct {
	app *Application
}

type applicationBlogServices struct {
	app *Application
}

type applicationBlogHandlers struct {
	app *Application
}

type applicationCourseServices struct {
	app *Application
}

type applicationCourseHandlers struct {
	app *Application
}

func (a *Application) Config() *config.Config {
	if a == nil {
		return nil
	}
	return a.cfg
}

func (a *Application) Cache() *cache.Cache {
	if a == nil {
		return nil
	}
	return a.cache
}

func (a *Application) Scheduler() *background.Scheduler {
	if a == nil {
		return nil
	}
	return a.scheduler
}

func (a *Application) ThemeManager() *theme.Manager {
	if a == nil {
		return nil
	}
	return a.themeManager
}

func (a *Application) Repositories() host.RepositoryAccess {
	return applicationRepositoryAccess{app: a}
}

func (a *Application) CoreServices() host.CoreServiceAccess {
	return applicationCoreServices{app: a}
}

func (a *Application) BlogServices() host.BlogServiceAccess {
	return applicationBlogServices{app: a}
}

func (a *Application) BlogHandlers() host.BlogHandlerAccess {
	return applicationBlogHandlers{app: a}
}

func (a *Application) CourseServices() host.CourseServiceAccess {
	return applicationCourseServices{app: a}
}

func (a *Application) CourseHandlers() host.CourseHandlerAccess {
	return applicationCourseHandlers{app: a}
}

func (a *Application) TemplateHandler() *handlers.TemplateHandler {
	if a == nil {
		return nil
	}
	return a.templateHandler
}

func (a *Application) SEOHandler() *handlers.SEOHandler {
	if a == nil {
		return nil
	}
	return a.handlers.SEO
}

func (a *Application) ThemeHandler() *handlers.ThemeHandler {
	if a == nil {
		return nil
	}
	return a.handlers.Theme
}

func (r applicationRepositoryAccess) Category() repository.CategoryRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Category
}

func (r applicationRepositoryAccess) Post() repository.PostRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Post
}

func (r applicationRepositoryAccess) Tag() repository.TagRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Tag
}

func (r applicationRepositoryAccess) Comment() repository.CommentRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Comment
}

func (r applicationRepositoryAccess) Search() repository.SearchRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Search
}

func (r applicationRepositoryAccess) Setting() repository.SettingRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.Setting
}

func (r applicationRepositoryAccess) User() repository.UserRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.User
}

func (r applicationRepositoryAccess) CourseVideo() repository.CourseVideoRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.CourseVideo
}

func (r applicationRepositoryAccess) CourseTopic() repository.CourseTopicRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.CourseTopic
}

func (r applicationRepositoryAccess) CoursePackage() repository.CoursePackageRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.CoursePackage
}

func (s applicationCoreServices) Auth() *service.AuthService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Auth
}

func (s applicationCoreServices) Setup() *service.SetupService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Setup
}

func (s applicationCoreServices) Theme() *service.ThemeService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Theme
}

func (s applicationCoreServices) SocialLink() *service.SocialLinkService {
	if s.app == nil {
		return nil
	}
	return s.app.services.SocialLink
}

func (s applicationCoreServices) Menu() *service.MenuService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Menu
}

func (s applicationCoreServices) Upload() *service.UploadService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Upload
}

func (s applicationCoreServices) Advertising() *service.AdvertisingService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Advertising
}

func (s applicationCoreServices) Language() *languageservice.LanguageService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Language
}

func (s applicationCoreServices) SetLanguage(language *languageservice.LanguageService) {
	if s.app == nil {
		return
	}
	s.app.services.Language = language
}

func (s applicationBlogServices) Category() *blogservice.CategoryService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Category
}

func (s applicationBlogServices) SetCategory(category *blogservice.CategoryService) {
	if s.app == nil {
		return
	}
	s.app.services.Category = category
}

func (s applicationBlogServices) Post() *blogservice.PostService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Post
}

func (s applicationBlogServices) SetPost(post *blogservice.PostService) {
	if s.app == nil {
		return
	}
	s.app.services.Post = post
}

func (s applicationBlogServices) Comment() *blogservice.CommentService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Comment
}

func (s applicationBlogServices) SetComment(comment *blogservice.CommentService) {
	if s.app == nil {
		return
	}
	s.app.services.Comment = comment
}

func (s applicationBlogServices) Search() *blogservice.SearchService {
	if s.app == nil {
		return nil
	}
	return s.app.services.Search
}

func (s applicationBlogServices) SetSearch(search *blogservice.SearchService) {
	if s.app == nil {
		return
	}
	s.app.services.Search = search
}

func (s applicationCourseServices) Video() *courseservice.VideoService {
	if s.app == nil {
		return nil
	}
	return s.app.services.CourseVideo
}

func (s applicationCourseServices) SetVideo(video *courseservice.VideoService) {
	if s.app == nil {
		return
	}
	s.app.services.CourseVideo = video
}

func (s applicationCourseServices) Topic() *courseservice.TopicService {
	if s.app == nil {
		return nil
	}
	return s.app.services.CourseTopic
}

func (s applicationCourseServices) SetTopic(topic *courseservice.TopicService) {
	if s.app == nil {
		return
	}
	s.app.services.CourseTopic = topic
}

func (s applicationCourseServices) Package() *courseservice.PackageService {
	if s.app == nil {
		return nil
	}
	return s.app.services.CoursePackage
}

func (s applicationCourseServices) SetPackage(pkg *courseservice.PackageService) {
	if s.app == nil {
		return
	}
	s.app.services.CoursePackage = pkg
}

func (s applicationCourseServices) Checkout() *courseservice.CheckoutService {
	if s.app == nil {
		return nil
	}
	return s.app.services.CourseCheckout
}

func (s applicationCourseServices) SetCheckout(checkout *courseservice.CheckoutService) {
	if s.app == nil {
		return
	}
	s.app.services.CourseCheckout = checkout
}

func (h applicationBlogHandlers) Post() *bloghandlers.PostHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.Post
}

func (h applicationBlogHandlers) SetPost(handler *bloghandlers.PostHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.Post = handler
}

func (h applicationBlogHandlers) Category() *bloghandlers.CategoryHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.Category
}

func (h applicationBlogHandlers) SetCategory(handler *bloghandlers.CategoryHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.Category = handler
}

func (h applicationBlogHandlers) Comment() *bloghandlers.CommentHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.Comment
}

func (h applicationBlogHandlers) SetComment(handler *bloghandlers.CommentHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.Comment = handler
}

func (h applicationBlogHandlers) Search() *bloghandlers.SearchHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.Search
}

func (h applicationBlogHandlers) SetSearch(handler *bloghandlers.SearchHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.Search = handler
}

func (h applicationCourseHandlers) Video() *coursehandlers.VideoHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.CourseVideo
}

func (h applicationCourseHandlers) SetVideo(handler *coursehandlers.VideoHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.CourseVideo = handler
}

func (h applicationCourseHandlers) Topic() *coursehandlers.TopicHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.CourseTopic
}

func (h applicationCourseHandlers) SetTopic(handler *coursehandlers.TopicHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.CourseTopic = handler
}

func (h applicationCourseHandlers) Package() *coursehandlers.PackageHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.CoursePackage
}

func (h applicationCourseHandlers) SetPackage(handler *coursehandlers.PackageHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.CoursePackage = handler
}

func (h applicationCourseHandlers) Checkout() *coursehandlers.CheckoutHandler {
	if h.app == nil {
		return nil
	}
	return h.app.handlers.CourseCheckout
}

func (h applicationCourseHandlers) SetCheckout(handler *coursehandlers.CheckoutHandler) {
	if h.app == nil {
		return
	}
	h.app.handlers.CourseCheckout = handler
}
