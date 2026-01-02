package workflow

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidationMetrics(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{
			name:    "enabled metrics",
			enabled: true,
		},
		{
			name:    "disabled metrics",
			enabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewValidationMetrics(tt.enabled)
			require.NotNil(t, metrics)
			assert.Equal(t, tt.enabled, metrics.IsEnabled())
			assert.NotZero(t, metrics.startTime)
			assert.NotNil(t, metrics.validatorTimings)
			assert.NotNil(t, metrics.externalAPICalls)
		})
	}
}

func TestStartValidator(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		validatorName string
		sleepDuration time.Duration
		expectTiming  bool
	}{
		{
			name:          "enabled - tracks timing",
			enabled:       true,
			validatorName: "test_validator",
			sleepDuration: 10 * time.Millisecond,
			expectTiming:  true,
		},
		{
			name:          "disabled - no timing",
			enabled:       false,
			validatorName: "test_validator",
			sleepDuration: 10 * time.Millisecond,
			expectTiming:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewValidationMetrics(tt.enabled)

			// Start validator timing
			done := metrics.StartValidator(tt.validatorName)
			time.Sleep(tt.sleepDuration)
			done()

			if tt.expectTiming {
				assert.Len(t, metrics.validatorTimings, 1, "Should have recorded one timing")
				assert.Equal(t, tt.validatorName, metrics.validatorTimings[0].Name)
				assert.GreaterOrEqual(t, metrics.validatorTimings[0].Duration, tt.sleepDuration,
					"Duration should be at least the sleep duration")
			} else {
				assert.Len(t, metrics.validatorTimings, 0, "Should not record timing when disabled")
			}
		})
	}
}

func TestStartValidatorDefer(t *testing.T) {
	metrics := NewValidationMetrics(true)

	func() {
		defer metrics.StartValidator("defer_test")()
		time.Sleep(5 * time.Millisecond)
	}()

	require.Len(t, metrics.validatorTimings, 1)
	assert.Equal(t, "defer_test", metrics.validatorTimings[0].Name)
	assert.GreaterOrEqual(t, metrics.validatorTimings[0].Duration, 5*time.Millisecond)
}

func TestRecordAPICall(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		service   string
		cached    bool
		duration  time.Duration
		timedOut  bool
		expectAPI bool
	}{
		{
			name:      "enabled - cache hit",
			enabled:   true,
			service:   "Docker Hub",
			cached:    true,
			duration:  50 * time.Millisecond,
			timedOut:  false,
			expectAPI: true,
		},
		{
			name:      "enabled - cache miss",
			enabled:   true,
			service:   "PyPI",
			cached:    false,
			duration:  100 * time.Millisecond,
			timedOut:  false,
			expectAPI: true,
		},
		{
			name:      "enabled - timeout",
			enabled:   true,
			service:   "NPM",
			cached:    false,
			duration:  5 * time.Second,
			timedOut:  true,
			expectAPI: true,
		},
		{
			name:      "disabled - no tracking",
			enabled:   false,
			service:   "Docker Hub",
			cached:    true,
			duration:  50 * time.Millisecond,
			timedOut:  false,
			expectAPI: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewValidationMetrics(tt.enabled)

			metrics.RecordAPICall(tt.service, tt.cached, tt.duration, tt.timedOut)

			if tt.expectAPI {
				stats, exists := metrics.externalAPICalls[tt.service]
				require.True(t, exists, "Service stats should exist")
				assert.Equal(t, 1, stats.Attempts)
				assert.Equal(t, tt.duration, stats.TotalTime)

				if tt.cached {
					assert.Equal(t, 1, stats.CacheHits)
					assert.Equal(t, 0, stats.CacheMisses)
				} else {
					assert.Equal(t, 0, stats.CacheHits)
					assert.Equal(t, 1, stats.CacheMisses)
				}

				if tt.timedOut {
					assert.Equal(t, 1, stats.Timeouts)
				} else {
					assert.Equal(t, 0, stats.Timeouts)
				}
			} else {
				assert.Len(t, metrics.externalAPICalls, 0, "Should not track API calls when disabled")
			}
		})
	}
}

func TestRecordAPICallMultiple(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Record multiple calls to the same service
	metrics.RecordAPICall("Docker Hub", true, 10*time.Millisecond, false)
	metrics.RecordAPICall("Docker Hub", true, 15*time.Millisecond, false)
	metrics.RecordAPICall("Docker Hub", false, 100*time.Millisecond, false)
	metrics.RecordAPICall("Docker Hub", false, 120*time.Millisecond, true)

	stats := metrics.externalAPICalls["Docker Hub"]
	require.NotNil(t, stats)

	assert.Equal(t, 4, stats.Attempts)
	assert.Equal(t, 2, stats.CacheHits)
	assert.Equal(t, 2, stats.CacheMisses)
	assert.Equal(t, 1, stats.Timeouts)
	assert.Equal(t, 245*time.Millisecond, stats.TotalTime)
}

func TestGetTotalDuration(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Initial duration should be time since start
	time.Sleep(10 * time.Millisecond)
	duration1 := metrics.GetTotalDuration()
	assert.GreaterOrEqual(t, duration1, 10*time.Millisecond)

	// After Complete(), duration should be fixed
	metrics.Complete()
	time.Sleep(10 * time.Millisecond)
	duration2 := metrics.GetTotalDuration()
	assert.Equal(t, duration2, metrics.GetTotalDuration(), "Duration should be fixed after Complete()")
}

func TestFormatProfileReport(t *testing.T) {
	tests := []struct {
		name               string
		enabled            bool
		setupFunc          func(*ValidationMetrics)
		expectedContains   []string
		expectedNotContain []string
	}{
		{
			name:    "disabled - empty report",
			enabled: false,
			setupFunc: func(vm *ValidationMetrics) {
				vm.StartValidator("test")()
			},
			expectedContains:   []string{},
			expectedNotContain: []string{"VALIDATION PERFORMANCE REPORT"},
		},
		{
			name:    "enabled - basic report",
			enabled: true,
			setupFunc: func(vm *ValidationMetrics) {
				done := vm.StartValidator("docker_validation")
				time.Sleep(5 * time.Millisecond)
				done()

				vm.RecordAPICall("Docker Hub", true, 10*time.Millisecond, false)
				vm.Complete()
			},
			expectedContains: []string{
				"VALIDATION PERFORMANCE REPORT",
				"Total validation time:",
				"Slowest validators:",
				"docker_validation:",
				"External API Calls:",
				"Docker Hub:",
				"Cache Effectiveness:",
			},
			expectedNotContain: []string{},
		},
		{
			name:    "enabled - multiple validators",
			enabled: true,
			setupFunc: func(vm *ValidationMetrics) {
				// Add multiple validators with different durations
				done1 := vm.StartValidator("docker_validation")
				time.Sleep(20 * time.Millisecond)
				done1()

				done2 := vm.StartValidator("pip_validation")
				time.Sleep(10 * time.Millisecond)
				done2()

				done3 := vm.StartValidator("npm_validation")
				time.Sleep(5 * time.Millisecond)
				done3()

				vm.Complete()
			},
			expectedContains: []string{
				"1. docker_validation:",
				"2. pip_validation:",
				"3. npm_validation:",
			},
			expectedNotContain: []string{},
		},
		{
			name:    "enabled - cache stats",
			enabled: true,
			setupFunc: func(vm *ValidationMetrics) {
				// Record API calls with various cache hit/miss patterns
				vm.RecordAPICall("Docker Hub", true, 10*time.Millisecond, false)
				vm.RecordAPICall("Docker Hub", true, 15*time.Millisecond, false)
				vm.RecordAPICall("Docker Hub", false, 100*time.Millisecond, false)

				vm.RecordAPICall("PyPI", true, 20*time.Millisecond, false)
				vm.RecordAPICall("PyPI", false, 80*time.Millisecond, false)

				vm.Complete()
			},
			expectedContains: []string{
				"Docker Hub: 3 attempts, 2 hits (67%)",
				"PyPI: 2 attempts, 1 hits (50%)",
				"Overall hit rate: 60.0% (3/5 cached)",
			},
			expectedNotContain: []string{},
		},
		{
			name:    "enabled - with timeouts",
			enabled: true,
			setupFunc: func(vm *ValidationMetrics) {
				vm.RecordAPICall("Docker Hub", false, 5*time.Second, true)
				vm.RecordAPICall("Docker Hub", false, 5*time.Second, true)
				vm.RecordAPICall("NPM", false, 100*time.Millisecond, false)
				vm.Complete()
			},
			expectedContains: []string{
				"Docker Hub: 2 attempts, 0 hits (0%), avg",
				"2 timeouts",
			},
			expectedNotContain: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := NewValidationMetrics(tt.enabled)
			if tt.setupFunc != nil {
				tt.setupFunc(metrics)
			}

			report := metrics.FormatProfileReport()

			for _, expected := range tt.expectedContains {
				assert.Contains(t, report, expected, "Report should contain: %s", expected)
			}

			for _, unexpected := range tt.expectedNotContain {
				assert.NotContains(t, report, unexpected, "Report should not contain: %s", unexpected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			expected: "500µs",
		},
		{
			name:     "milliseconds",
			duration: 50 * time.Millisecond,
			expected: "50ms",
		},
		{
			name:     "seconds",
			duration: 1234 * time.Millisecond,
			expected: "1.234s",
		},
		{
			name:     "sub-millisecond",
			duration: 100 * time.Microsecond,
			expected: "100µs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Simulate concurrent validator execution
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			validatorName := "validator_" + string(rune('A'+id))
			defer metrics.StartValidator(validatorName)()
			time.Sleep(time.Millisecond)
			metrics.RecordAPICall("Test Service", id%2 == 0, time.Millisecond, false)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	metrics.Complete()

	// Verify all validators were recorded
	assert.Len(t, metrics.validatorTimings, 10, "All validators should be recorded")

	// Verify API calls were recorded
	stats := metrics.externalAPICalls["Test Service"]
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.Attempts)
	assert.Equal(t, 5, stats.CacheHits)
	assert.Equal(t, 5, stats.CacheMisses)
}

func TestFormatProfileReportPercentages(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Add validators with known durations for percentage calculation
	done1 := metrics.StartValidator("validator_a")
	time.Sleep(100 * time.Millisecond)
	done1()

	done2 := metrics.StartValidator("validator_b")
	time.Sleep(50 * time.Millisecond)
	done2()

	metrics.Complete()

	report := metrics.FormatProfileReport()

	// Check that percentages are present and reasonable
	assert.Contains(t, report, "validator_a:")
	assert.Contains(t, report, "validator_b:")

	// Both should have percentage indicators
	assert.True(t, strings.Contains(report, "%"), "Report should contain percentage indicators")
}

func TestZeroAPICallsNoError(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Complete without any API calls
	metrics.Complete()

	// Should not panic and should return a valid report
	report := metrics.FormatProfileReport()
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "VALIDATION PERFORMANCE REPORT")
	assert.NotContains(t, report, "Cache Effectiveness") // No cache stats when no API calls
}

func TestNoValidatorsNoError(t *testing.T) {
	metrics := NewValidationMetrics(true)

	// Complete without any validators
	metrics.Complete()

	// Should not panic and should return a valid report
	report := metrics.FormatProfileReport()
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "VALIDATION PERFORMANCE REPORT")
	assert.NotContains(t, report, "Slowest validators") // No validators section when none recorded
}
