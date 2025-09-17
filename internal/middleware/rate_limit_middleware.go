package middleware

import (
	"context"
	"errors"
	"sync"
	"time"
	"tinyauth-analytics/internal/model"

	"fmt"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RateLimitMiddleware struct {
	database *gorm.DB
	mutex    sync.RWMutex
}

func NewRateLimitMiddleware(database *gorm.DB) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		database: database,
	}
}

func (m *RateLimitMiddleware) Init() error {
	return nil
}

func (m *RateLimitMiddleware) Middleware(count int64) gin.HandlerFunc {
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

		ctx := context.Background()

		entry, err := gorm.G[model.RateLimit](m.database).First(ctx)

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(500, gin.H{
				"status":  500,
				"message": "Database error",
			})
			c.Abort()
			return
		}

		c.Header("x-ratelimit-limit", fmt.Sprint(count))

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Header("x-ratelimit-remaining", fmt.Sprint(count-1))
			c.Header("x-ratelimit-used", fmt.Sprint(1))
			ctx := context.Background()
			err := gorm.G[model.RateLimit](m.database).Create(ctx, &model.RateLimit{
				Count: 1,
				IP:    clientIP,
			})
			if err != nil {
				c.JSON(500, gin.H{
					"status":  500,
					"message": "Database error",
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		if entry.Expiry > 0 && entry.Expiry < time.Now().UnixMilli() {
			entry.Count = 0
			entry.Expiry = 0
		}

		entry.Count++

		if entry.Count > count {
			expiry := time.Now().Add(time.Duration(12) * time.Hour).UnixMilli()
			_, err := gorm.G[model.RateLimit](m.database).Where("id = ?", entry.ID).Update(ctx, "expiry", fmt.Sprint(expiry))
			if err != nil {
				c.JSON(500, gin.H{
					"status":  500,
					"message": "Database error",
				})
				c.Abort()
				return
			}
			c.Header("x-ratelimit-remaining", "0")
			c.Header("x-ratelimit-used", fmt.Sprint(count))
			c.Header("x-ratelimit-reset", fmt.Sprint(expiry))
			c.JSON(429, gin.H{
				"status":  429,
				"message": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		_, err = gorm.G[model.RateLimit](m.database).Where("id = ?", entry.ID).Update(ctx, "count", entry.Count)

		if err != nil {
			c.JSON(500, gin.H{
				"status":  500,
				"message": "Database error",
			})
			c.Abort()
			return
		}

		c.Header("x-ratelimit-remaining", fmt.Sprint(count-entry.Count))
		c.Header("x-ratelimit-used", fmt.Sprint(entry.Count))
		c.Next()
	}
}

func (m *RateLimitMiddleware) getClientIP(c *gin.Context) string {
	cfConnectingIP := c.GetHeader("CF-Connecting-IP")

	if cfConnectingIP != "" {
		return cfConnectingIP
	}

	clientIP := c.ClientIP()
	remoteIP := c.RemoteIP()

	// If we are using a proxy like Cloudflare we don't want to rame limit the proxy's IP
	if clientIP == remoteIP {
		return ""
	}

	return clientIP
}
