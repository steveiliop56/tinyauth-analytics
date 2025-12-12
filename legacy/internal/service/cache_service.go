package service

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

type CacheService struct {
	cache []cacheField
	mutex sync.RWMutex
}

func NewCacheService() *CacheService {
	return &CacheService{
		cache: make([]cacheField, 0),
	}
}

func (cs *CacheService) Set(key string, value any, ttl int64) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	expire := time.Now().Add(time.Duration(ttl) * time.Second).Unix()
	for i, field := range cs.cache {
		if field.key == key {
			cs.cache[i].value = value
			cs.cache[i].expire = expire
			return
		}
	}
	cs.cache = append(cs.cache, cacheField{
		key:    key,
		value:  value,
		expire: expire,
	})
}

func (cs *CacheService) Get(key string) (any, bool) {
	cs.mutex.RLock()
	defer cs.mutex.RUnlock()
	for _, field := range cs.cache {
		if field.key == key {
			if time.Now().Unix() > field.expire {
				cs.Delete(key)
				return nil, false
			}
			return field.value, true
		}
	}
	return nil, false
}

func (cs *CacheService) Delete(key string) {
	cs.mutex.Lock()
	defer cs.mutex.Unlock()
	for i, field := range cs.cache {
		if field.key == key {
			cs.cache = slices.Delete(cs.cache, i, i+1)
			return
		}
	}
}

func (cs *CacheService) Clear() {
	cs.cache = make([]cacheField, 0)
}
