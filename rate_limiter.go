package main

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync"
	"time"
)

type RateLimitConfig struct {
	RateLimitCount int
	TrustedProxies []string
}

type RateLimiter struct {
	config RateLimitConfig
	cache  *Cache
	mutex  sync.RWMutex
}

func NewRateLimiter(config RateLimitConfig, cache *Cache) *RateLimiter {
	return &RateLimiter{
		config: config,
		cache:  cache,
	}
}

func (rl *RateLimiter) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rl.mutex.Lock()
		defer rl.mutex.Unlock()

		clientIP := rl.getClientIP(r)

		if clientIP == "" {
			http.Error(w, "failed to determine client ip", http.StatusInternalServerError)
			return
		}

		value, exists := rl.cache.Get(clientIP)

		w.Header().Set("x-ratelimit-limit", fmt.Sprint(rl.config.RateLimitCount))
		w.Header().Set("x-ratelimit-reset", fmt.Sprint(time.Now().Add(24*time.Hour).Unix()))

		if !exists {
			rl.cache.Set(clientIP, 1, 43200) // 12 hours TTL
			w.Header().Set("x-ratelimit-remaining", fmt.Sprint(rl.config.RateLimitCount-1))
			w.Header().Set("x-ratelimit-used", fmt.Sprint(1))
			next.ServeHTTP(w, r)
			return
		}

		used := value.(int) + 1

		if used > rl.config.RateLimitCount {
			w.Header().Set("x-ratelimit-remaining", fmt.Sprint(0))
			w.Header().Set("x-ratelimit-used", fmt.Sprint(used))
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		rl.cache.Set(clientIP, used, 43200) // 12 hours TTL

		w.Header().Set("x-ratelimit-remaining", fmt.Sprint(rl.config.RateLimitCount-used))
		w.Header().Set("x-ratelimit-used", fmt.Sprint(used))
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) getClientIP(r *http.Request) string {
	cfConnectingIP := r.Header.Values("cf-connecting-ip")

	if len(cfConnectingIP) > 0 {
		return cfConnectingIP[0]
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return ""
	}

	if slices.Contains(rl.config.TrustedProxies, ip) {
		xForwardedFor := r.Header.Values("x-forwarded-for")

		if len(xForwardedFor) > 0 {
			return xForwardedFor[0]
		}
	}

	return ip
}
