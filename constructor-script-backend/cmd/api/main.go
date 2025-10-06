package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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
		&models.Tag{},
		&models.Comment{},
	); err != nil {
		logger.Error(err, "Failed to migrate database", nil)
		log.Fatal(err)
	}

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

	// Initialize services
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	categoryService := service.NewCategoryService(categoryRepo, cacheService)
	postService := service.NewPostService(postRepo, tagRepo, cacheService)
	commentService := service.NewCommentService(commentRepo)
	searchService := service.NewSearchService(searchRepo)
	uploadService := service.NewUploadService(cfg.UploadDir)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	postHandler := handlers.NewPostHandler(postService)
	commentHandler := handlers.NewCommentHandler(commentService)
	searchHandler := handlers.NewSearchHandler(searchService)
	uploadHandler := handlers.NewUploadHandler(uploadService)

	// Configure Gin
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

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

	// Static files (uploaded images)
	router.Static("/uploads", cfg.UploadDir)

	// API v1
	v1 := router.Group("/api/v1")
	{
		// Public routes
		public := v1.Group("")
		{
			// Auth
			public.POST("/register", authHandler.Register)
			public.POST("/login", authHandler.Login)
			public.POST("/refresh", authHandler.RefreshToken)

			// Posts
			public.GET("/posts", postHandler.GetAll)
			public.GET("/posts/:id", postHandler.GetByID)
			public.GET("/posts/slug/:slug", postHandler.GetBySlug)

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

			// Posts management
			admin.GET("/posts", postHandler.GetAllAdmin)
			admin.PUT("/posts/:id/publish", postHandler.PublishPost)
			admin.PUT("/posts/:id/unpublish", postHandler.UnpublishPost)

			// Statistics
			admin.GET("/stats", handlers.GetStatistics(db))

			// Cache management
			if cacheService != nil {
				admin.DELETE("/cache", handlers.ClearCache(cacheService))
			}
		}
	}

	// Create HTTP server
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
