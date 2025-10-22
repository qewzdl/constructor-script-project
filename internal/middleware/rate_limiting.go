package middleware

import (
	"constructor-script-backend/internal/config"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors   = make(map[string]*visitor)
	visitorsMu sync.RWMutex
)

func getVisitor(ip string, cfg *config.Config) *rate.Limiter {
	visitorsMu.Lock()
	defer visitorsMu.Unlock()

	if cfg.RateLimitRequests <= 0 {
		return nil
	}

	v, exists := visitors[ip]
	if !exists {
		windowSeconds := cfg.RateLimitWindow
		if windowSeconds <= 0 {
			windowSeconds = 60
		}

		limitPerSecond := float64(cfg.RateLimitRequests) / float64(windowSeconds)
		limit := rate.Limit(limitPerSecond)
		if limitPerSecond <= 0 {
			limit = rate.Inf
		}

		burst := cfg.RateLimitBurst
		if burst <= 0 {
			burst = cfg.RateLimitRequests
		}
		if burst < cfg.RateLimitRequests {
			burst = cfg.RateLimitRequests
		}

		limiter := rate.NewLimiter(limit, burst)

		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		visitorsMu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		visitorsMu.Unlock()
	}
}

func init() {
	go cleanupVisitors()
}

func RateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldBypassRateLimit(c.Request) {
			c.Next()
			return
		}

		limiter := getVisitor(c.ClientIP(), cfg)
		if limiter == nil {
			c.Next()
			return
		}
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests, please try again later",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

func shouldBypassRateLimit(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	path := r.URL.Path
	if path == "" {
		return false
	}

	staticPrefixes := []string{
		"/static/",
		"/uploads/",
	}

	for _, prefix := range staticPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	switch path {
	case "/favicon.ico":
		return true
	}

	return false
}
