package main

import (
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"
	"time"
)

type RateLimitConfig struct {
	RateLimitCount int
	TrustedProxies []string
}

type RateLimiter struct {
	config RateLimitConfig
	caches struct {
		ratelimit *CacheStore[int]
	}
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		config: config,
	}

	ratelimitCache := NewCacheStore[int](0)
	rl.caches.ratelimit = ratelimitCache

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.caches.ratelimit.Sweep()
		}
	}()

	return rl
}

func (rl *RateLimiter) limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := rl.getClientIP(r)
		if clientIP == "" {
			http.Error(w, "failed to determine client ip", http.StatusInternalServerError)
			return
		}

		var used int
		rl.caches.ratelimit.WithLock(func(actions CacheStoreActions[int]) {
			current, exists := actions.Get(clientIP)
			if !exists {
				actions.Set(clientIP, 1, 12*time.Hour)
				used = 1
				return
			}
			current++
			used = current
			if current > rl.config.RateLimitCount {
				return
			}
			actions.Update(clientIP, current, 0)
		})

		w.Header().Set("x-ratelimit-limit", fmt.Sprint(rl.config.RateLimitCount))
		w.Header().Set("x-ratelimit-used", fmt.Sprint(used))

		if used > rl.config.RateLimitCount {
			w.Header().Set("x-ratelimit-remaining", "0")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		w.Header().Set("x-ratelimit-remaining", fmt.Sprint(rl.config.RateLimitCount-used))
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
		xForwardedFor := r.Header.Get("x-forwarded-for")

		if xForwardedFor != "" {
			firstIp := strings.SplitN(xForwardedFor, ",", 2)[0]
			if firstIp != "" {
				return firstIp
			}
		}
	}

	return ip
}
