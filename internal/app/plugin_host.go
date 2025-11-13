package app

import (
	"strings"
	"sync"

	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	archiveapi "constructor-script-backend/plugins/archive/api"
	archivehandlers "constructor-script-backend/plugins/archive/handlers"
	archiveservice "constructor-script-backend/plugins/archive/service"
	blogapi "constructor-script-backend/plugins/blog/api"
	bloghandlers "constructor-script-backend/plugins/blog/handlers"
	blogservice "constructor-script-backend/plugins/blog/service"
	courseapi "constructor-script-backend/plugins/courses/api"
	coursehandlers "constructor-script-backend/plugins/courses/handlers"
	courseservice "constructor-script-backend/plugins/courses/service"
	forumapi "constructor-script-backend/plugins/forum/api"
	forumhandlers "constructor-script-backend/plugins/forum/handlers"
	forumservice "constructor-script-backend/plugins/forum/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

type applicationRepositoryAccess struct {
	app *Application
}

type applicationCoreServices struct {
	app *Application
}

type registryKind uint8

const (
	registryKindServices registryKind = iota
	registryKindHandlers
)

type applicationRegistry struct {
	app       *Application
	namespace string
	kind      registryKind
}

type pluginBindingContainer struct {
	mu      sync.RWMutex
	values  map[registryKind]map[string]map[string]any
	getters map[registryKind]map[string]map[string]func() any
	setters map[registryKind]map[string]map[string]func(any)
}

func (c *pluginBindingContainer) register(kind registryKind, namespace, key string, getter func() any, setter func(any)) {
	ns := normalize(namespace)
	k := normalize(key)

	c.mu.Lock()
	if getter != nil {
		if c.getters == nil {
			c.getters = make(map[registryKind]map[string]map[string]func() any)
		}
		if c.getters[kind] == nil {
			c.getters[kind] = make(map[string]map[string]func() any)
		}
		if c.getters[kind][ns] == nil {
			c.getters[kind][ns] = make(map[string]func() any)
		}
		c.getters[kind][ns][k] = getter
	}

	if setter != nil {
		if c.setters == nil {
			c.setters = make(map[registryKind]map[string]map[string]func(any))
		}
		if c.setters[kind] == nil {
			c.setters[kind] = make(map[string]map[string]func(any))
		}
		if c.setters[kind][ns] == nil {
			c.setters[kind][ns] = make(map[string]func(any))
		}
		c.setters[kind][ns][k] = setter
	}
	c.mu.Unlock()
}

func (c *pluginBindingContainer) get(kind registryKind, namespace, key string) any {
	ns := normalize(namespace)
	k := normalize(key)

	c.mu.RLock()
	if c.getters != nil {
		if nsGetters := c.getters[kind]; nsGetters != nil {
			if getter := nsGetters[ns][k]; getter != nil {
				c.mu.RUnlock()
				return getter()
			}
		}
	}

	var value any
	if c.values != nil {
		if nsValues := c.values[kind]; nsValues != nil {
			value = nsValues[ns][k]
		}
	}
	c.mu.RUnlock()
	return value
}

func (c *pluginBindingContainer) set(kind registryKind, namespace, key string, value any) {
	ns := normalize(namespace)
	k := normalize(key)

	var setter func(any)

	c.mu.Lock()
	if c.values == nil {
		c.values = make(map[registryKind]map[string]map[string]any)
	}
	if c.values[kind] == nil {
		c.values[kind] = make(map[string]map[string]any)
	}
	if c.values[kind][ns] == nil {
		c.values[kind][ns] = make(map[string]any)
	}
	if value == nil {
		delete(c.values[kind][ns], k)
	} else {
		c.values[kind][ns][k] = value
	}

	if c.setters != nil {
		if nsSetters := c.setters[kind]; nsSetters != nil {
			setter = nsSetters[ns][k]
		}
	}
	c.mu.Unlock()

	if setter != nil {
		setter(value)
	}
}

func (c *pluginBindingContainer) delete(kind registryKind, namespace, key string) {
	c.set(kind, namespace, key, nil)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (r applicationRegistry) Get(key string) any {
	if r.app == nil {
		return nil
	}
	return r.app.pluginBindings.get(r.kind, r.namespace, key)
}

func (r applicationRegistry) Set(key string, value any) {
	if r.app == nil {
		return
	}
	r.app.pluginBindings.set(r.kind, r.namespace, key, value)
}

func (r applicationRegistry) Delete(key string) {
	if r.app == nil {
		return
	}
	r.app.pluginBindings.delete(r.kind, r.namespace, key)
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

func (a *Application) Services(namespace string) host.Registry {
	return applicationRegistry{app: a, namespace: namespace, kind: registryKindServices}
}

func (a *Application) Handlers(namespace string) host.Registry {
	return applicationRegistry{app: a, namespace: namespace, kind: registryKindHandlers}
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

func (a *Application) AuthHandler() *handlers.AuthHandler {
	if a == nil {
		return nil
	}
	return a.handlers.Auth
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

func (r applicationRepositoryAccess) CoursePackageAccess() repository.CoursePackageAccessRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.CoursePackageAccess
}

func (r applicationRepositoryAccess) CourseTest() repository.CourseTestRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.CourseTest
}

func (r applicationRepositoryAccess) ForumCategory() repository.ForumCategoryRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ForumCategory
}

func (r applicationRepositoryAccess) ForumQuestion() repository.ForumQuestionRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ForumQuestion
}

func (r applicationRepositoryAccess) ForumAnswer() repository.ForumAnswerRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ForumAnswer
}

func (r applicationRepositoryAccess) ForumQuestionVote() repository.ForumQuestionVoteRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ForumQuestionVote
}

func (r applicationRepositoryAccess) ArchiveDirectory() repository.ArchiveDirectoryRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ArchiveDirectory
}

func (r applicationRepositoryAccess) ArchiveFile() repository.ArchiveFileRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ArchiveFile
}

func (r applicationRepositoryAccess) ForumAnswerVote() repository.ForumAnswerVoteRepository {
	if r.app == nil {
		return nil
	}
	return r.app.repositories.ForumAnswerVote
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

// registerPluginServiceBindings configures the service registry adapters for built-in plugins.
func (a *Application) registerPluginServiceBindings() {
	a.pluginBindings.register(
		registryKindServices,
		blogapi.Namespace,
		blogapi.ServiceCategory,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.Category
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.Category = nil
				return
			}
			if svc, ok := value.(*blogservice.CategoryService); ok {
				a.services.Category = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		blogapi.Namespace,
		blogapi.ServicePost,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.Post
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.Post = nil
				return
			}
			if svc, ok := value.(*blogservice.PostService); ok {
				a.services.Post = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		blogapi.Namespace,
		blogapi.ServiceComment,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.Comment
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.Comment = nil
				return
			}
			if svc, ok := value.(*blogservice.CommentService); ok {
				a.services.Comment = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		blogapi.Namespace,
		blogapi.ServiceSearch,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.Search
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.Search = nil
				return
			}
			if svc, ok := value.(*blogservice.SearchService); ok {
				a.services.Search = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		forumapi.Namespace,
		forumapi.ServiceQuestion,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.ForumQuestion
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.ForumQuestion = nil
				return
			}
			if svc, ok := value.(*forumservice.QuestionService); ok {
				a.services.ForumQuestion = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		forumapi.Namespace,
		forumapi.ServiceCategory,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.ForumCategory
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.ForumCategory = nil
				return
			}
			if svc, ok := value.(*forumservice.CategoryService); ok {
				a.services.ForumCategory = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		forumapi.Namespace,
		forumapi.ServiceAnswer,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.ForumAnswer
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.ForumAnswer = nil
				return
			}
			if svc, ok := value.(*forumservice.AnswerService); ok {
				a.services.ForumAnswer = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		courseapi.Namespace,
		courseapi.ServiceVideo,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.CourseVideo
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.CourseVideo = nil
				return
			}
			if svc, ok := value.(*courseservice.VideoService); ok {
				a.services.CourseVideo = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		courseapi.Namespace,
		courseapi.ServiceTopic,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.CourseTopic
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.CourseTopic = nil
				return
			}
			if svc, ok := value.(*courseservice.TopicService); ok {
				a.services.CourseTopic = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		courseapi.Namespace,
		courseapi.ServicePackage,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.CoursePackage
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.CoursePackage = nil
				return
			}
			if svc, ok := value.(*courseservice.PackageService); ok {
				a.services.CoursePackage = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		courseapi.Namespace,
		courseapi.ServiceTest,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.CourseTest
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.CourseTest = nil
				return
			}
			if svc, ok := value.(*courseservice.TestService); ok {
				a.services.CourseTest = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		courseapi.Namespace,
		courseapi.ServiceCheckout,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.CourseCheckout
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.CourseCheckout = nil
				return
			}
			if svc, ok := value.(*courseservice.CheckoutService); ok {
				a.services.CourseCheckout = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		archiveapi.Namespace,
		archiveapi.ServiceDirectory,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.ArchiveDirectory
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.ArchiveDirectory = nil
				return
			}
			if svc, ok := value.(*archiveservice.DirectoryService); ok {
				a.services.ArchiveDirectory = svc
			}
		},
	)

	a.pluginBindings.register(
		registryKindServices,
		archiveapi.Namespace,
		archiveapi.ServiceFile,
		func() any {
			if a == nil {
				return nil
			}
			return a.services.ArchiveFile
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.services.ArchiveFile = nil
				return
			}
			if svc, ok := value.(*archiveservice.FileService); ok {
				a.services.ArchiveFile = svc
			}
		},
	)
}

// registerPluginHandlerBindings configures handler registry adapters for built-in plugins.
func (a *Application) registerPluginHandlerBindings() {
	a.pluginBindings.register(
		registryKindHandlers,
		blogapi.Namespace,
		blogapi.HandlerPost,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.Post
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.Post = nil
				return
			}
			if handler, ok := value.(*bloghandlers.PostHandler); ok {
				a.handlers.Post = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		blogapi.Namespace,
		blogapi.HandlerCategory,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.Category
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.Category = nil
				return
			}
			if handler, ok := value.(*bloghandlers.CategoryHandler); ok {
				a.handlers.Category = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		blogapi.Namespace,
		blogapi.HandlerComment,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.Comment
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.Comment = nil
				return
			}
			if handler, ok := value.(*bloghandlers.CommentHandler); ok {
				a.handlers.Comment = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		blogapi.Namespace,
		blogapi.HandlerSearch,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.Search
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.Search = nil
				return
			}
			if handler, ok := value.(*bloghandlers.SearchHandler); ok {
				a.handlers.Search = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		forumapi.Namespace,
		forumapi.HandlerQuestion,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ForumQuestion
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ForumQuestion = nil
				return
			}
			if handler, ok := value.(*forumhandlers.QuestionHandler); ok {
				a.handlers.ForumQuestion = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		forumapi.Namespace,
		forumapi.HandlerCategory,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ForumCategory
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ForumCategory = nil
				return
			}
			if handler, ok := value.(*forumhandlers.CategoryHandler); ok {
				a.handlers.ForumCategory = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		forumapi.Namespace,
		forumapi.HandlerAnswer,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ForumAnswer
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ForumAnswer = nil
				return
			}
			if handler, ok := value.(*forumhandlers.AnswerHandler); ok {
				a.handlers.ForumAnswer = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		courseapi.Namespace,
		courseapi.HandlerVideo,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.CourseVideo
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.CourseVideo = nil
				return
			}
			if handler, ok := value.(*coursehandlers.VideoHandler); ok {
				a.handlers.CourseVideo = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		courseapi.Namespace,
		courseapi.HandlerTopic,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.CourseTopic
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.CourseTopic = nil
				return
			}
			if handler, ok := value.(*coursehandlers.TopicHandler); ok {
				a.handlers.CourseTopic = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		courseapi.Namespace,
		courseapi.HandlerTest,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.CourseTest
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.CourseTest = nil
				return
			}
			if handler, ok := value.(*coursehandlers.TestHandler); ok {
				a.handlers.CourseTest = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		courseapi.Namespace,
		courseapi.HandlerPackage,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.CoursePackage
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.CoursePackage = nil
				return
			}
			if handler, ok := value.(*coursehandlers.PackageHandler); ok {
				a.handlers.CoursePackage = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		courseapi.Namespace,
		courseapi.HandlerCheckout,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.CourseCheckout
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.CourseCheckout = nil
				return
			}
			if handler, ok := value.(*coursehandlers.CheckoutHandler); ok {
				a.handlers.CourseCheckout = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		archiveapi.Namespace,
		archiveapi.HandlerDirectory,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ArchiveDirectory
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ArchiveDirectory = nil
				return
			}
			if handler, ok := value.(*archivehandlers.DirectoryHandler); ok {
				a.handlers.ArchiveDirectory = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		archiveapi.Namespace,
		archiveapi.HandlerFile,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ArchiveFile
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ArchiveFile = nil
				return
			}
			if handler, ok := value.(*archivehandlers.FileHandler); ok {
				a.handlers.ArchiveFile = handler
			}
		},
	)

	a.pluginBindings.register(
		registryKindHandlers,
		archiveapi.Namespace,
		archiveapi.HandlerPublic,
		func() any {
			if a == nil {
				return nil
			}
			return a.handlers.ArchivePublic
		},
		func(value any) {
			if a == nil {
				return
			}
			if value == nil {
				a.handlers.ArchivePublic = nil
				return
			}
			if handler, ok := value.(*archivehandlers.PublicHandler); ok {
				a.handlers.ArchivePublic = handler
			}
		},
	)
}
