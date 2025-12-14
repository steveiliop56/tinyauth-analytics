package main

import (
	"sync"
	"time"
)

type cacheField struct {
	value  any
	expire int64
}

type Cache struct {
	cache map[string]cacheField
	mutex sync.RWMutex
}

func NewCache() *Cache {
	cache := &Cache{
		cache: make(map[string]cacheField),
	}
	cache.cleanup()
	return cache
}

func (c *Cache) Set(key string, value any, ttl int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	expire := time.Now().Add(time.Duration(ttl) * time.Second).Unix()

	c.cache[key] = cacheField{
		value:  value,
		expire: expire,
	}
}

func (c *Cache) Get(key string) (any, bool) {
	c.mutex.RLock()

	field, ok := c.cache[key]

	if !ok {
		c.mutex.RUnlock()
		return nil, false
	}

	if time.Now().Unix() > field.expire {
		c.mutex.RUnlock()
		c.Delete(key)
		return nil, false
	}

	c.mutex.RUnlock()
	return field.value, true
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.cache, key)
}

func (c *Cache) Flush() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.cache = make(map[string]cacheField, 0)
}

func (c *Cache) cleanup() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			c.mutex.Lock()
			for key, field := range c.cache {
				if time.Now().Unix() > field.expire {
					delete(c.cache, key)
				}
			}
			c.mutex.Unlock()
		}
	}()
}
