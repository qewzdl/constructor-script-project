package adapters

import (
	"time"

	"constructor-script-backend/pkg/cache"
	"constructor-script-backend/pkg/pluginsdk"
)

// CacheAdapter adapts the host cache to pluginsdk.Cache interface
type CacheAdapter struct {
	cache *cache.Cache
}

func NewCacheAdapter(c *cache.Cache) pluginsdk.Cache {
	return &CacheAdapter{cache: c}
}

func (a *CacheAdapter) Get(key string) (interface{}, bool) {
	var result interface{}
	err := a.cache.Get(key, &result)
	return result, err == nil
}

func (a *CacheAdapter) Set(key string, value interface{}, expiration time.Duration) {
	_ = a.cache.Set(key, value, expiration)
}

func (a *CacheAdapter) Delete(key string) {
	_ = a.cache.Delete(key)
}

func (a *CacheAdapter) Clear() {
	a.cache.Clear()
}

func (a *CacheAdapter) Has(key string) bool {
	exists, _ := a.cache.Exists(key)
	return exists
}
