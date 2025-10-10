package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/middleware"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	"constructor-script-backend/pkg/validator"
)

func main() {
	// Initialize logger
	logger.Init()
	logger.Info("Starting Blog Backend API", nil)

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logger.Info("No .env file found, using environment variables", nil)
	}

	// Initialize configuration
	cfg := config.New()

	// Initialize validator
	validator.Init()

	// Connect to the database
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.NewGormLogger(),
	})
	if err != nil {
		logger.Error(err, "Failed to connect to database", nil)
		log.Fatal(err)
	}

	// Configure connection pool
	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Run database migrations
	logger.Info("Running database migrations", nil)
	if err := db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Post{},
		&models.Page{},
		&models.Tag{},
		&models.Comment{},
	); err != nil {
		logger.Error(err, "Failed to migrate database", nil)
		log.Fatal(err)
	}

	// Create database indexes
	logger.Info("Creating database indexes", nil)

	// Indexes for posts
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published) WHERE published = true")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug) WHERE published = true")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_template ON posts(template)")

	// Indexes for pages
	db.Exec("CREATE INDEX IF NOT EXISTS idx_pages_published ON pages(published) WHERE published = true")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_pages_slug ON pages(slug) WHERE published = true")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_pages_order ON pages(`order` ASC)")

	// GIN index for JSONB field 'sections' (PostgreSQL only)
	db.Exec("CREATE INDEX IF NOT EXISTS idx_posts_sections ON posts USING GIN (sections)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_pages_sections ON pages USING GIN (sections)")

	logger.Info("Database migration completed", nil)

	// Initialize Redis cache
	var cacheService *cache.Cache
	if cfg.EnableCache {
		cacheService = cache.NewCache(cfg.RedisURL, true)
	} else {
		cacheService = cache.NewCache("", false)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	postRepo := repository.NewPostRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	searchRepo := repository.NewSearchRepository(db)
	pageRepo := repository.NewPageRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	categoryService := service.NewCategoryService(categoryRepo, cacheService)
	postService := service.NewPostService(postRepo, tagRepo, cacheService)
	commentService := service.NewCommentService(commentRepo)
	searchService := service.NewSearchService(searchRepo)
	uploadService := service.NewUploadService(cfg.UploadDir)
	pageService := service.NewPageService(pageRepo, cacheService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	postHandler := handlers.NewPostHandler(postService)
	commentHandler := handlers.NewCommentHandler(commentService)
	searchHandler := handlers.NewSearchHandler(searchService)
	uploadHandler := handlers.NewUploadHandler(uploadService)
	pageHandler := handlers.NewPageHandler(pageService)

	// Configure Gin
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// ============================================
	// Load HTML Templates
	// ============================================
	templatesDir := "./templates"
	tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
	templates, err := tmpl.ParseGlob(filepath.Join(templatesDir, "*.html"))
	if err != nil {
		logger.Error(err, "Failed to load templates", nil)
		log.Fatal(err)
	}
	router.SetHTMLTemplate(templates)
	logger.Info("Templates loaded successfully", nil)

	// Initialize template handler (now it doesn't need to load templates)
	templateHandler, err := handlers.NewTemplateHandler(postService, pageService, templatesDir)
	if err != nil {
		logger.Error(err, "Failed to initialize template handler", nil)
		log.Fatal(err)
	}

	// Middleware
	router.Use(gin.Recovery())
	router.Use(logger.GinLogger())
	router.Use(middleware.MetricsMiddleware())
	router.Use(middleware.RateLimitMiddleware(cfg))

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	// Metrics endpoint for Prometheus
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Static files
	router.Static("/static", "./static")
	router.Static("/uploads", cfg.UploadDir)

	// ============================================
	// Frontend Routes (HTML rendering)
	// ============================================

	router.GET("/test-template", func(c *gin.Context) {
		logger.Info("Test template route called", nil)

		logger.Info("HTMLRender is OK", nil)

		c.HTML(200, "base.html", gin.H{
			"Title":   "Test",
			"Content": template.HTML("<h1>Test Content</h1>"),
		})
	})

	router.GET("/debug-templates", func(c *gin.Context) {
		tmpl := template.New("").Funcs(utils.GetTemplateFuncs())
		tmpl, _ = tmpl.ParseGlob("./templates/*.html")

		c.JSON(200, gin.H{
			"templates": tmpl.DefinedTemplates(),
		})
	})

	// Main page
	router.GET("/", templateHandler.RenderIndex)

	// Post by slug or ID (one route for both cases)
	router.GET("/blog/post/:slug", templateHandler.RenderPost)

	// Static pages
	router.GET("/page/:slug", templateHandler.RenderPage)

	// Blog with pagination
	router.GET("/blog", templateHandler.RenderBlog)

	// Posts by category
	router.GET("/category/:slug", templateHandler.RenderCategory)

	// Posts by tag
	router.GET("/tag/:slug", templateHandler.RenderTag)

	// ============================================
	// API Routes
	// ============================================

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Public API routes
		public := v1.Group("")
		{
			// Auth
			public.POST("/register", authHandler.Register)
			public.POST("/login", authHandler.Login)
			public.POST("/refresh", authHandler.RefreshToken)

			// Posts API
			public.GET("/posts", postHandler.GetAll)
			public.GET("/posts/:id", postHandler.GetByID)
			public.GET("/posts/slug/:slug", postHandler.GetBySlug)

			// Pages API
			public.GET("/pages", pageHandler.GetAll)
			public.GET("/pages/:id", pageHandler.GetByID)
			public.GET("/pages/slug/:slug", pageHandler.GetBySlug)

			// Categories
			public.GET("/categories", categoryHandler.GetAll)
			public.GET("/categories/:id", categoryHandler.GetByID)

			// Comments
			public.GET("/posts/:id/comments", commentHandler.GetByPostID)

			// Search
			public.GET("/search", searchHandler.Search)

			// Tags
			public.GET("/tags", postHandler.GetAllTags)
			public.GET("/tags/:slug/posts", postHandler.GetPostsByTag)
		}

		// Protected routes (authentication required)
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		{
			// Comments
			protected.POST("/posts/:id/comments", commentHandler.Create)
			protected.PUT("/comments/:id", commentHandler.Update)
			protected.DELETE("/comments/:id", commentHandler.Delete)

			// User profile
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)
			protected.PUT("/profile/password", authHandler.ChangePassword)
		}

		// Admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		admin.Use(middleware.AdminMiddleware())
		{
			// Posts
			admin.POST("/posts", postHandler.Create)
			admin.PUT("/posts/:id", postHandler.Update)
			admin.DELETE("/posts/:id", postHandler.Delete)
			admin.GET("/posts", postHandler.GetAllAdmin)
			admin.PUT("/posts/:id/publish", postHandler.PublishPost)
			admin.PUT("/posts/:id/unpublish", postHandler.UnpublishPost)

			// Pages
			admin.POST("/pages", pageHandler.Create)
			admin.PUT("/pages/:id", pageHandler.Update)
			admin.DELETE("/pages/:id", pageHandler.Delete)
			admin.GET("/pages", pageHandler.GetAllAdmin)
			admin.PUT("/pages/:id/publish", pageHandler.PublishPage)
			admin.PUT("/pages/:id/unpublish", pageHandler.UnpublishPage)

			// Upload
			admin.POST("/upload", uploadHandler.UploadImage)

			// Categories
			admin.POST("/categories", categoryHandler.Create)
			admin.PUT("/categories/:id", categoryHandler.Update)
			admin.DELETE("/categories/:id", categoryHandler.Delete)

			// Users
			admin.GET("/users", authHandler.GetAllUsers)
			admin.GET("/users/:id", authHandler.GetUser)
			admin.DELETE("/users/:id", authHandler.DeleteUser)
			admin.PUT("/users/:id/role", authHandler.UpdateUserRole)
			admin.PUT("/users/:id/status", authHandler.UpdateUserStatus)

			// Comments moderation
			admin.GET("/comments", commentHandler.GetAll)
			admin.PUT("/comments/:id/approve", commentHandler.ApproveComment)
			admin.PUT("/comments/:id/reject", commentHandler.RejectComment)

			// Statistics
			admin.GET("/stats", handlers.GetStatistics(db))

			// Cache management
			if cacheService != nil {
				admin.DELETE("/cache", handlers.ClearCache(cacheService))
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
				"Title":      "404 - Страница не найдена",
				"error":      "Страница не найдена",
				"StatusCode": 404,
			})
		}
	})

	srv := &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Run server in a goroutine
	go func() {
		logger.Info("Server starting", map[string]interface{}{
			"port":        cfg.Port,
			"environment": cfg.Environment,
		})

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(err, "Failed to start server", nil)
			log.Fatal(err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(err, "Server forced to shutdown", nil)
		log.Fatal(err)
	}

	// Close database connection
	sqlDB, _ = db.DB()
	sqlDB.Close()

	logger.Info("Server exited gracefully", nil)
}
