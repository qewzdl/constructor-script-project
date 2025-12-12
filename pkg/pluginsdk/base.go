package pluginsdk

import (
	"context"

	"gorm.io/gorm"
)

// BasePlugin provides default implementation for Plugin interface
type BasePlugin struct {
	name    string
	version string
	host    Host
}

// NewBasePlugin creates a new base plugin
func NewBasePlugin(name, version string, host Host) *BasePlugin {
	return &BasePlugin{
		name:    name,
		version: version,
		host:    host,
	}
}

// Name returns the plugin name
func (p *BasePlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *BasePlugin) Version() string {
	return p.version
}

// Host returns the host
func (p *BasePlugin) Host() Host {
	return p.host
}

// Activate is called when the plugin is activated (default implementation)
func (p *BasePlugin) Activate(ctx context.Context) error {
	return nil
}

// Deactivate is called when the plugin is deactivated (default implementation)
func (p *BasePlugin) Deactivate(ctx context.Context) error {
	return nil
}

// DB returns the database connection
func (p *BasePlugin) DB() *gorm.DB {
	if p.host != nil {
		return p.host.DB()
	}
	return nil
}

// Cache returns the cache service
func (p *BasePlugin) Cache() Cache {
	if p.host != nil {
		return p.host.Cache()
	}
	return nil
}

// Logger returns the logger
func (p *BasePlugin) Logger() Logger {
	if p.host != nil {
		return p.host.Logger()
	}
	return nil
}

// Scheduler returns the scheduler
func (p *BasePlugin) Scheduler() Scheduler {
	if p.host != nil {
		return p.host.Scheduler()
	}
	return nil
}
