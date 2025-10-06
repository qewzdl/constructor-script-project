package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	client  *redis.Client
	ctx     context.Context
	enabled bool
}

func NewCache(addr string, enable bool) *Cache {
	if !enable {
		return &Cache{enabled: false}
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

	ctx := context.Background()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	return &Cache{
		client:  client,
		ctx:     ctx,
		enabled: true,
	}
}

func (c *Cache) Set(key string, value interface{}, expiration time.Duration) error {
	if !c.enabled {
		return nil
	}
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(c.ctx, key, jsonData, expiration).Err()
}

func (c *Cache) Get(key string, dest interface{}) error {
	if !c.enabled {
		return fmt.Errorf("cache disabled")
	}
	val, err := c.client.Get(c.ctx, key).Result()
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
	return c.client.Del(c.ctx, key).Err()
}

func (c *Cache) DeletePattern(pattern string) error {
	if !c.enabled {
		return nil
	}
	iter := c.client.Scan(c.ctx, 0, pattern, 0).Iterator()
	for iter.Next(c.ctx) {
		if err := c.client.Del(c.ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}
	return iter.Err()
}

func (c *Cache) Exists(key string) (bool, error) {
	if !c.enabled {
		return false, nil
	}
	val, err := c.client.Exists(c.ctx, key).Result()
	return val > 0, err
}

func (c *Cache) Increment(key string) (int64, error) {
	if !c.enabled {
		return 0, nil
	}
	return c.client.Incr(c.ctx, key).Result()
}

func (c *Cache) Expire(key string, expiration time.Duration) error {
	if !c.enabled {
		return nil
	}
	return c.client.Expire(c.ctx, key, expiration).Err()
}

func (c *Cache) FlushAll() error {
	if !c.enabled {
		return nil
	}
	return c.client.FlushAll(c.ctx).Err()
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
	val, err := c.client.Get(c.ctx, fmt.Sprintf("views:%d", postID)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
