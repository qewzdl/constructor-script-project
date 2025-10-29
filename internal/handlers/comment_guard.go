package handlers

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/time/rate"

	"constructor-script-backend/internal/config"
	"constructor-script-backend/internal/models"
)

var (
	// ErrCommentRateLimited is returned when the caller exceeds the configured submission rate.
	ErrCommentRateLimited = errors.New("comment rate limit reached")

	// ErrCommentContentInvalid is returned when the submitted comment fails content validation checks.
	ErrCommentContentInvalid = errors.New("comment content failed validation")
)

// CommentGuardDecision describes the result of evaluating a comment submission.
type CommentGuardDecision struct {
	// Err contains the validation or throttling error that was encountered. When nil, the submission is allowed.
	Err error

	// RetryAfter communicates how long a client should wait before retrying after a throttling rejection.
	RetryAfter time.Duration
}

type commentRateMode int

const (
	commentRateModeRegular commentRateMode = iota
	commentRateModeNewUser
)

type rateSettings struct {
	requests int
	window   time.Duration
}

type userLimiter struct {
	limiter  *rate.Limiter
	mode     commentRateMode
	lastSeen time.Time
}

const (
	limiterCleanupInterval = 5 * time.Minute
	limiterIdleTTL         = 30 * time.Minute
	maxRepeatedCharacters  = 12
)

// CommentGuard encapsulates throttling and content validation for comment submissions.
type CommentGuard struct {
	cfg *config.Config

	mu          sync.Mutex
	limiters    map[uint]*userLimiter
	lastCleanup time.Time
}

// NewCommentGuard constructs a CommentGuard using the provided configuration.
func NewCommentGuard(cfg *config.Config) *CommentGuard {
	return &CommentGuard{
		cfg:      cfg,
		limiters: make(map[uint]*userLimiter),
	}
}

// Evaluate verifies that the given user may submit the provided comment content.
// It returns a decision that contains an error when the submission should be rejected.
func (g *CommentGuard) Evaluate(user *models.User, content string) CommentGuardDecision {
	if g == nil {
		return CommentGuardDecision{}
	}

	if reason := g.validateContent(content); reason != "" {
		return CommentGuardDecision{Err: fmt.Errorf("%w: %s", ErrCommentContentInvalid, reason)}
	}

	limiter := g.getLimiter(user)
	if limiter == nil {
		return CommentGuardDecision{}
	}

	reserve := limiter.Reserve()
	if !reserve.OK() {
		reserve.Cancel()
		return CommentGuardDecision{
			Err:        fmt.Errorf("%w: too many comments submitted", ErrCommentRateLimited),
			RetryAfter: g.rateWindowFor(user),
		}
	}

	delay := reserve.Delay()
	if delay > 0 {
		reserve.Cancel()
		return CommentGuardDecision{
			Err:        fmt.Errorf("%w: please wait %s before commenting again", ErrCommentRateLimited, humanizeDuration(delay)),
			RetryAfter: delay,
		}
	}

	return CommentGuardDecision{}
}

func (g *CommentGuard) validateContent(content string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "comment cannot be empty"
	}

	minLength := 3
	if g != nil && g.cfg != nil && g.cfg.CommentMinContentLength > minLength {
		minLength = g.cfg.CommentMinContentLength
	}

	if utf8.RuneCountInString(trimmed) < minLength {
		return fmt.Sprintf("comment must be at least %d characters long", minLength)
	}

	if hasExcessiveRepetition(trimmed, maxRepeatedCharacters) {
		return "comment contains excessive repeated characters"
	}

	maxLinks := -1
	if g != nil && g.cfg != nil {
		maxLinks = g.cfg.CommentMaxLinks
	}
	if maxLinks >= 0 {
		linkCount := countLinks(trimmed)
		if linkCount > maxLinks {
			return fmt.Sprintf("comment contains too many links (%d > %d)", linkCount, maxLinks)
		}
	}

	return ""
}

func (g *CommentGuard) getLimiter(user *models.User) *rate.Limiter {
	mode := g.modeForUser(user)
	settings := g.settingsForMode(mode)
	if settings.requests <= 0 || settings.window <= 0 {
		return nil
	}

	userID := uint(0)
	if user != nil {
		userID = user.ID
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.maybeCleanupLocked()

	if g.limiters == nil {
		g.limiters = make(map[uint]*userLimiter)
	}

	if limiter, ok := g.limiters[userID]; ok {
		if limiter != nil && limiter.mode == mode {
			limiter.lastSeen = time.Now()
			return limiter.limiter
		}
	}

	limit := rate.Limit(float64(settings.requests) / settings.window.Seconds())
	if settings.requests == 1 {
		limit = rate.Every(settings.window)
	}
	newLimiter := rate.NewLimiter(limit, settings.requests)
	g.limiters[userID] = &userLimiter{
		limiter:  newLimiter,
		mode:     mode,
		lastSeen: time.Now(),
	}

	return newLimiter
}

func (g *CommentGuard) settingsForMode(mode commentRateMode) rateSettings {
	if g == nil || g.cfg == nil {
		return rateSettings{}
	}

	switch mode {
	case commentRateModeNewUser:
		requests := g.cfg.CommentNewUserRateLimitRequests
		if requests <= 0 {
			requests = g.cfg.CommentRateLimitRequests
		}
		windowSeconds := g.cfg.CommentNewUserRateLimitWindow
		if windowSeconds <= 0 {
			windowSeconds = g.cfg.CommentRateLimitWindow
		}
		return buildRateSettings(requests, windowSeconds)
	default:
		return buildRateSettings(g.cfg.CommentRateLimitRequests, g.cfg.CommentRateLimitWindow)
	}
}

func buildRateSettings(requests, windowSeconds int) rateSettings {
	if requests <= 0 || windowSeconds <= 0 {
		return rateSettings{}
	}

	return rateSettings{
		requests: requests,
		window:   time.Duration(windowSeconds) * time.Second,
	}
}

func (g *CommentGuard) modeForUser(user *models.User) commentRateMode {
	if g == nil || g.cfg == nil || user == nil {
		return commentRateModeRegular
	}

	ageThreshold := time.Duration(g.cfg.CommentNewUserAgeHours) * time.Hour
	if g.cfg.CommentNewUserAgeHours <= 0 {
		return commentRateModeRegular
	}

	if time.Since(user.CreatedAt) < ageThreshold {
		return commentRateModeNewUser
	}

	return commentRateModeRegular
}

func (g *CommentGuard) rateWindowFor(user *models.User) time.Duration {
	settings := g.settingsForMode(g.modeForUser(user))
	return settings.window
}

func (g *CommentGuard) maybeCleanupLocked() {
	if time.Since(g.lastCleanup) < limiterCleanupInterval {
		return
	}

	cutoff := time.Now().Add(-limiterIdleTTL)
	for userID, limiter := range g.limiters {
		if limiter == nil || limiter.lastSeen.Before(cutoff) {
			delete(g.limiters, userID)
		}
	}

	g.lastCleanup = time.Now()
}

func humanizeDuration(d time.Duration) string {
	if d <= 0 {
		return "moment"
	}

	if d < time.Second {
		return "less than a second"
	}

	if d < time.Minute {
		seconds := int(math.Ceil(d.Seconds()))
		if seconds == 1 {
			return "1 second"
		}
		return fmt.Sprintf("%d seconds", seconds)
	}

	minutes := int(math.Ceil(d.Minutes()))
	if minutes == 1 {
		return "1 minute"
	}
	return fmt.Sprintf("%d minutes", minutes)
}

func hasExcessiveRepetition(s string, threshold int) bool {
	if threshold <= 0 {
		return false
	}

	var (
		lastRune rune
		count    int
	)

	for i, r := range s {
		if i == 0 || r != lastRune {
			lastRune = r
			count = 1
			continue
		}

		count++
		if count >= threshold {
			return true
		}
	}

	return false
}

func countLinks(s string) int {
	lower := strings.ToLower(s)
	return strings.Count(lower, "http://") + strings.Count(lower, "https://")
}
