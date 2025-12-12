package adapters

import (
	"constructor-script-backend/internal/background"
	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/theme"
	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/pluginsdk"

	"gorm.io/gorm"
)

// HostAdapter implements pluginsdk.Host interface
type HostAdapter struct {
	db           *gorm.DB
	cache        pluginsdk.Cache
	logger       pluginsdk.Logger
	scheduler    pluginsdk.Scheduler
	themeManager pluginsdk.ThemeManager
	config       *config.Config
}

func NewHostAdapter(
	db *gorm.DB,
	cacheService *cache.Cache,
	scheduler *background.Scheduler,
	themeManager *theme.Manager,
	cfg *config.Config,
) pluginsdk.Host {
	return &HostAdapter{
		db:           db,
		cache:        NewCacheAdapter(cacheService),
		logger:       NewLoggerAdapter(),
		scheduler:    NewSchedulerAdapter(scheduler),
		themeManager: NewThemeManagerAdapter(themeManager),
		config:       cfg,
	}
}

func (h *HostAdapter) DB() *gorm.DB {
	return h.db
}

func (h *HostAdapter) Cache() pluginsdk.Cache {
	return h.cache
}

func (h *HostAdapter) Logger() pluginsdk.Logger {
	return h.logger
}

func (h *HostAdapter) Scheduler() pluginsdk.Scheduler {
	return h.scheduler
}

func (h *HostAdapter) ThemeManager() pluginsdk.ThemeManager {
	return h.themeManager
}

func (h *HostAdapter) Config(key string) (string, bool) {
	if h.config == nil {
		return "", false
	}
	// Implement config lookup based on your config structure
	// This is a placeholder implementation
	return "", false
}
