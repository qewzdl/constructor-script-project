package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	// defaultOperationTimeout is the timeout for individual Redis operations
	defaultOperationTimeout = 5 * time.Second
)

type Cache struct {
	client  *redis.Client
	enabled bool
}

func NewCache(addr string, enable bool) (*Cache, error) {
	if !enable {
		return &Cache{enabled: false}, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := client.Ping(ctx).Result(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &Cache{
		client:  client,
		enabled: true,
	}, nil
}

// operationContext creates a context with timeout for Redis operations
func (c *Cache) operationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultOperationTimeout)
}

func (c *Cache) Set(key string, value interface{}, expiration time.Duration) error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, jsonData, expiration).Err()
}

func (c *Cache) Get(key string, dest interface{}) error {
	if !c.enabled {
		return fmt.Errorf("cache disabled")
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return fmt.Errorf("key not found")
	} else if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), dest)
}

func (c *Cache) Delete(key string) error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	return c.client.Del(ctx, key).Err()
}

func (c *Cache) DeletePattern(pattern string) error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (c *Cache) Exists(key string) (bool, error) {
	if !c.enabled {
		return false, nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	val, err := c.client.Exists(ctx, key).Result()
	return val > 0, err
}

func (c *Cache) Increment(key string) (int64, error) {
	if !c.enabled {
		return 0, nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	return c.client.Incr(ctx, key).Result()
}

func (c *Cache) Expire(key string, expiration time.Duration) error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	return c.client.Expire(ctx, key, expiration).Err()
}

func (c *Cache) FlushAll() error {
	if !c.enabled {
		return nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	return c.client.FlushAll(ctx).Err()
}

func (c *Cache) Close() error {
	if !c.enabled {
		return nil
	}
	return c.client.Close()
}

func (c *Cache) CachePost(postID uint, post interface{}) error {
	return c.Set(fmt.Sprintf("post:%d", postID), post, 1*time.Hour)
}

func (c *Cache) GetCachedPost(postID uint, dest interface{}) error {
	return c.Get(fmt.Sprintf("post:%d", postID), dest)
}

func (c *Cache) InvalidatePost(postID uint) error {
	return c.Delete(fmt.Sprintf("post:%d", postID))
}

func (c *Cache) CachePosts(cacheKey string, posts interface{}) error {
	return c.Set(cacheKey, posts, 5*time.Minute)
}

func (c *Cache) GetCachedPosts(cacheKey string, dest interface{}) error {
	return c.Get(cacheKey, dest)
}

func (c *Cache) InvalidatePostsCache() error {
	return c.DeletePattern("posts:*")
}

func (c *Cache) CacheCategory(categoryID uint, category interface{}) error {
	return c.Set(fmt.Sprintf("category:%d", categoryID), category, 2*time.Hour)
}

func (c *Cache) GetCachedCategory(categoryID uint, dest interface{}) error {
	return c.Get(fmt.Sprintf("category:%d", categoryID), dest)
}

func (c *Cache) InvalidateCategory(categoryID uint) error {
	return c.Delete(fmt.Sprintf("category:%d", categoryID))
}

func (c *Cache) IncrementViews(postID uint) (int64, error) {
	return c.Increment(fmt.Sprintf("views:%d", postID))
}

func (c *Cache) GetViews(postID uint) (int64, error) {
	if !c.enabled {
		return 0, nil
	}

	ctx, cancel := c.operationContext()
	defer cancel()

	val, err := c.client.Get(ctx, fmt.Sprintf("views:%d", postID)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// CachePage - кэширование страницы
func (c *Cache) CachePage(pageID uint, page interface{}) error {
	return c.Set(fmt.Sprintf("page:%d", pageID), page, 1*time.Hour)
}

// GetCachedPage - получение страницы из кэша
func (c *Cache) GetCachedPage(pageID uint, dest interface{}) error {
	return c.Get(fmt.Sprintf("page:%d", pageID), dest)
}

// InvalidatePage - инвалидация кэша страницы
func (c *Cache) InvalidatePage(pageID uint) error {
	// Удаляем кэш по ID
	if err := c.Delete(fmt.Sprintf("page:%d", pageID)); err != nil {
		return err
	}
	// Удаляем кэш по slug (используем pattern)
	if err := c.DeletePattern("page:slug:*"); err != nil {
		return err
	}
	return c.DeletePattern("page:path:*")
}

// InvalidatePagesCache - инвалидация всего кэша страниц
func (c *Cache) InvalidatePagesCache() error {
	return c.DeletePattern("page*")
}
