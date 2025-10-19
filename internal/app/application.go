package app

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/middleware"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/seed"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
)

type Options struct {
	TemplatesDir string
}

type Application struct {
	cfg     *config.Config
	options Options

	db    *gorm.DB
	cache *cache.Cache

	repositories repositoryContainer
	services     serviceContainer
	handlers     handlerContainer

	templateHandler *handlers.TemplateHandler
	router          *gin.Engine
	server          *http.Server
}

type repositoryContainer struct {
	User     repository.UserRepository
	Category repository.CategoryRepository
	Post     repository.PostRepository
	Tag      repository.TagRepository
	Comment  repository.CommentRepository
	Search   repository.SearchRepository
	Page     repository.PageRepository
	Setting  repository.SettingRepository
}

type serviceContainer struct {
	Auth     *service.AuthService
	Category *service.CategoryService
	Post     *service.PostService
	Comment  *service.CommentService
	Search   *service.SearchService
	Upload   *service.UploadService
	Page     *service.PageService
	Setup    *service.SetupService
}

type handlerContainer struct {
	Auth     *handlers.AuthHandler
	Category *handlers.CategoryHandler
	Post     *handlers.PostHandler
	Comment  *handlers.CommentHandler
	Search   *handlers.SearchHandler
	Upload   *handlers.UploadHandler
	Page     *handlers.PageHandler
	Setup    *handlers.SetupHandler
}

func New(cfg *config.Config, opts Options) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if opts.TemplatesDir == "" {
		opts.TemplatesDir = "./templates"
	}

	app := &Application{
		cfg:     cfg,
		options: opts,
	}

	if err := app.initDatabase(); err != nil {
		return nil, err
	}

	if err := app.runMigrations(); err != nil {
		return nil, err
	}

	if err := app.createIndexes(); err != nil {
		return nil, err
	}

	app.initCache()
	app.initRepositories()
	app.initServices()

	seed.EnsureDefaultCategory(app.services.Category)
	seed.EnsureDefaultPages(app.services.Page)

	if err := app.initHandlers(); err != nil {
		return nil, err
	}

	if err := app.initRouter(); err != nil {
		return nil, err
	}

	app.server = &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        app.router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	return app, nil
}

func (a *Application) Run() error {
	logger.Info("Server starting", map[string]interface{}{
		"port":        a.cfg.Port,
		"environment": a.cfg.Environment,
	})

	return a.server.ListenAndServe()
}

func (a *Application) Shutdown(ctx context.Context) error {
	if a.server != nil {
		if err := a.server.Shutdown(ctx); err != nil {
			return err
		}
	}

	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			logger.Error(err, "Failed to close cache connection", nil)
		}
	}

	if a.db != nil {
		if sqlDB, err := a.db.DB(); err == nil {
			sqlDB.Close()
		}
	}

	return nil
}

func (a *Application) Router() *gin.Engine {
	return a.router
}

func (a *Application) initDatabase() error {
	logger.Info("Connecting to database", nil)

	db, err := gorm.Open(postgres.Open(a.cfg.DatabaseURL), &gorm.Config{
		Logger: logger.NewGormLogger(),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	a.db = db
	return nil
}

func (a *Application) runMigrations() error {
	if a.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	logger.Info("Running database migrations", nil)

	if err := a.db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Post{},
		&models.Page{},
		&models.Tag{},
		&models.Comment{},
		&models.Setting{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Info("Database migration completed", nil)
	return nil
}

func (a *Application) createIndexes() error {
	if a.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	logger.Info("Creating database indexes", nil)

	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_posts_template ON posts(template)",
		"CREATE INDEX IF NOT EXISTS idx_pages_published ON pages(published) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_pages_slug ON pages(slug) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_pages_order ON pages(\"order\" ASC)",
		"CREATE INDEX IF NOT EXISTS idx_posts_sections ON posts USING GIN (sections)",
		"CREATE INDEX IF NOT EXISTS idx_pages_sections ON pages USING GIN (sections)",
	}

	for _, stmt := range statements {
		if err := a.db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (a *Application) initCache() {
	if a.cfg.EnableCache {
		a.cache = cache.NewCache(a.cfg.RedisURL, true)
	} else {
		a.cache = cache.NewCache("", false)
	}
}

func (a *Application) initRepositories() {
	a.repositories = repositoryContainer{
		User:     repository.NewUserRepository(a.db),
		Category: repository.NewCategoryRepository(a.db),
		Post:     repository.NewPostRepository(a.db),
		Tag:      repository.NewTagRepository(a.db),
		Comment:  repository.NewCommentRepository(a.db),
		Search:   repository.NewSearchRepository(a.db),
		Page:     repository.NewPageRepository(a.db),
		Setting:  repository.NewSettingRepository(a.db),
	}
}

func (a *Application) initServices() {
	a.services = serviceContainer{
		Auth:     service.NewAuthService(a.repositories.User, a.cfg.JWTSecret),
		Category: service.NewCategoryService(a.repositories.Category, a.repositories.Post, a.cache),
		Post:     service.NewPostService(a.repositories.Post, a.repositories.Tag, a.repositories.Category, a.cache, a.repositories.Setting),
		Comment:  service.NewCommentService(a.repositories.Comment),
		Search:   service.NewSearchService(a.repositories.Search),
		Upload:   service.NewUploadService(a.cfg.UploadDir),
		Page:     service.NewPageService(a.repositories.Page, a.cache),
		Setup:    service.NewSetupService(a.repositories.User, a.repositories.Setting),
	}
}

func (a *Application) initHandlers() error {
	a.handlers = handlerContainer{
		Auth:     handlers.NewAuthHandler(a.services.Auth),
		Category: handlers.NewCategoryHandler(a.services.Category),
		Post:     handlers.NewPostHandler(a.services.Post),
		Comment:  handlers.NewCommentHandler(a.services.Comment),
		Search:   handlers.NewSearchHandler(a.services.Search),
		Upload:   handlers.NewUploadHandler(a.services.Upload),
		Page:     handlers.NewPageHandler(a.services.Page),
		Setup:    handlers.NewSetupHandler(a.services.Setup, a.cfg),
	}

	templateHandler, err := handlers.NewTemplateHandler(
		a.services.Post,
		a.services.Page,
		a.services.Auth,
		a.services.Comment,
		a.services.Search,
		a.services.Setup,
		a.services.Category,
		a.cfg,
		a.options.TemplatesDir,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize template handler: %w", err)
	}

	a.templateHandler = templateHandler
	return nil
}

func (a *Application) initRouter() error {
	if a.cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger())
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RateLimitMiddleware(a.cfg))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     a.cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(middleware.SetupMiddleware(a.services.Setup))

	tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
	templates, err := tmpl.ParseGlob(filepath.Join(a.options.TemplatesDir, "*.html"))
	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}
	router.SetHTMLTemplate(templates)
	logger.Info("Templates loaded successfully", nil)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	router.Static("/static", "./static")
	router.Static("/uploads", a.cfg.UploadDir)
	router.StaticFile("/favicon.ico", "./favicon.ico")

	router.GET("/debug-templates", func(c *gin.Context) {
		tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
		tmpl, _ = tmpl.ParseGlob(filepath.Join(a.options.TemplatesDir, "*.html"))

		c.JSON(http.StatusOK, gin.H{
			"templates": tmpl.DefinedTemplates(),
		})
	})

	router.GET("/", a.templateHandler.RenderIndex)
	router.GET("/login", a.templateHandler.RenderLogin)
	router.GET("/register", a.templateHandler.RenderRegister)
	router.GET("/setup", a.templateHandler.RenderSetup)
	router.GET("/profile", a.templateHandler.RenderProfile)
	router.GET("/admin", a.templateHandler.RenderAdmin)
	router.GET("/blog/post/:slug", a.templateHandler.RenderPost)
	router.GET("/page/:slug", a.templateHandler.RenderPage)
	router.GET("/blog", a.templateHandler.RenderBlog)
	router.GET("/search", a.templateHandler.RenderSearch)
	router.GET("/category/:slug", a.templateHandler.RenderCategory)
	router.GET("/tag/:slug", a.templateHandler.RenderTag)

	v1 := router.Group("/api/v1")
	{
		public := v1.Group("")
		{
			public.GET("/setup/status", a.handlers.Setup.Status)
			public.POST("/setup", a.handlers.Setup.Complete)
			public.POST("/register", a.handlers.Auth.Register)
			public.POST("/login", a.handlers.Auth.Login)
			public.POST("/logout", a.handlers.Auth.Logout)
			public.POST("/refresh", a.handlers.Auth.RefreshToken)

			public.GET("/posts", a.handlers.Post.GetAll)
			public.GET("/posts/:id", a.handlers.Post.GetByID)
			public.GET("/posts/slug/:slug", a.handlers.Post.GetBySlug)

			public.GET("/pages", a.handlers.Page.GetAll)
			public.GET("/pages/:id", a.handlers.Page.GetByID)
			public.GET("/pages/slug/:slug", a.handlers.Page.GetBySlug)

			public.GET("/categories", a.handlers.Category.GetAll)
			public.GET("/categories/:id", a.handlers.Category.GetByID)

			public.GET("/posts/:id/comments", a.handlers.Comment.GetByPostID)

			public.GET("/search", a.handlers.Search.Search)

			public.GET("/tags", a.handlers.Post.GetAllTags)
			public.GET("/tags/:slug/posts", a.handlers.Post.GetPostsByTag)
		}

		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(a.cfg.JWTSecret))
		{
			protected.POST("/posts/:id/comments", a.handlers.Comment.Create)
			protected.PUT("/comments/:id", a.handlers.Comment.Update)
			protected.DELETE("/comments/:id", a.handlers.Comment.Delete)

			protected.GET("/profile", a.handlers.Auth.GetProfile)
			protected.PUT("/profile", a.handlers.Auth.UpdateProfile)
			protected.PUT("/profile/password", a.handlers.Auth.ChangePassword)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(a.cfg.JWTSecret))
		admin.Use(middleware.AdminMiddleware())
		{
			admin.POST("/posts", a.handlers.Post.Create)
			admin.PUT("/posts/:id", a.handlers.Post.Update)
			admin.DELETE("/posts/:id", a.handlers.Post.Delete)
			admin.GET("/posts", a.handlers.Post.GetAllAdmin)
			admin.PUT("/posts/:id/publish", a.handlers.Post.PublishPost)
			admin.PUT("/posts/:id/unpublish", a.handlers.Post.UnpublishPost)

			admin.POST("/pages", a.handlers.Page.Create)
			admin.PUT("/pages/:id", a.handlers.Page.Update)
			admin.DELETE("/pages/:id", a.handlers.Page.Delete)
			admin.GET("/pages", a.handlers.Page.GetAllAdmin)
			admin.PUT("/pages/:id/publish", a.handlers.Page.PublishPage)
			admin.PUT("/pages/:id/unpublish", a.handlers.Page.UnpublishPage)

			admin.POST("/upload", a.handlers.Upload.UploadImage)

			admin.POST("/categories", a.handlers.Category.Create)
			admin.PUT("/categories/:id", a.handlers.Category.Update)
			admin.DELETE("/categories/:id", a.handlers.Category.Delete)

			admin.GET("/users", a.handlers.Auth.GetAllUsers)
			admin.GET("/users/:id", a.handlers.Auth.GetUser)
			admin.DELETE("/users/:id", a.handlers.Auth.DeleteUser)
			admin.PUT("/users/:id/role", a.handlers.Auth.UpdateUserRole)
			admin.PUT("/users/:id/status", a.handlers.Auth.UpdateUserStatus)

			admin.GET("/comments", a.handlers.Comment.GetAll)
			admin.DELETE("/comments/:id", a.handlers.Comment.Delete)
			admin.PUT("/comments/:id/approve", a.handlers.Comment.ApproveComment)
			admin.PUT("/comments/:id/reject", a.handlers.Comment.RejectComment)

			admin.DELETE("/tags/:id", a.handlers.Post.DeleteTag)

			admin.GET("/settings/site", a.handlers.Setup.GetSiteSettings)
			admin.PUT("/settings/site", a.handlers.Setup.UpdateSiteSettings)

			admin.GET("/stats", handlers.GetStatistics(a.db))

			if a.cache != nil {
				admin.DELETE("/cache", handlers.ClearCache(a.cache))
			}
		}
	}

	router.NoRoute(func(c *gin.Context) {
		if len(c.Request.URL.Path) >= 4 && c.Request.URL.Path[:4] == "/api" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Route not found",
				"path":  c.Request.URL.Path,
			})
		} else {
			c.HTML(http.StatusNotFound, "error.html", gin.H{
				"Title":      "404 - Page not found",
				"error":      "The requested page could not be found",
				"StatusCode": 404,
			})
		}
	})

	a.router = router
	return nil
}
