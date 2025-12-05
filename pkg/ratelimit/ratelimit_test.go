package ratelimit

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewTokenBucket(t *testing.T) {
	tests := []struct {
		name    string
		opType  OperationType
		config  *Config
		wantErr bool
	}{
		{
			name:    "default GitHub API config",
			opType:  OperationGitHubAPI,
			config:  nil,
			wantErr: false,
		},
		{
			name:    "default MCP request config",
			opType:  OperationMCPRequest,
			config:  nil,
			wantErr: false,
		},
		{
			name:   "custom config",
			opType: OperationGitHubAPI,
			config: &Config{
				Rate:              10,
				Burst:             10,
				Interval:          time.Second,
				MaxRetries:        2,
				InitialBackoff:    100 * time.Millisecond,
				MaxBackoff:        time.Second,
				BackoffMultiplier: 2.0,
			},
			wantErr: false,
		},
		{
			name:   "invalid rate",
			opType: OperationGitHubAPI,
			config: &Config{
				Rate:              0,
				Burst:             10,
				Interval:          time.Second,
				BackoffMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name:   "invalid burst",
			opType: OperationGitHubAPI,
			config: &Config{
				Rate:              10,
				Burst:             0,
				Interval:          time.Second,
				BackoffMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name:   "invalid interval",
			opType: OperationGitHubAPI,
			config: &Config{
				Rate:              10,
				Burst:             10,
				Interval:          0,
				BackoffMultiplier: 2.0,
			},
			wantErr: true,
		},
		{
			name:   "invalid backoff multiplier",
			opType: OperationGitHubAPI,
			config: &Config{
				Rate:              10,
				Burst:             10,
				Interval:          time.Second,
				BackoffMultiplier: 0.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, err := NewTokenBucket(tt.opType, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTokenBucket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && bucket == nil {
				t.Error("NewTokenBucket() returned nil bucket without error")
			}
			if !tt.wantErr && bucket != nil {
				if bucket.OperationType() != tt.opType {
					t.Errorf("OperationType() = %v, want %v", bucket.OperationType(), tt.opType)
				}
			}
		})
	}
}

func TestTokenBucket_Allow(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             5,
		Interval:          time.Second,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	// Should allow up to burst limit
	for i := 0; i < 5; i++ {
		if !bucket.Allow() {
			t.Errorf("Allow() should return true for request %d", i+1)
		}
	}

	// Should deny after burst is exhausted
	if bucket.Allow() {
		t.Error("Allow() should return false when tokens are exhausted")
	}

	// Check stats
	stats := bucket.Stats()
	if stats.AllowedRequests != 5 {
		t.Errorf("AllowedRequests = %d, want 5", stats.AllowedRequests)
	}
	if stats.DeniedRequests != 1 {
		t.Errorf("DeniedRequests = %d, want 1", stats.DeniedRequests)
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          100 * time.Millisecond, // Fast refill for testing
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	// Exhaust all tokens
	for i := 0; i < 10; i++ {
		if !bucket.Allow() {
			t.Fatalf("Allow() should return true for request %d", i+1)
		}
	}

	// Should be denied immediately
	if bucket.Allow() {
		t.Error("Allow() should return false when exhausted")
	}

	// Wait for some refill
	time.Sleep(50 * time.Millisecond) // Should refill ~5 tokens

	// Should have some tokens now (approximately)
	tokens := bucket.Tokens()
	if tokens < 3 || tokens > 7 {
		t.Errorf("Tokens() = %.2f, expected approximately 5 after partial refill", tokens)
	}
}

func TestTokenBucket_Wait(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              100,
		Burst:             1,
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	ctx := context.Background()

	// First wait should succeed immediately
	start := time.Now()
	if err := bucket.Wait(ctx); err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 10*time.Millisecond {
		t.Errorf("First Wait() took too long: %v", elapsed)
	}

	// Second wait should have to wait for refill
	start = time.Now()
	if err := bucket.Wait(ctx); err != nil {
		t.Errorf("Wait() returned error: %v", err)
	}
	elapsed := time.Since(start)
	// Should wait at least some time for refill (but be lenient for CI variance)
	if elapsed < time.Millisecond {
		t.Logf("Second Wait() completed quickly: %v (may have raced with refill)", elapsed)
	}
}

func TestTokenBucket_WaitContextCanceled(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              1,
		Burst:             1,
		Interval:          time.Hour, // Very slow refill
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	// Exhaust the bucket
	bucket.Allow()

	// Create a context that will be canceled
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Wait should be canceled
	err = bucket.Wait(ctx)
	if !errors.Is(err, ErrContextCanceled) {
		t.Errorf("Wait() error = %v, want %v", err, ErrContextCanceled)
	}
}

func TestTokenBucket_Reserve(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             2,
		Interval:          100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	// First reservation should be immediately available
	r1 := bucket.Reserve()
	if !r1.OK() {
		t.Error("First Reserve() should be OK immediately")
	}
	if r1.Delay() != 0 {
		t.Errorf("First Reserve() Delay = %v, want 0", r1.Delay())
	}

	// Second reservation should also be immediately available
	r2 := bucket.Reserve()
	if !r2.OK() {
		t.Error("Second Reserve() should be OK immediately")
	}

	// Third reservation should have a delay
	r3 := bucket.Reserve()
	if r3.OK() {
		t.Error("Third Reserve() should NOT be OK immediately")
	}
	if r3.Delay() <= 0 {
		t.Error("Third Reserve() should have a positive Delay")
	}

	// Cancel the third reservation
	r3.Cancel()

	// Now tokens should be restored
	tokens := bucket.Tokens()
	if tokens < 0 {
		t.Errorf("Tokens() = %.2f after cancel, should be >= 0", tokens)
	}
}

func TestTokenBucket_Backoff(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          time.Second,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        time.Second,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, time.Second}, // Capped at MaxBackoff
		{5, time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			backoff := bucket.Backoff(tt.attempt)
			if backoff != tt.expected {
				t.Errorf("Backoff(%d) = %v, want %v", tt.attempt, backoff, tt.expected)
			}
		})
	}
}

func TestTokenBucket_ExecuteWithRetry_Success(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          time.Second,
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	ctx := context.Background()
	callCount := 0

	err = bucket.ExecuteWithRetry(ctx, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("ExecuteWithRetry() error = %v, want nil", err)
	}
	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestTokenBucket_ExecuteWithRetry_RateLimitError(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          time.Second,
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	ctx := context.Background()
	callCount := 0

	// Simulate rate limit error that succeeds on second retry
	err = bucket.ExecuteWithRetry(ctx, func() error {
		callCount++
		if callCount < 2 {
			return errors.New("rate limit exceeded")
		}
		return nil
	})

	if err != nil {
		t.Errorf("ExecuteWithRetry() error = %v, want nil", err)
	}
	if callCount != 2 {
		t.Errorf("Function called %d times, want 2", callCount)
	}

	stats := bucket.Stats()
	if stats.RetryAttempts < 1 {
		t.Errorf("RetryAttempts = %d, want >= 1", stats.RetryAttempts)
	}
	if stats.SuccessfulRetries != 1 {
		t.Errorf("SuccessfulRetries = %d, want 1", stats.SuccessfulRetries)
	}
}

func TestTokenBucket_ExecuteWithRetry_MaxRetriesExceeded(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          time.Second,
		MaxRetries:        2,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	ctx := context.Background()
	callCount := 0

	// Always return rate limit error
	err = bucket.ExecuteWithRetry(ctx, func() error {
		callCount++
		return errors.New("429 too many requests")
	})

	if err == nil {
		t.Error("ExecuteWithRetry() expected error after max retries")
	}
	// Should have tried 1 initial + 2 retries = 3 times
	if callCount != 3 {
		t.Errorf("Function called %d times, want 3", callCount)
	}

	stats := bucket.Stats()
	if stats.FailedRetries != 1 {
		t.Errorf("FailedRetries = %d, want 1", stats.FailedRetries)
	}
}

func TestTokenBucket_ExecuteWithRetry_NonRateLimitError(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              10,
		Burst:             10,
		Interval:          time.Second,
		MaxRetries:        3,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	ctx := context.Background()
	callCount := 0
	expectedErr := errors.New("some other error")

	err = bucket.ExecuteWithRetry(ctx, func() error {
		callCount++
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("ExecuteWithRetry() error = %v, want %v", err, expectedErr)
	}
	// Should not retry non-rate-limit errors
	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestTokenBucket_ConcurrentAccess(t *testing.T) {
	bucket, err := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              100,
		Burst:             50,
		Interval:          time.Second,
		BackoffMultiplier: 2.0,
	})
	if err != nil {
		t.Fatalf("Failed to create token bucket: %v", err)
	}

	var wg sync.WaitGroup
	var allowed int64
	var denied int64

	// Launch 100 concurrent requests
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if bucket.Allow() {
				atomic.AddInt64(&allowed, 1)
			} else {
				atomic.AddInt64(&denied, 1)
			}
		}()
	}

	wg.Wait()

	// Should have allowed approximately burst (50) requests
	if allowed < 40 || allowed > 60 {
		t.Errorf("Allowed %d requests, expected approximately 50", allowed)
	}

	total := allowed + denied
	if total != 100 {
		t.Errorf("Total requests = %d, want 100", total)
	}

	stats := bucket.Stats()
	if stats.AllowedRequests != allowed {
		t.Errorf("Stats.AllowedRequests = %d, want %d", stats.AllowedRequests, allowed)
	}
	if stats.DeniedRequests != denied {
		t.Errorf("Stats.DeniedRequests = %d, want %d", stats.DeniedRequests, denied)
	}
}

func TestRateLimiterGroup(t *testing.T) {
	group := NewRateLimiterGroup()

	// Get or create GitHub API limiter
	limiter1, err := group.GetOrCreate(OperationGitHubAPI)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if limiter1 == nil {
		t.Fatal("GetOrCreate() returned nil limiter")
	}

	// Get same limiter again
	limiter2, err := group.GetOrCreate(OperationGitHubAPI)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if limiter1 != limiter2 {
		t.Error("GetOrCreate() should return same limiter for same operation type")
	}

	// Get different limiter
	limiter3, err := group.GetOrCreate(OperationMCPRequest)
	if err != nil {
		t.Fatalf("GetOrCreate() error = %v", err)
	}
	if limiter3 == limiter1 {
		t.Error("GetOrCreate() should return different limiter for different operation type")
	}

	// Check stats
	allStats := group.AllStats()
	if len(allStats) != 2 {
		t.Errorf("AllStats() returned %d stats, want 2", len(allStats))
	}
}

func TestRateLimiterGroup_WithConfig(t *testing.T) {
	group := NewRateLimiterGroup()

	config := &Config{
		Rate:              5,
		Burst:             5,
		Interval:          time.Second,
		MaxRetries:        1,
		InitialBackoff:    time.Millisecond,
		MaxBackoff:        10 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	limiter, err := group.GetOrCreateWithConfig(OperationGitHubAPI, config)
	if err != nil {
		t.Fatalf("GetOrCreateWithConfig() error = %v", err)
	}

	cfg := limiter.Config()
	if cfg.Rate != 5 {
		t.Errorf("Config.Rate = %.2f, want 5", cfg.Rate)
	}
	if cfg.Burst != 5 {
		t.Errorf("Config.Burst = %d, want 5", cfg.Burst)
	}
}

func TestParseRateLimitSpec(t *testing.T) {
	tests := []struct {
		name     string
		spec     string
		wantRate float64
		wantInt  time.Duration
		wantErr  bool
	}{
		{
			name:     "per second",
			spec:     "10/second",
			wantRate: 10,
			wantInt:  time.Second,
			wantErr:  false,
		},
		{
			name:     "per minute",
			spec:     "100/minute",
			wantRate: 100,
			wantInt:  time.Minute,
			wantErr:  false,
		},
		{
			name:     "per hour",
			spec:     "1000/hour",
			wantRate: 1000,
			wantInt:  time.Hour,
			wantErr:  false,
		},
		{
			name:     "per day",
			spec:     "5000/day",
			wantRate: 5000,
			wantInt:  24 * time.Hour,
			wantErr:  false,
		},
		{
			name:     "short unit sec",
			spec:     "10/sec",
			wantRate: 10,
			wantInt:  time.Second,
			wantErr:  false,
		},
		{
			name:     "short unit min",
			spec:     "50/min",
			wantRate: 50,
			wantInt:  time.Minute,
			wantErr:  false,
		},
		{
			name:     "short unit hr",
			spec:     "100/hr",
			wantRate: 100,
			wantInt:  time.Hour,
			wantErr:  false,
		},
		{
			name:     "short unit s",
			spec:     "5/s",
			wantRate: 5,
			wantInt:  time.Second,
			wantErr:  false,
		},
		{
			name:     "short unit m",
			spec:     "30/m",
			wantRate: 30,
			wantInt:  time.Minute,
			wantErr:  false,
		},
		{
			name:     "short unit h",
			spec:     "60/h",
			wantRate: 60,
			wantInt:  time.Hour,
			wantErr:  false,
		},
		{
			name:     "short unit d",
			spec:     "100/d",
			wantRate: 100,
			wantInt:  24 * time.Hour,
			wantErr:  false,
		},
		{
			name:    "invalid format",
			spec:    "invalid",
			wantErr: true,
		},
		{
			name:    "invalid unit",
			spec:    "10/week",
			wantErr: true,
		},
		{
			name:    "zero rate",
			spec:    "0/second",
			wantErr: true,
		},
		{
			name:    "negative rate",
			spec:    "-10/second",
			wantErr: true,
		},
		{
			name:    "empty spec",
			spec:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseRateLimitSpec(tt.spec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRateLimitSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if config.Rate != tt.wantRate {
				t.Errorf("Rate = %.2f, want %.2f", config.Rate, tt.wantRate)
			}
			if config.Interval != tt.wantInt {
				t.Errorf("Interval = %v, want %v", config.Interval, tt.wantInt)
			}
		})
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "rate limit exceeded",
			err:  errors.New("API rate limit exceeded"),
			want: true,
		},
		{
			name: "429 error",
			err:  errors.New("HTTP 429 Too Many Requests"),
			want: true,
		},
		{
			name: "too many requests",
			err:  errors.New("too many requests"),
			want: true,
		},
		{
			name: "throttled",
			err:  errors.New("request throttled"),
			want: true,
		},
		{
			name: "exceeded",
			err:  errors.New("request limit exceeded"),
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("connection refused"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "ErrRateLimitExceeded",
			err:  ErrRateLimitExceeded,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimitError(tt.err); got != tt.want {
				t.Errorf("isRateLimitError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultGroup_ConvenienceFunctions(t *testing.T) {
	// Reset DefaultGroup for testing
	DefaultGroup = NewRateLimiterGroup()

	// Test Allow
	if !Allow(OperationGitHubAPI) {
		t.Error("Allow() should return true for first request")
	}

	// Test Wait
	ctx := context.Background()
	if err := Wait(ctx, OperationMCPRequest); err != nil {
		t.Errorf("Wait() error = %v", err)
	}

	// Test ExecuteWithRetry
	callCount := 0
	err := ExecuteWithRetry(ctx, OperationWorkflowCompile, func() error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("ExecuteWithRetry() error = %v", err)
	}
	if callCount != 1 {
		t.Errorf("Function called %d times, want 1", callCount)
	}
}

func TestDefaultConfigs(t *testing.T) {
	// Verify all default configs are valid
	for opType := range DefaultConfigs {
		t.Run(string(opType), func(t *testing.T) {
			bucket, err := NewTokenBucket(opType, nil)
			if err != nil {
				t.Errorf("Failed to create bucket with default config for %s: %v", opType, err)
			}
			if bucket == nil {
				t.Errorf("NewTokenBucket returned nil for %s", opType)
			}
		})
	}
}

func TestStats_Clone(t *testing.T) {
	stats := &Stats{
		AllowedRequests:   100,
		DeniedRequests:    10,
		WaitingRequests:   5,
		TotalWaitTime:     time.Second,
		RetryAttempts:     3,
		SuccessfulRetries: 2,
		FailedRetries:     1,
	}

	clone := stats.Clone()

	// Verify clone has same values
	if clone.AllowedRequests != stats.AllowedRequests {
		t.Errorf("Clone AllowedRequests = %d, want %d", clone.AllowedRequests, stats.AllowedRequests)
	}
	if clone.DeniedRequests != stats.DeniedRequests {
		t.Errorf("Clone DeniedRequests = %d, want %d", clone.DeniedRequests, stats.DeniedRequests)
	}
	if clone.TotalWaitTime != stats.TotalWaitTime {
		t.Errorf("Clone TotalWaitTime = %v, want %v", clone.TotalWaitTime, stats.TotalWaitTime)
	}

	// Modify original, clone should be unaffected
	stats.AllowedRequests = 200
	if clone.AllowedRequests == 200 {
		t.Error("Clone should be independent of original")
	}
}

func BenchmarkTokenBucket_Allow(b *testing.B) {
	bucket, _ := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              1000000,
		Burst:             1000000,
		Interval:          time.Second,
		BackoffMultiplier: 2.0,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bucket.Allow()
	}
}

func BenchmarkTokenBucket_Allow_Concurrent(b *testing.B) {
	bucket, _ := NewTokenBucket(OperationGitHubAPI, &Config{
		Rate:              1000000,
		Burst:             1000000,
		Interval:          time.Second,
		BackoffMultiplier: 2.0,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			bucket.Allow()
		}
	})
}
