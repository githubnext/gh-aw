// Package workflow provides validation performance profiling for agentic workflows.
//
// # Validation Performance Profiling
//
// This file implements performance metrics collection for the validation system.
// It tracks validator execution times, external API call patterns, and cache effectiveness
// to help identify bottlenecks and optimize validation performance.
//
// # Key Types
//
//   - ValidationMetrics: Main metrics container tracking all validation performance data
//   - APICallStats: Statistics for external API calls (Docker Hub, PyPI, NPM)
//   - ValidatorTiming: Individual validator execution timing information
//
// # Usage Pattern
//
// The metrics system uses a defer pattern for timing:
//
//	defer ctx.StartValidator("validator_name")()
//
// This ensures accurate timing even when validators return early or encounter errors.
//
// # Thread Safety
//
// All metrics collection is thread-safe using sync.Mutex to support concurrent
// validation operations that may occur when compiling multiple workflows.
package workflow

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var validationMetricsLog = logger.New("workflow:validation_metrics")

// APICallStats tracks statistics for external API calls to registries
type APICallStats struct {
	Attempts    int           // Total number of API call attempts
	CacheHits   int           // Number of times the cache was used
	CacheMisses int           // Number of times the cache missed
	Timeouts    int           // Number of timeout errors
	TotalTime   time.Duration // Total time spent on API calls
	Service     string        // Service name (Docker Hub, PyPI, NPM)
}

// ValidatorTiming tracks execution time for a single validator
type ValidatorTiming struct {
	Name     string        // Validator name
	Duration time.Duration // Total execution time
}

// ValidationMetrics tracks all validation performance metrics
type ValidationMetrics struct {
	mu               sync.Mutex               // Protects concurrent access
	validatorTimings []ValidatorTiming        // All validator timings
	externalAPICalls map[string]*APICallStats // API call stats by service
	startTime        time.Time                // When validation started
	endTime          time.Time                // When validation ended
	enabled          bool                     // Whether profiling is enabled
}

// NewValidationMetrics creates a new metrics collector
func NewValidationMetrics(enabled bool) *ValidationMetrics {
	return &ValidationMetrics{
		validatorTimings: make([]ValidatorTiming, 0),
		externalAPICalls: make(map[string]*APICallStats),
		enabled:          enabled,
		startTime:        time.Now(),
	}
}

// IsEnabled returns whether profiling is enabled
func (vm *ValidationMetrics) IsEnabled() bool {
	return vm.enabled
}

// StartValidator begins timing a validator and returns a function to stop timing
// Usage: defer ctx.StartValidator("validator_name")()
func (vm *ValidationMetrics) StartValidator(name string) func() {
	if !vm.enabled {
		return func() {} // No-op when profiling disabled
	}

	start := time.Now()
	validationMetricsLog.Printf("Starting validator timing: %s", name)

	return func() {
		duration := time.Since(start)
		validationMetricsLog.Printf("Validator completed: %s (took %v)", name, duration)

		vm.mu.Lock()
		defer vm.mu.Unlock()
		vm.validatorTimings = append(vm.validatorTimings, ValidatorTiming{
			Name:     name,
			Duration: duration,
		})
	}
}

// RecordAPICall records statistics for an external API call
func (vm *ValidationMetrics) RecordAPICall(service string, cached bool, duration time.Duration, timedOut bool) {
	if !vm.enabled {
		return
	}

	validationMetricsLog.Printf("Recording API call: service=%s, cached=%v, duration=%v, timeout=%v",
		service, cached, duration, timedOut)

	vm.mu.Lock()
	defer vm.mu.Unlock()

	stats, exists := vm.externalAPICalls[service]
	if !exists {
		stats = &APICallStats{Service: service}
		vm.externalAPICalls[service] = stats
	}

	stats.Attempts++
	if cached {
		stats.CacheHits++
	} else {
		stats.CacheMisses++
	}
	if timedOut {
		stats.Timeouts++
	}
	stats.TotalTime += duration
}

// Complete marks the end of validation timing
func (vm *ValidationMetrics) Complete() {
	if !vm.enabled {
		return
	}
	vm.endTime = time.Now()
	validationMetricsLog.Printf("Validation metrics collection completed")
}

// GetTotalDuration returns the total validation time
func (vm *ValidationMetrics) GetTotalDuration() time.Duration {
	if vm.endTime.IsZero() {
		return time.Since(vm.startTime)
	}
	return vm.endTime.Sub(vm.startTime)
}

// FormatProfileReport generates a formatted performance report
func (vm *ValidationMetrics) FormatProfileReport() string {
	if !vm.enabled {
		return ""
	}

	vm.mu.Lock()
	defer vm.mu.Unlock()

	var report strings.Builder

	// Calculate total duration
	totalDuration := vm.GetTotalDuration()

	// Header
	report.WriteString("\n")
	report.WriteString(console.FormatInfoMessage("VALIDATION PERFORMANCE REPORT"))
	report.WriteString("\n")
	report.WriteString(strings.Repeat("=", 60))
	report.WriteString("\n")

	// Total validation time
	report.WriteString(fmt.Sprintf("Total validation time: %v\n", formatDuration(totalDuration)))
	report.WriteString("\n")

	// Slowest validators (top 5)
	if len(vm.validatorTimings) > 0 {
		report.WriteString("Slowest validators:\n")

		// Sort by duration descending
		sortedTimings := make([]ValidatorTiming, len(vm.validatorTimings))
		copy(sortedTimings, vm.validatorTimings)
		sort.Slice(sortedTimings, func(i, j int) bool {
			return sortedTimings[i].Duration > sortedTimings[j].Duration
		})

		// Show top 5
		count := 5
		if len(sortedTimings) < count {
			count = len(sortedTimings)
		}

		for i := 0; i < count; i++ {
			timing := sortedTimings[i]
			percentage := float64(timing.Duration) / float64(totalDuration) * 100
			report.WriteString(fmt.Sprintf("%d. %s: %v (%.1f%%)\n",
				i+1,
				timing.Name,
				formatDuration(timing.Duration),
				percentage,
			))
		}
		report.WriteString("\n")
	}

	// External API calls
	if len(vm.externalAPICalls) > 0 {
		report.WriteString("External API Calls:\n")

		// Sort services alphabetically
		services := make([]string, 0, len(vm.externalAPICalls))
		for service := range vm.externalAPICalls {
			services = append(services, service)
		}
		sort.Strings(services)

		for _, service := range services {
			stats := vm.externalAPICalls[service]
			hitRate := 0.0
			if stats.Attempts > 0 {
				hitRate = float64(stats.CacheHits) / float64(stats.Attempts) * 100
			}
			avgLatency := time.Duration(0)
			if stats.Attempts > 0 {
				avgLatency = stats.TotalTime / time.Duration(stats.Attempts)
			}

			line := fmt.Sprintf("- %s: %d attempts, %d hits (%.0f%%)",
				service,
				stats.Attempts,
				stats.CacheHits,
				hitRate,
			)

			if avgLatency > 0 {
				line += fmt.Sprintf(", avg %v", formatDuration(avgLatency))
			}

			if stats.Timeouts > 0 {
				line += fmt.Sprintf(", %d timeouts", stats.Timeouts)
			}

			report.WriteString(line)
			report.WriteString("\n")
		}
		report.WriteString("\n")
	}

	// Cache effectiveness
	totalAttempts := 0
	totalHits := 0
	for _, stats := range vm.externalAPICalls {
		totalAttempts += stats.Attempts
		totalHits += stats.CacheHits
	}

	if totalAttempts > 0 {
		report.WriteString("Cache Effectiveness:\n")
		hitRate := float64(totalHits) / float64(totalAttempts) * 100
		report.WriteString(fmt.Sprintf("- Overall hit rate: %.1f%% (%d/%d cached)\n",
			hitRate,
			totalHits,
			totalAttempts,
		))

		// Estimate time saved by caching (assuming cache is 10x faster than API call)
		var totalTimeSaved time.Duration
		for _, stats := range vm.externalAPICalls {
			if stats.CacheHits > 0 && stats.Attempts > 0 {
				avgCallTime := stats.TotalTime / time.Duration(stats.Attempts)
				// Cache saves ~90% of call time (assuming cache is ~10x faster)
				timeSavedPerHit := avgCallTime * 9 / 10
				totalTimeSaved += timeSavedPerHit * time.Duration(stats.CacheHits)
			}
		}
		if totalTimeSaved > 0 {
			report.WriteString(fmt.Sprintf("- Cache saves: ~%v (estimated)\n",
				formatDuration(totalTimeSaved),
			))
		}
	}

	return report.String()
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	// For durations < 1ms, show in microseconds
	if d < time.Millisecond {
		return fmt.Sprintf("%.0fµs", float64(d.Microseconds()))
	}
	// For durations < 1s, show in milliseconds
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Milliseconds()))
	}
	// For durations >= 1s, show in seconds with 3 decimal places
	return fmt.Sprintf("%.3fs", d.Seconds())
}

// PrintProfileReport prints the performance report to stderr
func (vm *ValidationMetrics) PrintProfileReport() {
	if !vm.enabled {
		return
	}

	report := vm.FormatProfileReport()
	if report != "" {
		fmt.Fprintln(os.Stderr, report)
	}
}
