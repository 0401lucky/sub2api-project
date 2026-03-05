package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	mu          sync.Mutex
	limiters    map[string]*ipLimiter
	rate        rate.Limit
	burst       int
	ttl         time.Duration
	lastCleanup time.Time
	keyFn       func(*gin.Context) string
}

func NewRateLimiter(r rate.Limit, burst int, keyFn func(*gin.Context) string) *RateLimiter {
	if keyFn == nil {
		keyFn = func(c *gin.Context) string {
			return strings.TrimSpace(c.ClientIP())
		}
	}
	return &RateLimiter{
		limiters:    make(map[string]*ipLimiter),
		rate:        r,
		burst:       burst,
		ttl:         30 * time.Minute,
		lastCleanup: time.Now(),
		keyFn:       keyFn,
	}
}

func NewIPRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return NewRateLimiter(r, burst, nil)
}

func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(l.keyFn(c))
		if key == "" {
			key = c.ClientIP()
		}
		limiter := l.getLimiter(key)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"code": http.StatusTooManyRequests, "message": "too many requests"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func (l *RateLimiter) getLimiter(key string) *rate.Limiter {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanupExpiredLocked(now)

	entry, ok := l.limiters[key]
	if !ok {
		entry = &ipLimiter{limiter: rate.NewLimiter(l.rate, l.burst)}
		l.limiters[key] = entry
	}
	entry.lastSeen = now
	return entry.limiter
}

func (l *RateLimiter) cleanupExpiredLocked(now time.Time) {
	if now.Sub(l.lastCleanup) < 10*time.Minute {
		return
	}
	cutoff := now.Add(-l.ttl)
	for key, entry := range l.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(l.limiters, key)
		}
	}
	l.lastCleanup = now
}
