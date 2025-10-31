package registry

import (
	"strings"
	"sync"

	"constructor-script-backend/internal/plugin/host"
	pluginruntime "constructor-script-backend/internal/plugin/runtime"
)

type Factory func(host.Host) (pluginruntime.Feature, error)

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

func Register(slug string, factory Factory) {
	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" || factory == nil {
		return
	}

	mu.Lock()
	factories[cleaned] = factory
	mu.Unlock()
}

func FactoryFor(slug string) (Factory, bool) {
	cleaned := strings.ToLower(strings.TrimSpace(slug))
	if cleaned == "" {
		return nil, false
	}

	mu.RLock()
	factory, ok := factories[cleaned]
	mu.RUnlock()
	return factory, ok
}

func All() map[string]Factory {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]Factory, len(factories))
	for slug, factory := range factories {
		result[slug] = factory
	}
	return result
}
