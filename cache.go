package main

import (
	"slices"
	"sync"
	"time"
)

type cacheField struct {
	key    string
	value  any
	expire int64
}

type Cache struct {
	cache []cacheField
	mutex sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		cache: make([]cacheField, 0),
	}
}

func (c *Cache) Set(key string, value any, ttl int64) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	expire := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	for i, field := range c.cache {
		if field.key == key {
			c.cache[i].value = value
			c.cache[i].expire = expire
			return
		}
	}
	c.cache = append(c.cache, cacheField{
		key:    key,
		value:  value,
		expire: expire,
	})
}

func (c *Cache) Get(key string) (any, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	for _, field := range c.cache {
		if field.key == key {
			if time.Now().Unix() > field.expire {
				c.Delete(key)
				return nil, false
			}
			return field.value, true
		}
	}
	return nil, false
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	for i, field := range c.cache {
		if field.key == key {
			c.cache = slices.Delete(c.cache, i, i+1)
			return
		}
	}
}

func (c *Cache) Flush() {
	c.cache = make([]cacheField, 0)
}
