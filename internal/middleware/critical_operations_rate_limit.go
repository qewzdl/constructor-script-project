package middleware

import (
	"constructor-script-backend/internal/config"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// criticalOperationVisitor tracks rate limit state for critical operations
type criticalOperationVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	// uploadLimiters tracks rate limits per IP for upload operations
	uploadLimiters   = make(map[string]*criticalOperationVisitor)
	uploadLimitersMu sync.RWMutex

	// backupLimiters tracks rate limits per IP for backup operations
	backupLimiters   = make(map[string]*criticalOperationVisitor)
	backupLimitersMu sync.RWMutex
)

func getCriticalOperationLimiter(ip string, limiters map[string]*criticalOperationVisitor, mu *sync.RWMutex, requestsPerWindow int, windowSeconds int) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	if requestsPerWindow <= 0 {
		return nil
	}

	v, exists := limiters[ip]
	if !exists {
		if windowSeconds <= 0 {
			windowSeconds = 60
		}

		limitPerSecond := float64(requestsPerWindow) / float64(windowSeconds)
		limit := rate.Limit(limitPerSecond)
		if limitPerSecond <= 0 {
			limit = rate.Inf
		}

		limiter := rate.NewLimiter(limit, requestsPerWindow)
		limiters[ip] = &criticalOperationVisitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func cleanupCriticalOperationVisitors() {
	for {
		time.Sleep(5 * time.Minute)

		uploadLimitersMu.Lock()
		for ip, v := range uploadLimiters {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(uploadLimiters, ip)
			}
		}
		uploadLimitersMu.Unlock()

		backupLimitersMu.Lock()
		for ip, v := range backupLimiters {
			if time.Since(v.lastSeen) > 10*time.Minute {
				delete(backupLimiters, ip)
			}
		}
		backupLimitersMu.Unlock()
	}
}

func init() {
	go cleanupCriticalOperationVisitors()
}

// UploadRateLimitMiddleware limits file upload operations per IP
// Default: 10 requests per 300 seconds (5 minutes)
func UploadRateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	requestsPerWindow := cfg.UploadRateLimitRequests
	if requestsPerWindow <= 0 {
		requestsPerWindow = 10
	}
	windowSeconds := cfg.UploadRateLimitWindow
	if windowSeconds <= 0 {
		windowSeconds = 300
	}

	return func(c *gin.Context) {
		limiter := getCriticalOperationLimiter(c.ClientIP(), uploadLimiters, &uploadLimitersMu, requestsPerWindow, windowSeconds)
		if limiter == nil {
			c.Next()
			return
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":          "upload rate limit exceeded",
				"message":        "Too many upload requests. Please try again later.",
				"retry_after":    int(windowSeconds),
				"max_requests":   requestsPerWindow,
				"window_seconds": windowSeconds,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// BackupRateLimitMiddleware limits backup import/export operations per IP
// Default: 5 requests per 3600 seconds (1 hour)
func BackupRateLimitMiddleware(cfg *config.Config) gin.HandlerFunc {
	requestsPerWindow := cfg.BackupRateLimitRequests
	if requestsPerWindow <= 0 {
		requestsPerWindow = 5
	}
	windowSeconds := cfg.BackupRateLimitWindow
	if windowSeconds <= 0 {
		windowSeconds = 3600
	}

	return func(c *gin.Context) {
		limiter := getCriticalOperationLimiter(c.ClientIP(), backupLimiters, &backupLimitersMu, requestsPerWindow, windowSeconds)
		if limiter == nil {
			c.Next()
			return
		}

		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":          "backup rate limit exceeded",
				"message":        "Too many backup requests. Please try again later.",
				"retry_after":    int(windowSeconds),
				"max_requests":   requestsPerWindow,
				"window_seconds": windowSeconds,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
