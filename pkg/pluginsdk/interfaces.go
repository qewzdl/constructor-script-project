// Package pluginsdk provides interfaces and types for building independent plugins
package pluginsdk

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Plugin represents a plugin that can be activated and deactivated
type Plugin interface {
	// Name returns the plugin name
	Name() string
	// Version returns the plugin version
	Version() string
	// Activate is called when the plugin is activated
	Activate(ctx context.Context) error
	// Deactivate is called when the plugin is deactivated
	Deactivate(ctx context.Context) error
}

// Host provides access to application services for plugins
type Host interface {
	// DB returns the database connection
	DB() *gorm.DB
	// Cache returns the cache service
	Cache() Cache
	// Logger returns the logger
	Logger() Logger
	// Scheduler returns the background job scheduler
	Scheduler() Scheduler
	// ThemeManager returns the theme manager
	ThemeManager() ThemeManager
	// Config returns configuration value by key
	Config(key string) (string, bool)
}

// Cache provides caching functionality
type Cache interface {
	// Get retrieves value from cache
	Get(key string) (interface{}, bool)
	// Set stores value in cache with expiration
	Set(key string, value interface{}, expiration time.Duration)
	// Delete removes value from cache
	Delete(key string)
	// Clear removes all values from cache
	Clear()
	// Has checks if key exists in cache
	Has(key string) bool
}

// Logger provides logging functionality
type Logger interface {
	// Debug logs debug message
	Debug(msg string, fields ...interface{})
	// Info logs info message
	Info(msg string, fields ...interface{})
	// Warn logs warning message
	Warn(msg string, fields ...interface{})
	// Error logs error message
	Error(msg string, fields ...interface{})
	// Fatal logs fatal message and exits
	Fatal(msg string, fields ...interface{})
}

// Scheduler provides background job scheduling
type Scheduler interface {
	// Schedule adds a job to run at specified interval
	Schedule(name string, interval time.Duration, fn func() error) error
	// ScheduleOnce adds a job to run once after delay
	ScheduleOnce(name string, delay time.Duration, fn func() error) error
	// Cancel cancels a scheduled job
	Cancel(name string) error
}

// ThemeManager provides theme-related functionality
type ThemeManager interface {
	// RenderTemplate renders a template with data
	RenderTemplate(name string, data interface{}) (string, error)
	// GetActiveTheme returns active theme name
	GetActiveTheme() string
	// ThemeExists checks if theme exists
	ThemeExists(name string) bool
}

// RepositoryRegistry allows plugins to register their repositories
type RepositoryRegistry interface {
	// Register registers a repository factory
	Register(name string, factory interface{})
	// Get retrieves a registered repository
	Get(name string) interface{}
}

// HandlerRegistry allows plugins to register HTTP handlers
type HandlerRegistry interface {
	// RegisterRoute registers an HTTP route
	RegisterRoute(method, path string, handler interface{})
	// RegisterMiddleware registers middleware
	RegisterMiddleware(name string, middleware interface{})
}

// ModelMigrator handles database migrations for plugin models
type ModelMigrator interface {
	// AutoMigrate runs auto migration for models
	AutoMigrate(models ...interface{}) error
}
