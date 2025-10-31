package blog

import (
	"fmt"

	"constructor-script-backend/internal/plugin/host"
	"constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
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
	services := f.host.BlogServices()

	categorySvc := services.Category()
	if categorySvc == nil {
		categorySvc = blogservice.NewCategoryService(repos.Category(), repos.Post(), f.host.Cache())
		services.SetCategory(categorySvc)
	}

	postSvc := services.Post()
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
		services.SetPost(postSvc)
	}

	commentSvc := services.Comment()
	if commentSvc == nil {
		commentSvc = blogservice.NewCommentService(repos.Comment())
		services.SetComment(commentSvc)
	}

	searchSvc := services.Search()
	if searchSvc == nil {
		searchSvc = blogservice.NewSearchService(repos.Search())
		services.SetSearch(searchSvc)
	}

	handlers := f.host.BlogHandlers()

	postHandler := handlers.Post()
	if postHandler == nil {
		postHandler = bloghandlers.NewPostHandler(postSvc)
		handlers.SetPost(postHandler)
	} else {
		postHandler.SetService(postSvc)
	}

	categoryHandler := handlers.Category()
	if categoryHandler == nil {
		categoryHandler = bloghandlers.NewCategoryHandler(categorySvc)
		handlers.SetCategory(categoryHandler)
	} else {
		categoryHandler.SetService(categorySvc)
	}

	commentHandler := handlers.Comment()
	guard := bloghandlers.NewCommentGuard(f.host.Config())
	if commentHandler == nil {
		commentHandler = bloghandlers.NewCommentHandler(commentSvc, f.host.CoreServices().Auth(), guard)
		handlers.SetComment(commentHandler)
	} else {
		commentHandler.SetService(commentSvc)
	}

	searchHandler := handlers.Search()
	if searchHandler == nil {
		searchHandler = bloghandlers.NewSearchHandler(searchSvc)
		handlers.SetSearch(searchHandler)
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

	handlers := f.host.BlogHandlers()
	if postHandler := handlers.Post(); postHandler != nil {
		postHandler.SetService(nil)
	}
	if categoryHandler := handlers.Category(); categoryHandler != nil {
		categoryHandler.SetService(nil)
	}
	if commentHandler := handlers.Comment(); commentHandler != nil {
		commentHandler.SetService(nil)
	}
	if searchHandler := handlers.Search(); searchHandler != nil {
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

	services := f.host.BlogServices()
	services.SetPost(nil)
	services.SetCategory(nil)
	services.SetComment(nil)
	services.SetSearch(nil)

	return nil
}
