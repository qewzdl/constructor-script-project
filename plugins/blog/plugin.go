package blog

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	blogapi "constructor-script-backend/plugins/blog/api"
	bloghandlers "constructor-script-backend/plugins/blog/handlers"
	blogseed "constructor-script-backend/plugins/blog/seed"
	blogservice "constructor-script-backend/plugins/blog/service"
)

func init() {
	registry.Register("blog", NewFeature)
}

type Feature struct {
	host host.Host
}

func NewFeature(h host.Host) (pluginruntime.Feature, error) {
	if h == nil {
		return nil, fmt.Errorf("host is required")
	}
	return &Feature{host: h}, nil
}

func (f *Feature) Activate() error {
	if f == nil || f.host == nil {
		return fmt.Errorf("feature host is not configured")
	}

	repos := f.host.Repositories()
	services := f.host.Services(blogapi.Namespace)

	var categorySvc *blogservice.CategoryService
	if value, ok := services.Get(blogapi.ServiceCategory).(*blogservice.CategoryService); ok {
		categorySvc = value
	}
	if categorySvc == nil {
		categorySvc = blogservice.NewCategoryService(repos.Category(), repos.Post(), f.host.Cache())
		services.Set(blogapi.ServiceCategory, categorySvc)
	}

	var postSvc *blogservice.PostService
	if value, ok := services.Get(blogapi.ServicePost).(*blogservice.PostService); ok {
		postSvc = value
	}
	if postSvc == nil {
		postSvc = blogservice.NewPostService(
			repos.Post(),
			repos.Tag(),
			repos.Category(),
			repos.Comment(),
			f.host.Cache(),
			repos.Setting(),
			f.host.Scheduler(),
			f.host.ThemeManager(),
		)
		services.Set(blogapi.ServicePost, postSvc)
	}

	var commentSvc *blogservice.CommentService
	if value, ok := services.Get(blogapi.ServiceComment).(*blogservice.CommentService); ok {
		commentSvc = value
	}
	if commentSvc == nil {
		commentSvc = blogservice.NewCommentService(repos.Comment())
		services.Set(blogapi.ServiceComment, commentSvc)
	}

	var searchSvc *blogservice.SearchService
	if value, ok := services.Get(blogapi.ServiceSearch).(*blogservice.SearchService); ok {
		searchSvc = value
	}
	if searchSvc == nil {
		searchSvc = blogservice.NewSearchService(repos.Search())
		services.Set(blogapi.ServiceSearch, searchSvc)
	}

	handlers := f.host.Handlers(blogapi.Namespace)

	var postHandler *bloghandlers.PostHandler
	if value, ok := handlers.Get(blogapi.HandlerPost).(*bloghandlers.PostHandler); ok {
		postHandler = value
	}
	if postHandler == nil {
		postHandler = bloghandlers.NewPostHandler(postSvc)
		handlers.Set(blogapi.HandlerPost, postHandler)
	} else {
		postHandler.SetService(postSvc)
	}

	var categoryHandler *bloghandlers.CategoryHandler
	if value, ok := handlers.Get(blogapi.HandlerCategory).(*bloghandlers.CategoryHandler); ok {
		categoryHandler = value
	}
	if categoryHandler == nil {
		categoryHandler = bloghandlers.NewCategoryHandler(categorySvc)
		handlers.Set(blogapi.HandlerCategory, categoryHandler)
	} else {
		categoryHandler.SetService(categorySvc)
	}

	var commentHandler *bloghandlers.CommentHandler
	if value, ok := handlers.Get(blogapi.HandlerComment).(*bloghandlers.CommentHandler); ok {
		commentHandler = value
	}
	guard := bloghandlers.NewCommentGuard(f.host.Config())
	if commentHandler == nil {
		commentHandler = bloghandlers.NewCommentHandler(commentSvc, f.host.CoreServices().Auth(), guard)
		handlers.Set(blogapi.HandlerComment, commentHandler)
	} else {
		commentHandler.SetService(commentSvc)
	}

	var searchHandler *bloghandlers.SearchHandler
	if value, ok := handlers.Get(blogapi.HandlerSearch).(*bloghandlers.SearchHandler); ok {
		searchHandler = value
	}
	if searchHandler == nil {
		searchHandler = bloghandlers.NewSearchHandler(searchSvc)
		handlers.Set(blogapi.HandlerSearch, searchHandler)
	} else {
		searchHandler.SetService(searchSvc)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetBlogServices(postSvc, categorySvc, commentSvc, searchSvc)
	}
	if seoHandler := f.host.SEOHandler(); seoHandler != nil {
		seoHandler.SetBlogServices(postSvc, categorySvc)
	}
	if themeHandler := f.host.ThemeHandler(); themeHandler != nil {
		themeHandler.SetPostService(postSvc)
	}

	if categorySvc != nil {
		blogseed.EnsureDefaultCategory(categorySvc)
	}

	themeManager := f.host.ThemeManager()
	if themeManager != nil && postSvc != nil {
		if active := themeManager.Active(); active != nil {
			blogseed.EnsureDefaultPosts(postSvc, repos.User(), active.PostsFS())
		}
	}

	return nil
}

func (f *Feature) Deactivate() error {
	if f == nil || f.host == nil {
		return nil
	}

	handlers := f.host.Handlers(blogapi.Namespace)
	if postHandler, _ := handlers.Get(blogapi.HandlerPost).(*bloghandlers.PostHandler); postHandler != nil {
		postHandler.SetService(nil)
	}
	if categoryHandler, _ := handlers.Get(blogapi.HandlerCategory).(*bloghandlers.CategoryHandler); categoryHandler != nil {
		categoryHandler.SetService(nil)
	}
	if commentHandler, _ := handlers.Get(blogapi.HandlerComment).(*bloghandlers.CommentHandler); commentHandler != nil {
		commentHandler.SetService(nil)
	}
	if searchHandler, _ := handlers.Get(blogapi.HandlerSearch).(*bloghandlers.SearchHandler); searchHandler != nil {
		searchHandler.SetService(nil)
	}

	if templateHandler := f.host.TemplateHandler(); templateHandler != nil {
		templateHandler.SetBlogServices(nil, nil, nil, nil)
	}
	if seoHandler := f.host.SEOHandler(); seoHandler != nil {
		seoHandler.SetBlogServices(nil, nil)
	}
	if themeHandler := f.host.ThemeHandler(); themeHandler != nil {
		themeHandler.SetPostService(nil)
	}

	services := f.host.Services(blogapi.Namespace)
	services.Set(blogapi.ServicePost, nil)
	services.Set(blogapi.ServiceCategory, nil)
	services.Set(blogapi.ServiceComment, nil)
	services.Set(blogapi.ServiceSearch, nil)

	return nil
}
