package middleware

import (
	"sync"
	"time"
	"tinyauth-analytics/internal/service"

	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RateLimitMiddleware struct {
	database *gorm.DB
	cache    *service.CacheService
	mutex    sync.RWMutex
}

func NewRateLimitMiddleware(database *gorm.DB, cache *service.CacheService) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		database: database,
		cache:    cache,
	}
}

func (m *RateLimitMiddleware) Init() error {
	return nil
}

func (m *RateLimitMiddleware) Middleware(count int) gin.HandlerFunc {
	return func(c *gin.Context) {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		clientIP := m.getClientIP(c)

		if clientIP == "" {
			c.JSON(500, gin.H{
				"status":  500,
				"message": "Failed to determine client IP",
			})
			c.Abort()
			return
		}

		value, exists := m.cache.Get(clientIP)

		c.Header("x-ratelimit-limit", fmt.Sprint(count))
		c.Header("x-ratelimit-reset", fmt.Sprint(time.Now().Add(time.Duration(24)*time.Hour).Unix()))

		if !exists {
			m.cache.Set(clientIP, 1, 86400) // 1 day TTL
			c.Header("x-ratelimit-remaining", fmt.Sprint(count-1))
			c.Header("x-ratelimit-used", fmt.Sprint(1))
			c.Next()
			return
		}

		used := value.(int) + 1

		if used > count {
			c.Header("x-ratelimit-remaining", fmt.Sprint(0))
			c.Header("x-ratelimit-used", fmt.Sprint(used))
			c.JSON(429, gin.H{
				"status":  429,
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		m.cache.Set(clientIP, used, 86400) // 1 day TTL

		c.Header("x-ratelimit-remaining", fmt.Sprint(count-used))
		c.Header("x-ratelimit-used", fmt.Sprint(used))
		c.Next()
	}
}

func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	cfConnectingIP := c.GetHeader("CF-Connecting-IP")

	if cfConnectingIP != "" {
		return cfConnectingIP
	}

	return c.ClientIP()
}
