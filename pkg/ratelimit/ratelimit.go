// Package ratelimit provides rate limiting infrastructure for DoS prevention.
// It implements a token bucket algorithm with configurable limits, exponential backoff,
// and telemetry hooks for monitoring rate limit events across GitHub API calls,
// MCP server requests, and workflow compilation operations.
package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var log = logger.New("ratelimit:limiter")

// Common errors returned by the rate limiter
var (
	// ErrRateLimitExceeded is returned when a request exceeds the rate limit
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	// ErrContextCanceled is returned when the context is canceled while waiting
	ErrContextCanceled = errors.New("context canceled while waiting for rate limit")
	// ErrInvalidConfig is returned when the rate limiter configuration is invalid
	ErrInvalidConfig = errors.New("invalid rate limiter configuration")
)

// OperationType represents different types of operations that can be rate limited
type OperationType string

const (
	// OperationGitHubAPI represents GitHub API operations
	OperationGitHubAPI OperationType = "github-api"
	// OperationMCPRequest represents MCP server request operations
	OperationMCPRequest OperationType = "mcp-request"
	// OperationWorkflowCompile represents workflow compilation operations
	OperationWorkflowCompile OperationType = "workflow-compile"
	// OperationMCPRegistry represents MCP registry API operations
	OperationMCPRegistry OperationType = "mcp-registry"
	// OperationFileRead represents file read operations
	OperationFileRead OperationType = "file-read"
	// OperationNetworkRequest represents generic network request operations
	OperationNetworkRequest OperationType = "network-request"
)

// Config holds configuration for rate limiting
type Config struct {
	// Rate is the number of tokens added per interval
	Rate float64
	// Burst is the maximum number of tokens the bucket can hold
	Burst int
	// Interval is the duration between token additions
	Interval time.Duration
	// MaxRetries is the maximum number of retry attempts on rate limit errors
	MaxRetries int
	// InitialBackoff is the initial backoff duration for exponential backoff
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff
	BackoffMultiplier float64
}

// DefaultConfigs provides sensible default configurations for different operation types
var DefaultConfigs = map[OperationType]Config{
	OperationGitHubAPI: {
		Rate:              100,
		Burst:             100,
		Interval:          time.Hour,
		MaxRetries:        3,
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Minute * 5,
		BackoffMultiplier: 2.0,
	},
	OperationMCPRequest: {
		Rate:              50,
		Burst:             50,
		Interval:          time.Minute,
		MaxRetries:        3,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        time.Minute,
		BackoffMultiplier: 2.0,
	},
	OperationWorkflowCompile: {
		Rate:              100,
		Burst:             100,
		Interval:          time.Minute,
		MaxRetries:        2,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        time.Second * 10,
		BackoffMultiplier: 2.0,
	},
	OperationMCPRegistry: {
		Rate:              30,
		Burst:             30,
		Interval:          time.Minute,
		MaxRetries:        3,
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Minute * 2,
		BackoffMultiplier: 2.0,
	},
	OperationFileRead: {
		Rate:              1000,
		Burst:             1000,
		Interval:          time.Minute,
		MaxRetries:        1,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 2.0,
	},
	OperationNetworkRequest: {
		Rate:              60,
		Burst:             60,
		Interval:          time.Minute,
		MaxRetries:        3,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        time.Minute,
		BackoffMultiplier: 2.0,
	},
}

// Stats holds statistics about rate limiter usage
type Stats struct {
	mu                sync.RWMutex
	AllowedRequests   int64
	DeniedRequests    int64
	WaitingRequests   int64
	TotalWaitTime     time.Duration
	RetryAttempts     int64
	SuccessfulRetries int64
	FailedRetries     int64
}

// Clone returns a copy of the stats
func (s *Stats) Clone() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Stats{
		AllowedRequests:   s.AllowedRequests,
		DeniedRequests:    s.DeniedRequests,
		WaitingRequests:   s.WaitingRequests,
		TotalWaitTime:     s.TotalWaitTime,
		RetryAttempts:     s.RetryAttempts,
		SuccessfulRetries: s.SuccessfulRetries,
		FailedRetries:     s.FailedRetries,
	}
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	mu            sync.Mutex
	config        Config
	operationType OperationType
	tokens        float64
	lastRefill    time.Time
	stats         Stats
}

// NewTokenBucket creates a new token bucket rate limiter for the given operation type
func NewTokenBucket(opType OperationType, config *Config) (*TokenBucket, error) {
	cfg := DefaultConfigs[opType]
	if config != nil {
		cfg = *config
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	log.Printf("Creating token bucket: operation=%s, rate=%.2f, burst=%d, interval=%v",
		opType, cfg.Rate, cfg.Burst, cfg.Interval)

	return &TokenBucket{
		config:        cfg,
		operationType: opType,
		tokens:        float64(cfg.Burst),
		lastRefill:    time.Now(),
	}, nil
}

// validateConfig validates a rate limiter configuration
func validateConfig(cfg Config) error {
	if cfg.Rate <= 0 {
		return fmt.Errorf("rate must be positive, got %.2f", cfg.Rate)
	}
	if cfg.Burst <= 0 {
		return fmt.Errorf("burst must be positive, got %d", cfg.Burst)
	}
	if cfg.Interval <= 0 {
		return fmt.Errorf("interval must be positive, got %v", cfg.Interval)
	}
	if cfg.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got %d", cfg.MaxRetries)
	}
	if cfg.BackoffMultiplier < 1.0 {
		return fmt.Errorf("backoff multiplier must be >= 1.0, got %.2f", cfg.BackoffMultiplier)
	}
	return nil
}

// refill adds tokens to the bucket based on elapsed time
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Calculate tokens to add based on elapsed time
	tokensToAdd := (elapsed.Seconds() / tb.config.Interval.Seconds()) * tb.config.Rate
	tb.tokens = math.Min(float64(tb.config.Burst), tb.tokens+tokensToAdd)
	tb.lastRefill = now
}

// Allow checks if a request is allowed and consumes a token if so
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	if tb.tokens >= 1 {
		tb.tokens--
		tb.stats.mu.Lock()
		tb.stats.AllowedRequests++
		tb.stats.mu.Unlock()
		log.Printf("Request allowed: operation=%s, remaining_tokens=%.2f", tb.operationType, tb.tokens)
		return true
	}

	tb.stats.mu.Lock()
	tb.stats.DeniedRequests++
	tb.stats.mu.Unlock()
	log.Printf("Request denied: operation=%s, tokens=%.2f", tb.operationType, tb.tokens)
	return false
}

// Wait blocks until a token is available or the context is canceled
func (tb *TokenBucket) Wait(ctx context.Context) error {
	tb.stats.mu.Lock()
	tb.stats.WaitingRequests++
	tb.stats.mu.Unlock()
	defer func() {
		tb.stats.mu.Lock()
		tb.stats.WaitingRequests--
		tb.stats.mu.Unlock()
	}()

	startWait := time.Now()

	for {
		select {
		case <-ctx.Done():
			return ErrContextCanceled
		default:
			if tb.Allow() {
				waitDuration := time.Since(startWait)
				tb.stats.mu.Lock()
				tb.stats.TotalWaitTime += waitDuration
				tb.stats.mu.Unlock()
				if waitDuration > time.Millisecond {
					log.Printf("Request allowed after wait: operation=%s, wait_time=%v", tb.operationType, waitDuration)
				}
				return nil
			}

			// Calculate wait time until next token
			waitTime := tb.timeUntilNextToken()
			if waitTime > 0 {
				select {
				case <-ctx.Done():
					return ErrContextCanceled
				case <-time.After(waitTime):
					// Continue to try again
				}
			}
		}
	}
}

// timeUntilNextToken calculates the duration until the next token is available
func (tb *TokenBucket) timeUntilNextToken() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	if tb.tokens >= 1 {
		return 0
	}

	// Calculate time needed to refill one token
	tokensNeeded := 1.0 - tb.tokens
	secondsNeeded := (tokensNeeded / tb.config.Rate) * tb.config.Interval.Seconds()
	return time.Duration(secondsNeeded * float64(time.Second))
}

// Reserve reserves a token for future use, returning a Reservation
func (tb *TokenBucket) Reserve() *Reservation {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.refill()

	delay := time.Duration(0)
	if tb.tokens < 1 {
		tokensNeeded := 1.0 - tb.tokens
		secondsNeeded := (tokensNeeded / tb.config.Rate) * tb.config.Interval.Seconds()
		delay = time.Duration(secondsNeeded * float64(time.Second))
	}

	// Always consume the token (allowing "debt")
	tb.tokens--

	return &Reservation{
		bucket: tb,
		delay:  delay,
		ok:     delay == 0,
	}
}

// Reservation represents a reserved token from the rate limiter
type Reservation struct {
	bucket *TokenBucket
	delay  time.Duration
	ok     bool
}

// OK returns whether the reservation is immediately available
func (r *Reservation) OK() bool {
	return r.ok
}

// Delay returns the duration to wait before the reservation is ready
func (r *Reservation) Delay() time.Duration {
	return r.delay
}

// Cancel cancels the reservation, returning the token to the bucket
func (r *Reservation) Cancel() {
	r.bucket.mu.Lock()
	defer r.bucket.mu.Unlock()
	r.bucket.tokens = math.Min(float64(r.bucket.config.Burst), r.bucket.tokens+1)
}

// Tokens returns the current number of available tokens
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// Stats returns a copy of the rate limiter statistics
func (tb *TokenBucket) Stats() Stats {
	return tb.stats.Clone()
}

// Config returns the rate limiter configuration
func (tb *TokenBucket) Config() Config {
	return tb.config
}

// OperationType returns the operation type this limiter is for
func (tb *TokenBucket) OperationType() OperationType {
	return tb.operationType
}

// Backoff calculates the backoff duration for a given retry attempt
func (tb *TokenBucket) Backoff(attempt int) time.Duration {
	if attempt <= 0 {
		return tb.config.InitialBackoff
	}

	backoff := float64(tb.config.InitialBackoff) * math.Pow(tb.config.BackoffMultiplier, float64(attempt))
	if backoff > float64(tb.config.MaxBackoff) {
		return tb.config.MaxBackoff
	}
	return time.Duration(backoff)
}

// ExecuteWithRetry executes a function with exponential backoff on rate limit errors
func (tb *TokenBucket) ExecuteWithRetry(ctx context.Context, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= tb.config.MaxRetries; attempt++ {
		// Wait for rate limit
		if err := tb.Wait(ctx); err != nil {
			return err
		}

		// Execute the function
		if err := fn(); err != nil {
			lastErr = err

			// Check if it's a rate limit error (indicated by specific error types)
			if errors.Is(err, ErrRateLimitExceeded) || isRateLimitError(err) {
				tb.stats.mu.Lock()
				tb.stats.RetryAttempts++
				tb.stats.mu.Unlock()

				if attempt < tb.config.MaxRetries {
					backoff := tb.Backoff(attempt)
					log.Printf("Rate limit error, backing off: operation=%s, attempt=%d, backoff=%v, error=%v",
						tb.operationType, attempt+1, backoff, err)

					select {
					case <-ctx.Done():
						return ErrContextCanceled
					case <-time.After(backoff):
						continue
					}
				}

				tb.stats.mu.Lock()
				tb.stats.FailedRetries++
				tb.stats.mu.Unlock()
				return fmt.Errorf("rate limit exceeded after %d retries: %w", attempt+1, err)
			}

			// Non-rate-limit error, return immediately
			return err
		}

		// Success
		if attempt > 0 {
			tb.stats.mu.Lock()
			tb.stats.SuccessfulRetries++
			tb.stats.mu.Unlock()
			log.Printf("Request succeeded after retry: operation=%s, attempt=%d", tb.operationType, attempt+1)
		}
		return nil
	}

	tb.stats.mu.Lock()
	tb.stats.FailedRetries++
	tb.stats.mu.Unlock()
	return lastErr
}

// isRateLimitError checks if an error is a rate limit error based on common patterns
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	rateLimitPatterns := []string{
		"rate limit",
		"429",
		"too many requests",
		"exceeded",
		"throttl",
	}

	for _, pattern := range rateLimitPatterns {
		if containsIgnoreCase(errStr, pattern) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	return containsLower(toLower(s), toLower(substr))
}

// toLower converts a string to lowercase (simple ASCII version)
func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		result[i] = c
	}
	return string(result)
}

// containsLower checks if s contains substr (both assumed to be lowercase)
func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RateLimiterGroup manages multiple rate limiters for different operation types
type RateLimiterGroup struct {
	mu       sync.RWMutex
	limiters map[OperationType]*TokenBucket
}

// NewRateLimiterGroup creates a new rate limiter group
func NewRateLimiterGroup() *RateLimiterGroup {
	return &RateLimiterGroup{
		limiters: make(map[OperationType]*TokenBucket),
	}
}

// GetOrCreate gets an existing rate limiter or creates a new one with default config
func (g *RateLimiterGroup) GetOrCreate(opType OperationType) (*TokenBucket, error) {
	g.mu.RLock()
	limiter, exists := g.limiters[opType]
	g.mu.RUnlock()

	if exists {
		return limiter, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	// Double-check after acquiring write lock
	if limiter, exists = g.limiters[opType]; exists {
		return limiter, nil
	}

	limiter, err := NewTokenBucket(opType, nil)
	if err != nil {
		return nil, err
	}
	g.limiters[opType] = limiter
	return limiter, nil
}

// GetOrCreateWithConfig gets an existing rate limiter or creates a new one with custom config
func (g *RateLimiterGroup) GetOrCreateWithConfig(opType OperationType, config *Config) (*TokenBucket, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if limiter, exists := g.limiters[opType]; exists {
		return limiter, nil
	}

	limiter, err := NewTokenBucket(opType, config)
	if err != nil {
		return nil, err
	}
	g.limiters[opType] = limiter
	return limiter, nil
}

// AllStats returns statistics for all rate limiters in the group
func (g *RateLimiterGroup) AllStats() map[OperationType]Stats {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[OperationType]Stats)
	for opType, limiter := range g.limiters {
		result[opType] = limiter.Stats()
	}
	return result
}

// ParseRateLimitSpec parses a rate limit specification string (e.g., "100/hour", "50/minute")
func ParseRateLimitSpec(spec string) (*Config, error) {
	if spec == "" {
		return nil, fmt.Errorf("empty rate limit specification")
	}

	var rate float64
	var unit string
	_, err := fmt.Sscanf(spec, "%f/%s", &rate, &unit)
	if err != nil {
		return nil, fmt.Errorf("invalid rate limit format: %s (expected format: N/unit)", spec)
	}

	if rate <= 0 {
		return nil, fmt.Errorf("rate must be positive, got %.2f", rate)
	}

	var interval time.Duration
	switch unit {
	case "second", "sec", "s":
		interval = time.Second
	case "minute", "min", "m":
		interval = time.Minute
	case "hour", "hr", "h":
		interval = time.Hour
	case "day", "d":
		interval = 24 * time.Hour
	default:
		return nil, fmt.Errorf("unknown time unit: %s (expected: second, minute, hour, day)", unit)
	}

	return &Config{
		Rate:              rate,
		Burst:             int(rate),
		Interval:          interval,
		MaxRetries:        3,
		InitialBackoff:    time.Second,
		MaxBackoff:        time.Minute * 5,
		BackoffMultiplier: 2.0,
	}, nil
}

// DefaultGroup is a global rate limiter group for shared use
var DefaultGroup = NewRateLimiterGroup()

// Allow is a convenience function to check if a request is allowed using the default group
func Allow(opType OperationType) bool {
	limiter, err := DefaultGroup.GetOrCreate(opType)
	if err != nil {
		log.Printf("Failed to get rate limiter: %v", err)
		return true // Fail open if we can't create the limiter
	}
	return limiter.Allow()
}

// Wait is a convenience function to wait for a token using the default group
func Wait(ctx context.Context, opType OperationType) error {
	limiter, err := DefaultGroup.GetOrCreate(opType)
	if err != nil {
		log.Printf("Failed to get rate limiter: %v", err)
		return nil // Fail open if we can't create the limiter
	}
	return limiter.Wait(ctx)
}

// ExecuteWithRetry is a convenience function to execute with retry using the default group
func ExecuteWithRetry(ctx context.Context, opType OperationType, fn func() error) error {
	limiter, err := DefaultGroup.GetOrCreate(opType)
	if err != nil {
		log.Printf("Failed to get rate limiter: %v", err)
		return fn() // Fall back to executing without rate limiting
	}
	return limiter.ExecuteWithRetry(ctx, fn)
}
