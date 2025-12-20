package app

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"constructor-script-backend/internal/authorization"
	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/handlers"
	"constructor-script-backend/internal/middleware"
	"constructor-script-backend/internal/models"
	"constructor-script-backend/internal/plugin"
	_ "constructor-script-backend/internal/plugin/builtin"
	pluginregistry "constructor-script-backend/internal/plugin/registry"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
	"constructor-script-backend/internal/repository"
	"constructor-script-backend/internal/seed"
	"constructor-script-backend/internal/service"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/logger"
	"constructor-script-backend/pkg/utils"
	archivehandlers "constructor-script-backend/plugins/archive/handlers"
	archiveservice "constructor-script-backend/plugins/archive/service"
	bloghandlers "constructor-script-backend/plugins/blog/handlers"
	blogseed "constructor-script-backend/plugins/blog/seed"
	blogservice "constructor-script-backend/plugins/blog/service"
	coursehandlers "constructor-script-backend/plugins/courses/handlers"
	courseservice "constructor-script-backend/plugins/courses/service"
	forumhandlers "constructor-script-backend/plugins/forum/handlers"
	forumservice "constructor-script-backend/plugins/forum/service"
	languageservice "constructor-script-backend/plugins/language/service"
)

type Options struct {
	ThemesDir    string
	DefaultTheme string
	PluginsDir   string
}

type Application struct {
	cfg     *config.Config
	options Options

	db        *gorm.DB
	cache     *cache.Cache
	scheduler *background.Scheduler

	repositories   repositoryContainer
	services       serviceContainer
	handlers       handlerContainer
	pluginBindings pluginBindingContainer

	themeManager     *theme.Manager
	pluginManager    *plugin.Manager
	pluginRuntime    *pluginruntime.Runtime
	rateLimitManager *middleware.RateLimitManager
	templateHandler  *handlers.TemplateHandler
	router           *gin.Engine
	server           *http.Server
}

type repositoryContainer struct {
	User                repository.UserRepository
	Category            repository.CategoryRepository
	Post                repository.PostRepository
	Tag                 repository.TagRepository
	Comment             repository.CommentRepository
	Search              repository.SearchRepository
	Page                repository.PageRepository
	Setting             repository.SettingRepository
	SocialLink          repository.SocialLinkRepository
	Menu                repository.MenuRepository
	Plugin              repository.PluginRepository
	CourseVideo         repository.CourseVideoRepository
	CourseContent       repository.CourseContentRepository
	CourseTopic         repository.CourseTopicRepository
	CoursePackage       repository.CoursePackageRepository
	CoursePackageAccess repository.CoursePackageAccessRepository
	CourseTest          repository.CourseTestRepository
	ForumCategory       repository.ForumCategoryRepository
	ForumQuestion       repository.ForumQuestionRepository
	ForumAnswer         repository.ForumAnswerRepository
	ForumQuestionVote   repository.ForumQuestionVoteRepository
	ArchiveDirectory    repository.ArchiveDirectoryRepository
	ArchiveFile         repository.ArchiveFileRepository
	ForumAnswerVote     repository.ForumAnswerVoteRepository
}

type serviceContainer struct {
	Auth             *service.AuthService
	Category         *blogservice.CategoryService
	Post             *blogservice.PostService
	Comment          *blogservice.CommentService
	Search           *blogservice.SearchService
	Upload           *service.UploadService
	Backup           *service.BackupService
	Page             *service.PageService
	Setup            *service.SetupService
	Language         *languageservice.LanguageService
	Homepage         *service.HomepageService
	SocialLink       *service.SocialLinkService
	Menu             *service.MenuService
	Theme            *service.ThemeService
	Advertising      *service.AdvertisingService
	Plugin           *service.PluginService
	Font             *service.FontService
	CourseVideo      *courseservice.VideoService
	CourseContent    *courseservice.ContentService
	CourseTopic      *courseservice.TopicService
	CoursePackage    *courseservice.PackageService
	CourseTest       *courseservice.TestService
	CourseCheckout   *courseservice.CheckoutService
	ForumCategory    *forumservice.CategoryService
	ForumQuestion    *forumservice.QuestionService
	ArchiveDirectory *archiveservice.DirectoryService
	ArchiveFile      *archiveservice.FileService
	ForumAnswer      *forumservice.AnswerService
}

type handlerContainer struct {
	Auth             *handlers.AuthHandler
	Category         *bloghandlers.CategoryHandler
	Post             *bloghandlers.PostHandler
	Comment          *bloghandlers.CommentHandler
	Search           *bloghandlers.SearchHandler
	Upload           *handlers.UploadHandler
	Backup           *handlers.BackupHandler
	Page             *handlers.PageHandler
	PageBuilder      *handlers.PageBuilderHandler
	Setup            *handlers.SetupHandler
	Homepage         *handlers.HomepageHandler
	SocialLink       *handlers.SocialLinkHandler
	Menu             *handlers.MenuHandler
	SEO              *handlers.SEOHandler
	Theme            *handlers.ThemeHandler
	Advertising      *handlers.AdvertisingHandler
	Plugin           *handlers.PluginHandler
	Font             *handlers.FontHandler
	CourseVideo      *coursehandlers.VideoHandler
	CourseContent    *coursehandlers.ContentHandler
	CourseTopic      *coursehandlers.TopicHandler
	CourseTest       *coursehandlers.TestHandler
	CoursePackage    *coursehandlers.PackageHandler
	CourseCheckout   *coursehandlers.CheckoutHandler
	ForumCategory    *forumhandlers.CategoryHandler
	ForumQuestion    *forumhandlers.QuestionHandler
	ArchiveDirectory *archivehandlers.DirectoryHandler
	ArchiveFile      *archivehandlers.FileHandler
	ArchivePublic    *archivehandlers.PublicHandler
	ForumAnswer      *forumhandlers.AnswerHandler
}

func New(cfg *config.Config, opts Options) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if opts.ThemesDir == "" {
		opts.ThemesDir = "./themes"
	}

	if opts.DefaultTheme == "" {
		opts.DefaultTheme = "default"
	}

	if opts.PluginsDir == "" {
		opts.PluginsDir = "./plugins"
	}

	if cfg.JWTSecretAutoGenerated {
		logger.Warn("JWT secret was automatically generated; set JWT_SECRET to a strong, persistent value", map[string]interface{}{
			"cause": cfg.JWTSecretAutoGeneratedCause,
		})
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

	// Initialize rate limit manager with application context
	app.rateLimitManager = middleware.NewRateLimitManager(context.Background())

	app.scheduler = background.NewScheduler(background.SchedulerConfig{})
	app.scheduler.Start(context.Background())

	cleanupNeeded := true
	defer func() {
		if !cleanupNeeded {
			return
		}
		if app.rateLimitManager != nil {
			if err := app.rateLimitManager.Shutdown(); err != nil {
				logger.Error(err, "Failed to shutdown rate limit manager during initialization", nil)
			}
		}
		if app.scheduler != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := app.scheduler.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error(err, "Failed to stop background scheduler during initialization", nil)
			}
		}
	}()

	if err := app.initThemeManager(); err != nil {
		return nil, err
	}

	if err := app.initPluginManager(); err != nil {
		return nil, err
	}

	app.pluginRuntime = pluginruntime.New()

	app.initServices()

	if theme := app.themeManager.Active(); theme != nil {
		applyDefaults := true
		if app.services.Theme != nil {
			if needsInitialization, err := app.services.Theme.RequiresInitialization(theme.Slug); err != nil {
				logger.Error(err, "Failed to determine if theme defaults should be applied", map[string]interface{}{"theme": theme.Slug})
				applyDefaults = false
			} else {
				applyDefaults = needsInitialization
			}
		}

		if applyDefaults {
			seed.EnsureDefaultPages(app.services.Page, theme.PagesFS())
			seed.EnsureDefaultMenu(app.services.Menu, theme.MenuFS())
			blogseed.EnsureDefaultPosts(app.services.Post, app.repositories.User, theme.PostsFS())
			if app.services.Theme != nil {
				if err := app.services.Theme.MarkInitialized(theme.Slug); err != nil {
					logger.Error(err, "Failed to mark theme defaults as applied", map[string]interface{}{"theme": theme.Slug})
				}
			}
		}
	}

	if err := app.initHandlers(); err != nil {
		return nil, err
	}

	if err := app.initPluginRuntime(); err != nil {
		return nil, err
	}

	if err := app.initRouter(); err != nil {
		return nil, err
	}

	// Configure server timeouts based on config
	// Default values allow for large file uploads (up to 2GB)
	readTimeout := time.Duration(cfg.ServerReadTimeout) * time.Second
	writeTimeout := time.Duration(cfg.ServerWriteTimeout) * time.Second
	idleTimeout := time.Duration(cfg.ServerIdleTimeout) * time.Second

	// Fallback to reasonable defaults if config values are invalid
	if readTimeout <= 0 {
		readTimeout = 5 * time.Minute
	}
	if writeTimeout <= 0 {
		writeTimeout = 5 * time.Minute
	}
	if idleTimeout <= 0 {
		idleTimeout = 2 * time.Minute
	}

	app.server = &http.Server{
		Addr:           ":" + cfg.Port,
		Handler:        app.router,
		ReadTimeout:    readTimeout,
		WriteTimeout:   writeTimeout,
		IdleTimeout:    idleTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB for headers
	}

	cleanupNeeded = false
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

	if a.rateLimitManager != nil {
		if err := a.rateLimitManager.Shutdown(); err != nil {
			logger.Error(err, "Failed to shutdown rate limit manager", nil)
		}
	}

	if a.scheduler != nil {
		if err := a.scheduler.Shutdown(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.Error(err, "Failed to shut down background scheduler", nil)
		}
	}

	if a.services.Backup != nil {
		a.services.Backup.ShutdownAutoBackups()
	}

	// Clean up plugin runtime to free memory
	if a.pluginRuntime != nil {
		if err := a.pluginRuntime.Clear(); err != nil {
			logger.Error(err, "Failed to clear plugin runtime", nil)
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

	migrator := a.db.Migrator()

	if migrator.HasTable(&models.ForumCategory{}) {
		indexes := []string{"idx_forum_categories_name", "idx_forum_categories_slug"}
		for _, indexName := range indexes {
			if migrator.HasIndex(&models.ForumCategory{}, indexName) {
				if err := migrator.DropIndex(&models.ForumCategory{}, indexName); err != nil {
					logger.Warn("Failed to drop legacy forum category index", map[string]interface{}{
						"index": indexName,
						"error": err.Error(),
					})
				}
			}
		}
	}

	// Fix comment foreign key constraints to cascade delete
	if migrator.HasTable(&models.Comment{}) {
		// Drop old constraints without cascade
		if err := a.db.Migrator().DropConstraint(&models.Comment{}, "fk_users_comments"); err == nil {
			logger.Info("Dropped old foreign key constraint fk_users_comments", nil)
		}
		if err := a.db.Migrator().DropConstraint(&models.Comment{}, "fk_posts_comments"); err == nil {
			logger.Info("Dropped old foreign key constraint fk_posts_comments", nil)
		}
		if err := a.db.Migrator().DropConstraint(&models.Comment{}, "fk_comments_parent_id"); err == nil {
			logger.Info("Dropped old foreign key constraint fk_comments_parent_id", nil)
		}
	}

	if migrator.HasTable(&models.CourseTopicStep{}) {
		if err := a.db.Exec("ALTER TABLE course_topic_steps ADD COLUMN IF NOT EXISTS test_id bigint").Error; err != nil {
			return fmt.Errorf("failed to ensure course topic step test reference: %w", err)
		}
		if err := a.db.Exec("ALTER TABLE course_topic_steps ADD COLUMN IF NOT EXISTS content_id bigint").Error; err != nil {
			return fmt.Errorf("failed to ensure course topic step content reference: %w", err)
		}
	}

	if err := a.ensureCourseSlugs(migrator); err != nil {
		return err
	}

	if migrator.HasTable(&models.Page{}) && !migrator.HasColumn(&models.Page{}, "path") {
		if err := a.db.Exec("ALTER TABLE pages ADD COLUMN path text").Error; err != nil {
			return fmt.Errorf("failed to add page path column: %w", err)
		}
	}

	if err := a.db.AutoMigrate(
		&models.User{},
		&models.Category{},
		&models.Post{},
		&models.PostViewStat{},
		&models.Page{},
		&models.ArchiveDirectory{},
		&models.ArchiveFile{},
		&models.Tag{},
		&models.Comment{},
		&models.ForumCategory{},
		&models.ForumQuestion{},
		&models.ForumAnswer{},
		&models.ForumQuestionVote{},
		&models.ForumAnswerVote{},
		&models.CourseVideo{},
		&models.CourseTopic{},
		&models.CourseContent{},
		&models.CoursePackage{},
		&models.CourseTopicVideo{},
		&models.CoursePackageTopic{},
		&models.CoursePackageAccess{},
		&models.CourseTest{},
		&models.CourseTestQuestion{},
		&models.CourseTestQuestionOption{},
		&models.CourseTopicStep{},
		&models.CourseTestResult{},
		&models.Setting{},
		&models.SocialLink{},
		&models.MenuItem{},
		&models.Plugin{},
		&models.SetupProgress{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	if migrator.HasTable(&models.CourseTopicStep{}) {
		if err := a.db.Exec("CREATE INDEX IF NOT EXISTS idx_course_topic_steps_test_id ON course_topic_steps(test_id)").Error; err != nil {
			return fmt.Errorf("failed to ensure course topic step test index: %w", err)
		}
		if err := a.db.Exec("CREATE INDEX IF NOT EXISTS idx_course_topic_steps_content_id ON course_topic_steps(content_id)").Error; err != nil {
			return fmt.Errorf("failed to ensure course topic step content index: %w", err)
		}
	}

	if err := a.migrateTopicVideosToSteps(); err != nil {
		return err
	}

	if err := a.db.Exec(`
                UPDATE posts
                SET publish_at = COALESCE(publish_at, created_at)
                WHERE publish_at IS NULL AND published = TRUE
        `).Error; err != nil {
		return fmt.Errorf("failed to backfill post publish_at: %w", err)
	}

	if err := a.db.Exec(`
                UPDATE posts
                SET published_at = COALESCE(published_at, publish_at, created_at)
                WHERE published = TRUE
        `).Error; err != nil {
		return fmt.Errorf("failed to backfill post published_at: %w", err)
	}

	if err := a.db.Exec(`
                UPDATE pages
                SET publish_at = COALESCE(publish_at, created_at)
                WHERE publish_at IS NULL AND published = TRUE
        `).Error; err != nil {
		return fmt.Errorf("failed to backfill page publish_at: %w", err)
	}

	if err := a.db.Exec(`
                UPDATE pages
                SET published_at = COALESCE(published_at, publish_at, created_at)
                WHERE published = TRUE
        `).Error; err != nil {
		return fmt.Errorf("failed to backfill page published_at: %w", err)
	}

	if migrator.HasTable(&models.Page{}) {
		if err := a.db.Exec(`
                        UPDATE pages
                        SET path = CASE
                                WHEN slug = 'home' THEN '/'
                                ELSE '/' || slug
                        END
                        WHERE (path IS NULL OR path = '') AND slug <> ''
                `).Error; err != nil {
			return fmt.Errorf("failed to backfill page paths: %w", err)
		}

		if err := a.db.Exec(`
                        UPDATE pages
                        SET path = '/'
                        WHERE (path IS NULL OR path = '') AND slug = ''
                `).Error; err != nil {
			return fmt.Errorf("failed to normalize empty page paths: %w", err)
		}

		if err := a.db.Exec("ALTER TABLE pages ALTER COLUMN path SET NOT NULL").Error; err != nil {
			return fmt.Errorf("failed to enforce page path requirement: %w", err)
		}

		if err := a.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_pages_path ON pages(path)").Error; err != nil {
			return fmt.Errorf("failed to ensure page path uniqueness: %w", err)
		}
	}

	logger.Info("Database migration completed", nil)
	return nil
}

func (a *Application) ensureCourseSlugs(migrator gorm.Migrator) error {
	if err := a.ensureCourseTopicSlugs(migrator); err != nil {
		return err
	}

	if err := a.ensureCoursePackageSlugs(migrator); err != nil {
		return err
	}

	return nil
}

func (a *Application) ensureCourseTopicSlugs(migrator gorm.Migrator) error {
	if !migrator.HasTable(&models.CourseTopic{}) {
		return nil
	}

	if !migrator.HasColumn(&models.CourseTopic{}, "slug") {
		if err := a.db.Exec("ALTER TABLE course_topics ADD COLUMN slug text").Error; err != nil {
			return fmt.Errorf("failed to add course topic slug column: %w", err)
		}
	}

	if err := a.ensureSlugsForTable("course_topics", "topic", "idx_course_topics_slug"); err != nil {
		return fmt.Errorf("failed to ensure course topic slugs: %w", err)
	}

	return nil
}

func (a *Application) ensureCoursePackageSlugs(migrator gorm.Migrator) error {
	if !migrator.HasTable(&models.CoursePackage{}) {
		return nil
	}

	if !migrator.HasColumn(&models.CoursePackage{}, "slug") {
		if err := a.db.Exec("ALTER TABLE course_packages ADD COLUMN slug text").Error; err != nil {
			return fmt.Errorf("failed to add course package slug column: %w", err)
		}
	}

	if err := a.ensureSlugsForTable("course_packages", "package", "idx_course_packages_slug"); err != nil {
		return fmt.Errorf("failed to ensure course package slugs: %w", err)
	}

	return nil
}

func (a *Application) ensureSlugsForTable(tableName, fallbackPrefix, indexName string) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		type slugRow struct {
			ID        uint
			Title     string
			Slug      sql.NullString
			DeletedAt gorm.DeletedAt
		}

		var rows []slugRow
		if err := tx.Table(tableName).Select("id, title, slug, deleted_at").Order("id ASC").Find(&rows).Error; err != nil {
			return fmt.Errorf("failed to load %s rows for slug backfill: %w", tableName, err)
		}

		existing := make(map[string]struct{}, len(rows))

		ensureSlug := func(row slugRow, enforceUnique bool) error {
			base := strings.TrimSpace(strings.ToLower(row.Slug.String))
			if base == "" {
				generated := utils.GenerateSlug(row.Title)
				base = strings.TrimSpace(strings.ToLower(generated))
			}

			if base == "" {
				base = fmt.Sprintf("%s-%d", fallbackPrefix, row.ID)
			}

			candidate := base
			if enforceUnique {
				suffix := 0
				for {
					var attempt string
					switch suffix {
					case 0:
						attempt = base
					case 1:
						attempt = fmt.Sprintf("%s-%d", base, row.ID)
					default:
						attempt = fmt.Sprintf("%s-%d-%d", base, row.ID, suffix)
					}

					if _, exists := existing[attempt]; !exists {
						candidate = attempt
						break
					}

					suffix++
				}
			}

			current := strings.TrimSpace(row.Slug.String)
			currentLower := strings.TrimSpace(strings.ToLower(row.Slug.String))
			if !row.Slug.Valid || currentLower != candidate || current != candidate || row.Slug.String != current {
				if err := tx.Exec(fmt.Sprintf("UPDATE %s SET slug = ? WHERE id = ?", tableName), candidate, row.ID).Error; err != nil {
					return fmt.Errorf("failed to update %s slug for id %d: %w", tableName, row.ID, err)
				}
				current = candidate
			}

			if enforceUnique {
				existing[current] = struct{}{}
			}

			return nil
		}

		for _, row := range rows {
			if row.DeletedAt.Valid {
				continue
			}
			if err := ensureSlug(row, true); err != nil {
				return err
			}
		}

		for _, row := range rows {
			if !row.DeletedAt.Valid {
				continue
			}
			if err := ensureSlug(row, false); err != nil {
				return err
			}
		}

		if err := tx.Exec(fmt.Sprintf("ALTER TABLE %s ALTER COLUMN slug SET NOT NULL", tableName)).Error; err != nil {
			return fmt.Errorf("failed to enforce NOT NULL on %s slug column: %w", tableName, err)
		}

		if err := tx.Exec(fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)).Error; err != nil {
			return fmt.Errorf("failed to drop legacy %s slug index: %w", tableName, err)
		}

		createStmt := fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (slug) WHERE deleted_at IS NULL", indexName, tableName)
		if err := tx.Exec(createStmt).Error; err != nil {
			return fmt.Errorf("failed to ensure unique index for %s slug column: %w", tableName, err)
		}

		return nil
	})
}

func (a *Application) createIndexes() error {
	if a.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	logger.Info("Creating database indexes", nil)

	statements := []string{
		"CREATE INDEX IF NOT EXISTS idx_posts_published ON posts(published) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC)",
		"CREATE INDEX IF NOT EXISTS idx_posts_publish_at ON posts(publish_at)",
		"CREATE INDEX IF NOT EXISTS idx_posts_slug ON posts(slug) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_posts_template ON posts(template)",
		"CREATE INDEX IF NOT EXISTS idx_pages_published ON pages(published) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_pages_slug ON pages(slug) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_pages_path ON pages(path) WHERE published = true",
		"CREATE INDEX IF NOT EXISTS idx_pages_publish_at ON pages(publish_at)",
		"CREATE INDEX IF NOT EXISTS idx_pages_order ON pages(\"order\" ASC)",
		"CREATE INDEX IF NOT EXISTS idx_posts_sections ON posts USING GIN (sections)",
		"CREATE INDEX IF NOT EXISTS idx_pages_sections ON pages USING GIN (sections)",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_post_view_stats_post_date ON post_view_stats(post_id, date)",
	}

	for _, stmt := range statements {
		if err := a.db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (a *Application) migrateTopicVideosToSteps() error {
	if a.db == nil {
		return fmt.Errorf("database connection is not initialized")
	}

	migrator := a.db.Migrator()
	if !migrator.HasTable(&models.CourseTopicVideo{}) {
		return nil
	}
	if !migrator.HasTable(&models.CourseTopicStep{}) {
		return nil
	}

	var stepCount int64
	if err := a.db.Model(&models.CourseTopicStep{}).Count(&stepCount).Error; err != nil {
		return fmt.Errorf("failed to count course topic steps: %w", err)
	}
	if stepCount > 0 {
		return nil
	}

	var legacyCount int64
	if err := a.db.Model(&models.CourseTopicVideo{}).Count(&legacyCount).Error; err != nil {
		return fmt.Errorf("failed to count legacy topic videos: %w", err)
	}
	if legacyCount == 0 {
		return nil
	}

	var links []models.CourseTopicVideo
	if err := a.db.Order("topic_id ASC, position ASC").Find(&links).Error; err != nil {
		return fmt.Errorf("failed to load legacy topic videos: %w", err)
	}
	if len(links) == 0 {
		return nil
	}

	if err := a.db.Transaction(func(tx *gorm.DB) error {
		for _, link := range links {
			videoID := link.VideoID
			step := models.CourseTopicStep{
				CreatedAt: link.CreatedAt,
				UpdatedAt: link.UpdatedAt,
				TopicID:   link.TopicID,
				StepType:  models.CourseTopicStepTypeVideo,
				Position:  link.Position,
				VideoID:   &videoID,
			}
			if err := tx.Create(&step).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to migrate topic videos to steps: %w", err)
	}

	return nil
}

func (a *Application) initCache() {
	if !a.cfg.EnableCache || !a.cfg.EnableRedis {
		disabledCache, err := cache.NewCache("", false)
		if err != nil {
			logger.Error(err, "Failed to initialize disabled cache", nil)
			return
		}
		a.cache = disabledCache
		return
	}

	cacheInstance, err := cache.NewCache(a.cfg.RedisURL, true)
	if err != nil {
		logger.Error(err, "Failed to initialize Redis cache, caching disabled", map[string]interface{}{"redis_url": a.cfg.RedisURL})
		fallbackCache, fallbackErr := cache.NewCache("", false)
		if fallbackErr != nil {
			logger.Error(fallbackErr, "Failed to initialize fallback cache", nil)
			return
		}
		a.cache = fallbackCache
		return
	}

	a.cache = cacheInstance
}

func (a *Application) initRepositories() {
	a.repositories = repositoryContainer{
		User:                repository.NewUserRepository(a.db),
		Category:            repository.NewCategoryRepository(a.db),
		Post:                repository.NewPostRepository(a.db),
		Tag:                 repository.NewTagRepository(a.db),
		Comment:             repository.NewCommentRepository(a.db),
		Search:              repository.NewSearchRepository(a.db),
		Page:                repository.NewPageRepository(a.db),
		Setting:             repository.NewSettingRepository(a.db),
		SocialLink:          repository.NewSocialLinkRepository(a.db),
		Menu:                repository.NewMenuRepository(a.db),
		Plugin:              repository.NewPluginRepository(a.db),
		CourseVideo:         repository.NewCourseVideoRepository(a.db),
		CourseContent:       repository.NewCourseContentRepository(a.db),
		CourseTopic:         repository.NewCourseTopicRepository(a.db),
		CoursePackage:       repository.NewCoursePackageRepository(a.db),
		CoursePackageAccess: repository.NewCoursePackageAccessRepository(a.db),
		CourseTest:          repository.NewCourseTestRepository(a.db),
		ForumCategory:       repository.NewForumCategoryRepository(a.db),
		ForumQuestion:       repository.NewForumQuestionRepository(a.db),
		ArchiveDirectory:    repository.NewArchiveDirectoryRepository(a.db),
		ArchiveFile:         repository.NewArchiveFileRepository(a.db),
		ForumAnswer:         repository.NewForumAnswerRepository(a.db),
		ForumQuestionVote:   repository.NewForumQuestionVoteRepository(a.db),
		ForumAnswerVote:     repository.NewForumAnswerVoteRepository(a.db),
	}
}

func (a *Application) initThemeManager() error {
	manager, err := theme.NewManager(a.options.ThemesDir)
	if err != nil {
		return fmt.Errorf("failed to load themes: %w", err)
	}

	activeSlug := strings.ToLower(strings.TrimSpace(a.options.DefaultTheme))
	storedSlug := ""

	if a.repositories.Setting != nil {
		setting, err := a.repositories.Setting.Get(service.SettingKeyActiveTheme)
		if err == nil && setting != nil {
			storedSlug = strings.ToLower(strings.TrimSpace(setting.Value))
			if storedSlug != "" {
				activeSlug = storedSlug
			}
		} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Error(err, "Failed to read active theme setting", nil)
		}
	}

	finalSlug := activeSlug

	if err := manager.Activate(activeSlug); err != nil {
		logger.Error(err, "Failed to activate stored theme, attempting default", map[string]interface{}{"theme": activeSlug})
		if fallbackErr := manager.Activate(a.options.DefaultTheme); fallbackErr != nil {
			return fmt.Errorf("failed to activate default theme: %w", fallbackErr)
		}
		finalSlug = a.options.DefaultTheme
	}

	if a.repositories.Setting != nil {
		if storedSlug == "" || storedSlug != finalSlug {
			if err := a.repositories.Setting.Set(service.SettingKeyActiveTheme, finalSlug); err != nil {
				logger.Error(err, "Failed to persist active theme setting", map[string]interface{}{"theme": finalSlug})
			}
		}
	}

	a.themeManager = manager
	return nil
}

func (a *Application) initPluginManager() error {
	manager, err := plugin.NewManager(a.options.PluginsDir)
	if err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	a.pluginManager = manager
	return nil
}

func (a *Application) initServices() {
	uploadService := service.NewUploadService(a.cfg.UploadDir)
	var languageService *languageservice.LanguageService
	setupService := service.NewSetupService(a.repositories.User, a.repositories.Setting, uploadService, languageService)

	// Set database connection for setup service to enable progress tracking
	if setupService != nil && a.db != nil {
		setupService.SetDB(a.db)
	}

	subtitleDefaults := models.SubtitleSettings{}
	if a.cfg != nil {
		subtitleDefaults.Enabled = a.cfg.SubtitleGenerationEnabled
		subtitleDefaults.Provider = strings.TrimSpace(a.cfg.SubtitleProvider)
		subtitleDefaults.PreferredName = strings.TrimSpace(a.cfg.SubtitlePreferredName)
		subtitleDefaults.Language = strings.TrimSpace(a.cfg.SubtitleLanguage)
		subtitleDefaults.Prompt = strings.TrimSpace(a.cfg.SubtitlePrompt)
		subtitleDefaults.OpenAIModel = strings.TrimSpace(a.cfg.OpenAIModel)
		subtitleDefaults.OpenAIAPIKey = strings.TrimSpace(a.cfg.OpenAIAPIKey)
		if a.cfg.SubtitleTemperature != nil {
			value := *a.cfg.SubtitleTemperature
			subtitleDefaults.Temperature = &value
		}
	}

	subtitleSettings := subtitleDefaults
	if setupService != nil {
		if resolved, err := setupService.GetSubtitleSettings(subtitleDefaults); err != nil {
			logger.Error(err, "Failed to load subtitle settings", nil)
		} else {
			subtitleSettings = resolved
		}
	}

	service.ConfigureUploadSubtitles(uploadService, subtitleSettings)

	backupOptions := service.BackupOptions{UploadDir: a.cfg.UploadDir}

	if key := strings.TrimSpace(a.cfg.BackupEncryptionKey); key != "" {
		decoded, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			logger.Error(err, "Invalid backup encryption key format", nil)
		} else if len(decoded) < 32 {
			logger.Warn("Backup encryption key is too short; encryption disabled", nil)
		} else {
			backupOptions.EncryptionKey = decoded
		}
	}

	if a.cfg.BackupS3Enabled {
		endpoint := strings.TrimSpace(a.cfg.BackupS3Endpoint)
		accessKey := strings.TrimSpace(a.cfg.BackupS3AccessKey)
		secretKey := strings.TrimSpace(a.cfg.BackupS3SecretKey)
		bucket := strings.TrimSpace(a.cfg.BackupS3Bucket)

		if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
			logger.Warn("Incomplete S3 backup configuration; remote uploads disabled", map[string]interface{}{
				"endpoint_configured": endpoint != "",
				"bucket_configured":   bucket != "",
				"access_configured":   accessKey != "" && secretKey != "",
			})
		} else {
			backupOptions.S3 = &service.BackupS3Config{
				Endpoint:  endpoint,
				AccessKey: accessKey,
				SecretKey: secretKey,
				Bucket:    bucket,
				Region:    strings.TrimSpace(a.cfg.BackupS3Region),
				UseSSL:    a.cfg.BackupS3UseSSL,
				Prefix:    strings.Trim(a.cfg.BackupS3Prefix, "/"),
			}
		}
	}

	backupService := service.NewBackupService(a.db, a.repositories.Setting, backupOptions)

	authService := service.NewAuthService(a.repositories.User, a.cfg.JWTSecret)
	pageService := service.NewPageService(a.repositories.Page, a.cache, a.themeManager)
	homepageService := service.NewHomepageService(a.repositories.Setting, a.repositories.Page)
	socialLinkService := service.NewSocialLinkService(a.repositories.SocialLink)
	menuService := service.NewMenuService(a.repositories.Menu)
	advertisingService := service.NewAdvertisingService(a.repositories.Setting)
	fontService := service.NewFontService(a.repositories.Setting)

	themeService := service.NewThemeService(
		a.repositories.Setting,
		a.themeManager,
		a.options.DefaultTheme,
	)

	pluginService := service.NewPluginService(
		a.repositories.Plugin,
		a.pluginManager,
		a.pluginRuntime,
	)

	a.services = serviceContainer{
		Auth:           authService,
		Category:       nil,
		Post:           nil,
		Comment:        nil,
		Search:         nil,
		Upload:         uploadService,
		Backup:         backupService,
		Page:           pageService,
		Setup:          setupService,
		Language:       languageService,
		Homepage:       homepageService,
		SocialLink:     socialLinkService,
		Menu:           menuService,
		Theme:          themeService,
		Advertising:    advertisingService,
		Plugin:         pluginService,
		Font:           fontService,
		CourseVideo:    nil,
		CourseContent:  nil,
		CourseTopic:    nil,
		CoursePackage:  nil,
		CourseTest:     nil,
		CourseCheckout: nil,
		ForumCategory:  nil,
		ForumQuestion:  nil,
		ForumAnswer:    nil,
	}

	a.registerPluginServiceBindings()

	backupService.InitializeAutoBackups()
}

func (a *Application) initHandlers() error {
	commentGuard := bloghandlers.NewCommentGuard(a.cfg)

	a.handlers = handlerContainer{
		Auth:             handlers.NewAuthHandler(a.services.Auth),
		Category:         bloghandlers.NewCategoryHandler(nil),
		Post:             bloghandlers.NewPostHandler(nil),
		Comment:          bloghandlers.NewCommentHandler(nil, a.services.Auth, commentGuard),
		Search:           bloghandlers.NewSearchHandler(nil),
		Upload:           handlers.NewUploadHandler(a.services.Upload),
		Backup:           handlers.NewBackupHandler(a.services.Backup),
		Page:             handlers.NewPageHandler(a.services.Page),
		PageBuilder:      handlers.NewPageBuilderHandler(a.services.Page),
		Setup:            handlers.NewSetupHandler(a.services.Setup, a.services.Font, a.cfg),
		Homepage:         handlers.NewHomepageHandler(a.services.Homepage),
		SocialLink:       handlers.NewSocialLinkHandler(a.services.SocialLink),
		Menu:             handlers.NewMenuHandler(a.services.Menu),
		SEO:              handlers.NewSEOHandler(nil, a.services.Page, nil, a.services.Setup, a.services.Language, a.cfg),
		Advertising:      handlers.NewAdvertisingHandler(a.services.Advertising),
		Plugin:           handlers.NewPluginHandler(a.services.Plugin),
		CourseVideo:      coursehandlers.NewVideoHandler(nil),
		CourseContent:    coursehandlers.NewContentHandler(nil),
		CourseTopic:      coursehandlers.NewTopicHandler(nil),
		CourseTest:       coursehandlers.NewTestHandler(nil),
		CoursePackage:    coursehandlers.NewPackageHandler(nil),
		CourseCheckout:   coursehandlers.NewCheckoutHandler(nil),
		ForumCategory:    forumhandlers.NewCategoryHandler(nil),
		ForumQuestion:    forumhandlers.NewQuestionHandler(nil),
		ArchiveDirectory: archivehandlers.NewDirectoryHandler(nil),
		ArchiveFile:      archivehandlers.NewFileHandler(nil),
		ArchivePublic:    archivehandlers.NewPublicHandler(nil, nil),
		ForumAnswer:      forumhandlers.NewAnswerHandler(nil),
	}

	templateHandler, err := handlers.NewTemplateHandler(
		nil,
		a.services.Page,
		a.services.Auth,
		nil,
		nil,
		a.services.Setup,
		a.services.Language,
		a.services.Homepage,
		nil,
		a.services.SocialLink,
		a.services.Menu,
		a.services.Font,
		a.services.Advertising,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		a.cfg,
		a.themeManager,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize template handler: %w", err)
	}

	a.templateHandler = templateHandler

	a.handlers.Font = handlers.NewFontHandler(a.services.Font)

	a.handlers.Theme = handlers.NewThemeHandler(
		a.services.Theme,
		a.services.Page,
		a.services.Menu,
		nil,
		a.repositories.User,
		a.templateHandler,
	)
	a.registerPluginHandlerBindings()
	return nil
}

func (a *Application) initRouter() error {
	if a.cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(nil); err != nil {
		return fmt.Errorf("failed to configure trusted proxies: %w", err)
	}

	// Set max multipart memory to handle large file uploads
	// Files larger than this will be written to disk temporarily
	router.MaxMultipartMemory = 32 << 20 // 32MB in memory, rest on disk

	router.Use(logger.GinRecovery(true))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(logger.GinLogger())
	router.Use(middleware.SecurityHeadersMiddleware(a.cfg, a.services.Advertising))
	router.Use(middleware.MetricsMiddleware())

	// Set rate limit manager in context for all requests
	router.Use(func(c *gin.Context) {
		if a.rateLimitManager != nil {
			c.Set("rateLimitManager", a.rateLimitManager)
		}
		c.Next()
	})

	router.Use(middleware.RateLimitMiddleware(a.cfg))
	router.Use(middleware.CSRFMiddleware())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     a.cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.Use(middleware.SetupMiddleware(a.services.Setup, a.cfg))
	router.Use(middleware.LanguageNegotiationMiddleware(func() *languageservice.LanguageService {
		return a.services.Language
	}))

	if a.themeManager != nil {
		if active := a.themeManager.Active(); active != nil {
			logger.Info("Active theme loaded", map[string]interface{}{"theme": active.Slug})
		}
	}

	router.GET("/health", middleware.NoIndexMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
		})
	})

	router.GET("/metrics", middleware.NoIndexMiddleware(), a.metricsHandler())

	if a.themeManager != nil {
		router.StaticFS("/static", theme.NewFileSystem(a.themeManager, "./static"))
	} else {
		router.Static("/static", "./static")
	}
	router.Static("/uploads", a.cfg.UploadDir)
	router.StaticFile("/favicon.ico", "./favicon.ico")

	if a.handlers.SEO != nil {
		router.GET("/sitemap.xml", a.handlers.SEO.Sitemap)
		router.GET("/robots.txt", a.handlers.SEO.Robots)
	}

	router.GET("/.well-known/appspecific/com.chrome.devtools.json", middleware.NoIndexMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Chrome DevTools discovery is not enabled",
		})
	})

	router.GET("/debug-templates", middleware.NoIndexMiddleware(), func(c *gin.Context) {
		if a.themeManager == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "theme manager unavailable"})
			return
		}

		active := a.themeManager.Active()
		if active == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no active theme"})
			return
		}

		names, err := active.TemplateNames()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "failed to enumerate templates",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"theme":     active.Slug,
			"templates": names,
		})
	})

	router.GET("/", a.templateHandler.RenderIndex)
	router.GET("/login", a.templateHandler.RenderLogin)
	router.GET("/register", a.templateHandler.RenderRegister)
	router.GET("/setup", a.templateHandler.RenderSetup)
	router.GET("/setup/key-required", a.templateHandler.RenderSetupKeyRequired)
	router.GET("/profile", a.templateHandler.RenderProfile)
	router.GET("/courses/:slug", a.templateHandler.RenderCourse)
	router.GET("/admin", a.templateHandler.RenderAdmin)
	router.GET("/blog/post/:slug", a.templateHandler.RenderPost)
	router.GET("/page/:slug", a.templateHandler.RenderPage)
	router.GET("/blog", a.templateHandler.RenderBlog)
	router.GET("/search", a.templateHandler.RenderSearch)
	router.GET("/forum", a.templateHandler.RenderForum)
	router.GET("/forum/:slug", a.templateHandler.RenderForumQuestion)
	router.GET("/category/:slug", a.templateHandler.RenderCategory)
	router.GET("/tag/:slug", a.templateHandler.RenderTag)
	router.GET("/archive", a.templateHandler.RenderArchive)
	router.GET("/archive/*path", a.templateHandler.RenderArchivePath)

	v1 := router.Group("/api/v1")
	v1.Use(middleware.NoIndexMiddleware())
	{
		public := v1.Group("")
		{
			public.GET("/setup/status", a.handlers.Setup.Status)
			public.GET("/setup/progress", a.handlers.Setup.GetStepProgress)
			public.POST("/setup/step", a.handlers.Setup.SaveStep)
			public.POST("/setup/complete", a.handlers.Setup.CompleteStepwiseSetup)
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
			public.POST("/courses/checkout/webhook", a.handlers.CourseCheckout.HandleWebhook)
			public.GET("/forum/questions", a.handlers.ForumQuestion.List)
			public.GET("/forum/questions/:id", a.handlers.ForumQuestion.GetByID)
			public.GET("/forum/categories", a.handlers.ForumCategory.List)
			public.GET("/forum/categories/:id", a.handlers.ForumCategory.GetByID)
			public.GET("/archive/tree", a.handlers.ArchivePublic.Tree)
			public.GET("/archive/directories/*path", a.handlers.ArchivePublic.GetDirectory)
			public.GET("/archive/files/*path", a.handlers.ArchivePublic.GetFile)
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
			protected.POST("/courses/checkout", a.handlers.CourseCheckout.CreateSession)
			protected.GET("/courses/packages/:id", a.handlers.CoursePackage.GetForUser)
			protected.GET("/courses/tests/:id", a.handlers.CourseTest.Get)
			protected.POST("/courses/tests/:id/submit", a.handlers.CourseTest.Submit)
			protected.POST("/forum/questions", a.handlers.ForumQuestion.Create)
			protected.PUT("/forum/questions/:id", a.handlers.ForumQuestion.Update)
			protected.DELETE("/forum/questions/:id", a.handlers.ForumQuestion.Delete)
			protected.POST("/forum/questions/:id/vote", a.handlers.ForumQuestion.Vote)
			protected.POST("/forum/questions/:id/answers", a.handlers.ForumAnswer.Create)
			protected.PUT("/forum/answers/:id", a.handlers.ForumAnswer.Update)
			protected.DELETE("/forum/answers/:id", a.handlers.ForumAnswer.Delete)
			protected.POST("/forum/answers/:id/vote", a.handlers.ForumAnswer.Vote)
		}

		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware(a.cfg.JWTSecret))

		content := admin.Group("")
		content.Use(middleware.RequirePermissions(authorization.PermissionManageAllContent))
		{
			content.POST("/posts", a.handlers.Post.Create)
			content.PUT("/posts/:id", a.handlers.Post.Update)
			content.DELETE("/posts/:id", a.handlers.Post.Delete)
			content.GET("/posts", a.handlers.Post.GetAllAdmin)
			content.GET("/posts/:id/analytics", a.handlers.Post.GetAnalytics)

			content.POST("/pages", a.handlers.Page.Create)
			content.PUT("/pages/:id", a.handlers.Page.Update)
			content.DELETE("/pages/:id", a.handlers.Page.Delete)
			content.GET("/pages", a.handlers.Page.GetAllAdmin)
			content.POST("/pages/sections/padding", a.handlers.Page.UpdateAllSectionPadding)

			// Enhanced page builder endpoints
			content.GET("/pages/:id/builder", a.handlers.PageBuilder.GetPageBuilder)
			content.POST("/pages/:id/duplicate", a.handlers.PageBuilder.DuplicatePage)
			content.POST("/pages/:id/sections/reorder", a.handlers.PageBuilder.ReorderSections)
			content.POST("/pages/:id/sections", a.handlers.PageBuilder.AddSection)
			content.PUT("/pages/:id/sections/:sectionId", a.handlers.PageBuilder.UpdateSection)
			content.DELETE("/pages/:id/sections/:sectionId", a.handlers.PageBuilder.DeleteSection)
			content.POST("/pages/:id/sections/:sectionId/duplicate", a.handlers.PageBuilder.DuplicateSection)
			content.GET("/pages/templates", a.handlers.PageBuilder.GetPageTemplates)
			content.POST("/pages/templates/:templateId", a.handlers.PageBuilder.CreateFromTemplate)
			content.GET("/pages/:id/preview", a.handlers.PageBuilder.PreviewPage)
			content.POST("/pages/validate-slug", a.handlers.PageBuilder.ValidatePageSlug)
			content.GET("/pages/builder/config", a.handlers.PageBuilder.GetPageBuilderConfig)

			// Upload operations with rate limiting
			uploads := content.Group("")
			uploads.Use(middleware.UploadRateLimitMiddleware(a.cfg))
			{
				uploads.POST("/upload", a.handlers.Upload.Upload)
			}
			content.GET("/uploads", a.handlers.Upload.List)
			content.DELETE("/uploads", a.handlers.Upload.Delete)
			content.PUT("/uploads/rename", a.handlers.Upload.Rename)

			content.POST("/categories", a.handlers.Category.Create)
			content.PUT("/categories/:id", a.handlers.Category.Update)
			content.DELETE("/categories/:id", a.handlers.Category.Delete)

			content.GET("/forum/categories", a.handlers.ForumCategory.List)
			content.GET("/forum/categories/:id", a.handlers.ForumCategory.GetByID)
			content.POST("/forum/categories", a.handlers.ForumCategory.Create)
			content.PUT("/forum/categories/:id", a.handlers.ForumCategory.Update)
			content.DELETE("/forum/categories/:id", a.handlers.ForumCategory.Delete)
			content.DELETE("/forum/questions/:id", a.handlers.ForumQuestion.AdminDelete)

			content.POST("/courses/videos", a.handlers.CourseVideo.Create)
			content.PUT("/courses/videos/:id", a.handlers.CourseVideo.Update)
			content.PUT("/courses/videos/:id/subtitle", a.handlers.CourseVideo.UpdateSubtitle)
			content.DELETE("/courses/videos/:id", a.handlers.CourseVideo.Delete)
			content.GET("/courses/videos", a.handlers.CourseVideo.List)
			content.GET("/courses/videos/:id", a.handlers.CourseVideo.Get)
			content.POST("/courses/contents", a.handlers.CourseContent.Create)
			content.PUT("/courses/contents/:id", a.handlers.CourseContent.Update)
			content.DELETE("/courses/contents/:id", a.handlers.CourseContent.Delete)
			content.GET("/courses/contents", a.handlers.CourseContent.List)
			content.GET("/courses/contents/:id", a.handlers.CourseContent.Get)

			content.POST("/courses/topics", a.handlers.CourseTopic.Create)
			content.PUT("/courses/topics/:id", a.handlers.CourseTopic.Update)
			content.PUT("/courses/topics/:id/videos", a.handlers.CourseTopic.UpdateVideos)
			content.PUT("/courses/topics/:id/steps", a.handlers.CourseTopic.UpdateSteps)
			content.DELETE("/courses/topics/:id", a.handlers.CourseTopic.Delete)
			content.GET("/courses/topics", a.handlers.CourseTopic.List)
			content.GET("/courses/topics/:id", a.handlers.CourseTopic.Get)

			content.POST("/courses/tests", a.handlers.CourseTest.Create)
			content.PUT("/courses/tests/:id", a.handlers.CourseTest.Update)
			content.DELETE("/courses/tests/:id", a.handlers.CourseTest.Delete)
			content.GET("/courses/tests", a.handlers.CourseTest.List)
			content.GET("/courses/tests/:id", a.handlers.CourseTest.Get)

			content.POST("/courses/packages", a.handlers.CoursePackage.Create)
			content.PUT("/courses/packages/:id", a.handlers.CoursePackage.Update)
			content.PUT("/courses/packages/:id/topics", a.handlers.CoursePackage.UpdateTopics)
			content.POST("/courses/packages/:id/grants", a.handlers.CoursePackage.GrantToUser)
			content.DELETE("/courses/packages/:id", a.handlers.CoursePackage.Delete)
			content.GET("/courses/packages", a.handlers.CoursePackage.List)
			content.GET("/courses/packages/:id", a.handlers.CoursePackage.Get)

			content.GET("/archive/directories", a.handlers.ArchiveDirectory.List)
			content.GET("/archive/directories/:id", a.handlers.ArchiveDirectory.Get)
			content.POST("/archive/directories", a.handlers.ArchiveDirectory.Create)
			content.PUT("/archive/directories/:id", a.handlers.ArchiveDirectory.Update)
			content.DELETE("/archive/directories/:id", a.handlers.ArchiveDirectory.Delete)

			content.GET("/archive/files", a.handlers.ArchiveFile.List)
			content.GET("/archive/files/:id", a.handlers.ArchiveFile.Get)
			content.POST("/archive/files", a.handlers.ArchiveFile.Create)
			content.PUT("/archive/files/:id", a.handlers.ArchiveFile.Update)
			content.DELETE("/archive/files/:id", a.handlers.ArchiveFile.Delete)

			content.DELETE("/tags/:id", a.handlers.Post.DeleteTag)
		}

		publish := admin.Group("")
		publish.Use(middleware.RequirePermissions(authorization.PermissionPublishContent))
		{
			publish.PUT("/posts/:id/publish", a.handlers.Post.PublishPost)
			publish.PUT("/posts/:id/unpublish", a.handlers.Post.UnpublishPost)
			publish.PUT("/pages/:id/publish", a.handlers.Page.PublishPage)
			publish.PUT("/pages/:id/unpublish", a.handlers.Page.UnpublishPage)
		}

		users := admin.Group("")
		users.Use(middleware.RequirePermissions(authorization.PermissionManageUsers))
		{
			users.GET("/users", a.handlers.Auth.GetAllUsers)
			users.GET("/users/:id", a.handlers.Auth.GetUser)
			users.DELETE("/users/:id", a.handlers.Auth.DeleteUser)
			users.PUT("/users/:id/role", a.handlers.Auth.UpdateUserRole)
			users.PUT("/users/:id/status", a.handlers.Auth.UpdateUserStatus)
		}

		comments := admin.Group("")
		comments.Use(middleware.RequirePermissions(authorization.PermissionModerateComments))
		{
			comments.GET("/comments", a.handlers.Comment.GetAll)
			comments.DELETE("/comments/:id", a.handlers.Comment.Delete)
			comments.PUT("/comments/:id/approve", a.handlers.Comment.ApproveComment)
			comments.PUT("/comments/:id/reject", a.handlers.Comment.RejectComment)
		}

		settings := admin.Group("")
		settings.Use(middleware.RequirePermissions(authorization.PermissionManageSettings))
		{
			settings.GET("/settings/site", a.handlers.Setup.GetSiteSettings)
			settings.PUT("/settings/site", a.handlers.Setup.UpdateSiteSettings)
			settings.GET("/settings/homepage", a.handlers.Homepage.Get)
			settings.PUT("/settings/homepage", a.handlers.Homepage.Update)

			// Settings file upload operations with rate limiting
			settingsUploads := settings.Group("")
			settingsUploads.Use(middleware.UploadRateLimitMiddleware(a.cfg))
			{
				settingsUploads.POST("/settings/favicon", a.handlers.Setup.UploadFavicon)
				settingsUploads.POST("/settings/logo", a.handlers.Setup.UploadLogo)
			}

			settings.GET("/settings/advertising", a.handlers.Advertising.Get)
			settings.PUT("/settings/advertising", a.handlers.Advertising.Update)

			settings.GET("/social-links", a.handlers.SocialLink.List)
			settings.POST("/social-links", a.handlers.SocialLink.Create)
			settings.PUT("/social-links/:id", a.handlers.SocialLink.Update)
			settings.DELETE("/social-links/:id", a.handlers.SocialLink.Delete)

			settings.GET("/settings/fonts", a.handlers.Font.List)
			settings.POST("/settings/fonts", a.handlers.Font.Create)
			settings.PUT("/settings/fonts/:id", a.handlers.Font.Update)
			settings.DELETE("/settings/fonts/:id", a.handlers.Font.Delete)
			settings.PUT("/settings/fonts/reorder", a.handlers.Font.Reorder)

			settings.GET("/menu-items", a.handlers.Menu.List)
			settings.POST("/menu-items", a.handlers.Menu.Create)
			settings.PUT("/menu-items/reorder", a.handlers.Menu.Reorder)
			settings.PUT("/menu-items/:id", a.handlers.Menu.Update)
			settings.DELETE("/menu-items/:id", a.handlers.Menu.Delete)

			settings.GET("/stats", handlers.GetStatistics(a.db))

			if a.cache != nil {
				settings.DELETE("/cache", handlers.ClearCache(a.cache))
			}
		}

		themes := admin.Group("")
		themes.Use(middleware.RequirePermissions(authorization.PermissionManageThemes))
		{
			themes.GET("/themes", a.handlers.Theme.List)
			themes.PUT("/themes/:slug/activate", a.handlers.Theme.Activate)
			themes.PUT("/themes/:slug/reload", a.handlers.Theme.Reload)
		}

		plugins := admin.Group("")
		plugins.Use(middleware.RequirePermissions(authorization.PermissionManagePlugins))
		{
			plugins.GET("/plugins", a.handlers.Plugin.List)
			plugins.POST("/plugins", a.handlers.Plugin.Install)
			plugins.PUT("/plugins/:slug/activate", a.handlers.Plugin.Activate)
			plugins.PUT("/plugins/:slug/deactivate", a.handlers.Plugin.Deactivate)
			plugins.DELETE("/plugins/:slug", a.handlers.Plugin.Delete)
		}

		backups := admin.Group("")
		backups.Use(middleware.RequirePermissions(authorization.PermissionManageBackups))
		{
			backups.GET("/backups/settings", a.handlers.Backup.GetSettings)
			backups.PUT("/backups/settings", a.handlers.Backup.UpdateSettings)

			// Backup export/import operations with rate limiting
			backupOps := backups.Group("")
			backupOps.Use(middleware.BackupRateLimitMiddleware(a.cfg))
			{
				backupOps.GET("/backups/export", a.handlers.Backup.Export)
				backupOps.POST("/backups/import", a.handlers.Backup.Import)
			}
		}
	}

	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.Header("X-Robots-Tag", "noindex, nofollow")
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Route not found",
				"path":  c.Request.URL.Path,
			})
			return
		}

		if a.templateHandler != nil {
			if a.templateHandler.TryRenderPage(c) {
				return
			}
			a.templateHandler.RenderErrorPage(c, http.StatusNotFound, "404 - Page not found", "The requested page could not be found")
			return
		}

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Route not found",
			"path":  c.Request.URL.Path,
		})
	})

	a.router = router
	return nil
}

func (a *Application) initPluginRuntime() error {
	if a.pluginRuntime == nil {
		a.pluginRuntime = pluginruntime.New()
	}

	for slug, factory := range pluginregistry.All() {
		feature, err := factory(a)
		if err != nil {
			logger.Error(err, "Failed to initialize plugin feature", map[string]interface{}{"slug": slug})
			continue
		}
		a.pluginRuntime.Register(slug, feature)
	}

	if a.services.Plugin != nil {
		if err := a.services.Plugin.ApplyRuntimeState(); err != nil {
			return err
		}
	}

	return nil
}

func (a *Application) metricsHandler() gin.HandlerFunc {
	promHandler := promhttp.Handler()

	allowedExact := make(map[string]struct{})
	var allowedNetworks []*net.IPNet

	for _, value := range a.cfg.MetricsAllowedIPs {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		if strings.Contains(trimmed, "/") {
			if _, network, err := net.ParseCIDR(trimmed); err == nil {
				allowedNetworks = append(allowedNetworks, network)
			} else {
				logger.Warn("Invalid metrics allowed IP entry", map[string]interface{}{
					"value": trimmed,
					"error": err.Error(),
				})
			}
			continue
		}

		ip := net.ParseIP(trimmed)
		if ip == nil {
			logger.Warn("Invalid metrics allowed IP entry", map[string]interface{}{
				"value": trimmed,
			})
			continue
		}

		allowedExact[ip.String()] = struct{}{}
	}

	authUser := strings.TrimSpace(a.cfg.MetricsBasicAuthUsername)
	authPassword := a.cfg.MetricsBasicAuthPassword
	authConfigured := authUser != "" && authPassword != ""
	ipConfigured := len(allowedExact) > 0 || len(allowedNetworks) > 0

	return func(c *gin.Context) {
		if !a.cfg.EnableMetrics {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		clientIPStr := c.ClientIP()
		clientIP := net.ParseIP(clientIPStr)

		if ipConfigured {
			if clientIP != nil {
				if _, ok := allowedExact[clientIP.String()]; ok {
					promHandler.ServeHTTP(c.Writer, c.Request)
					c.Abort()
					return
				}

				for _, network := range allowedNetworks {
					if network.Contains(clientIP) {
						promHandler.ServeHTTP(c.Writer, c.Request)
						c.Abort()
						return
					}
				}
			}

			if !authConfigured {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
		}

		if authConfigured {
			username, password, ok := c.Request.BasicAuth()
			if ok && subtle.ConstantTimeCompare([]byte(username), []byte(authUser)) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte(authPassword)) == 1 {
				promHandler.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}

			c.Header("WWW-Authenticate", `Basic realm="metrics"`)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		if clientIP != nil && clientIP.IsLoopback() {
			promHandler.ServeHTTP(c.Writer, c.Request)
			c.Abort()
			return
		}

		c.AbortWithStatus(http.StatusForbidden)
	}
}
