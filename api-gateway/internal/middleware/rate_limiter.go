package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type RateLimiter struct {
	requests map[string]*clientLimiter
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type clientLimiter struct {
	count     int
	resetTime time.Time
}

func NewRateLimiter(requestsPerWindow int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*clientLimiter),
		limit:    requestsPerWindow,
		window:   window,
	}

	go rl.cleanup()
	return rl
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, limiter := range rl.requests {
			if now.After(limiter.resetTime) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		rl.mu.Lock()
		limiter, exists := rl.requests[clientIP]
		now := time.Now()

		if !exists || now.After(limiter.resetTime) {
			rl.requests[clientIP] = &clientLimiter{
				count:     1,
				resetTime: now.Add(rl.window),
			}
			rl.mu.Unlock()
			c.Next()
			return
		}

		if limiter.count >= rl.limit {
			rl.mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"retry_after": limiter.resetTime.Sub(now).Seconds(),
			})
			c.Abort()
			return
		}

		limiter.count++
		rl.mu.Unlock()
		c.Next()
	}
}
