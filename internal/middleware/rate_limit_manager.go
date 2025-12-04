package middleware

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimitManager manages rate limiters with lifecycle control
type RateLimitManager struct {
	visitors         map[string]*visitor
	visitorsMu       sync.RWMutex
	uploadLimiters   map[string]*criticalOperationVisitor
	uploadLimitersMu sync.RWMutex
	backupLimiters   map[string]*criticalOperationVisitor
	backupLimitersMu sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewRateLimitManager creates a new rate limit manager with context-based lifecycle
func NewRateLimitManager(ctx context.Context) *RateLimitManager {
	managerCtx, cancel := context.WithCancel(ctx)

	m := &RateLimitManager{
		visitors:       make(map[string]*visitor),
		uploadLimiters: make(map[string]*criticalOperationVisitor),
		backupLimiters: make(map[string]*criticalOperationVisitor),
		ctx:            managerCtx,
		cancel:         cancel,
	}

	m.wg.Add(1)
	go m.cleanupLoop()

	return m
}

// GetVisitor retrieves or creates a rate limiter for the given IP
func (m *RateLimitManager) GetVisitor(ip string, requestsPerWindow int, windowSeconds int, burst int) *rate.Limiter {
	m.visitorsMu.Lock()
	defer m.visitorsMu.Unlock()

	if requestsPerWindow <= 0 {
		return nil
	}

	v, exists := m.visitors[ip]
	if !exists {
		if windowSeconds <= 0 {
			windowSeconds = 60
		}

		limitPerSecond := float64(requestsPerWindow) / float64(windowSeconds)
		limit := rate.Limit(limitPerSecond)
		if limitPerSecond <= 0 {
			limit = rate.Inf
		}

		if burst <= 0 {
			burst = requestsPerWindow
		}
		if burst < requestsPerWindow {
			burst = requestsPerWindow
		}

		limiter := rate.NewLimiter(limit, burst)
		m.visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	v.lastSeen = time.Now()
	return v.limiter
}

// GetCriticalOperationLimiter retrieves or creates a rate limiter for critical operations
func (m *RateLimitManager) GetCriticalOperationLimiter(ip string, operationType string, requestsPerWindow int, windowSeconds int) *rate.Limiter {
	var limiters map[string]*criticalOperationVisitor
	var mu *sync.RWMutex

	switch operationType {
	case "upload":
		limiters = m.uploadLimiters
		mu = &m.uploadLimitersMu
	case "backup":
		limiters = m.backupLimiters
		mu = &m.backupLimitersMu
	default:
		return nil
	}

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

// cleanupLoop periodically removes inactive rate limiters
func (m *RateLimitManager) cleanupLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup removes inactive rate limiters
func (m *RateLimitManager) cleanup() {
	// Cleanup general visitors (3 minute threshold)
	m.visitorsMu.Lock()
	for ip, v := range m.visitors {
		if time.Since(v.lastSeen) > 3*time.Minute {
			delete(m.visitors, ip)
		}
	}
	m.visitorsMu.Unlock()

	// Cleanup upload limiters (10 minute threshold)
	m.uploadLimitersMu.Lock()
	for ip, v := range m.uploadLimiters {
		if time.Since(v.lastSeen) > 10*time.Minute {
			delete(m.uploadLimiters, ip)
		}
	}
	m.uploadLimitersMu.Unlock()

	// Cleanup backup limiters (10 minute threshold)
	m.backupLimitersMu.Lock()
	for ip, v := range m.backupLimiters {
		if time.Since(v.lastSeen) > 10*time.Minute {
			delete(m.backupLimiters, ip)
		}
	}
	m.backupLimitersMu.Unlock()
}

// Shutdown stops the cleanup goroutine and waits for it to finish
func (m *RateLimitManager) Shutdown() error {
	m.cancel()
	m.wg.Wait()
	return nil
}
