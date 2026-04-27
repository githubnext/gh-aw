//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractCircuitBreakerConfig tests that circuit-breaker frontmatter is correctly parsed.
func TestExtractCircuitBreakerConfig(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name           string
		frontmatter    map[string]any
		expectedConfig *CircuitBreakerConfig
	}{
		{
			name:           "no circuit-breaker key",
			frontmatter:    map[string]any{},
			expectedConfig: nil,
		},
		{
			name:           "circuit-breaker: false (explicit disable)",
			frontmatter:    map[string]any{"circuit-breaker": false},
			expectedConfig: nil,
		},
		{
			name:        "circuit-breaker: true (boolean enable, use defaults)",
			frontmatter: map[string]any{"circuit-breaker": true},
			expectedConfig: func() *CircuitBreakerConfig {
				t := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 5,
					TimeWindow:             "24h",
					Cooldown:               "1h",
					Notify:                 &t,
				}
			}(),
		},
		{
			name: "circuit-breaker: object with all fields",
			frontmatter: map[string]any{
				"circuit-breaker": map[string]any{
					"max-consecutive-failures": 3,
					"time-window":              "6h",
					"cooldown":                 "30m",
					"notify":                   false,
				},
			},
			expectedConfig: func() *CircuitBreakerConfig {
				f := false
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 3,
					TimeWindow:             "6h",
					Cooldown:               "30m",
					Notify:                 &f,
				}
			}(),
		},
		{
			name: "circuit-breaker: object with defaults applied",
			frontmatter: map[string]any{
				"circuit-breaker": map[string]any{
					"max-consecutive-failures": 10,
				},
			},
			expectedConfig: func() *CircuitBreakerConfig {
				tr := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 10,
					TimeWindow:             "24h",
					Cooldown:               "1h",
					Notify:                 &tr,
				}
			}(),
		},
		{
			name: "circuit-breaker enabled via features flag",
			frontmatter: map[string]any{
				"features": map[string]any{
					"circuit-breaker": true,
				},
			},
			expectedConfig: func() *CircuitBreakerConfig {
				tr := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 5,
					TimeWindow:             "24h",
					Cooldown:               "1h",
					Notify:                 &tr,
				}
			}(),
		},
		{
			name: "circuit-breaker NOT enabled via features flag (false)",
			frontmatter: map[string]any{
				"features": map[string]any{
					"circuit-breaker": false,
				},
			},
			expectedConfig: nil,
		},
		{
			name: "max-consecutive-failures as float64 (YAML parser produces float64 for numbers)",
			frontmatter: map[string]any{
				"circuit-breaker": map[string]any{
					"max-consecutive-failures": float64(7),
				},
			},
			expectedConfig: func() *CircuitBreakerConfig {
				tr := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 7,
					TimeWindow:             "24h",
					Cooldown:               "1h",
					Notify:                 &tr,
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compiler.extractCircuitBreakerConfig(tt.frontmatter)
			if tt.expectedConfig == nil {
				assert.Nil(t, got, "expected nil CircuitBreakerConfig")
				return
			}
			require.NotNil(t, got, "expected non-nil CircuitBreakerConfig")
			assert.Equal(t, tt.expectedConfig.MaxConsecutiveFailures, got.MaxConsecutiveFailures,
				"MaxConsecutiveFailures should match")
			assert.Equal(t, tt.expectedConfig.TimeWindow, got.TimeWindow,
				"TimeWindow should match")
			assert.Equal(t, tt.expectedConfig.Cooldown, got.Cooldown,
				"Cooldown should match")
			require.NotNil(t, got.Notify, "Notify should not be nil")
			assert.Equal(t, *tt.expectedConfig.Notify, *got.Notify,
				"Notify should match")
		})
	}
}

// TestApplyCircuitBreakerDefaults tests that defaults are applied correctly.
func TestApplyCircuitBreakerDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    *CircuitBreakerConfig
		expected *CircuitBreakerConfig
	}{
		{
			name:  "empty config gets all defaults",
			input: &CircuitBreakerConfig{},
			expected: func() *CircuitBreakerConfig {
				tr := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 5,
					TimeWindow:             "24h",
					Cooldown:               "1h",
					Notify:                 &tr,
				}
			}(),
		},
		{
			name: "existing values are preserved",
			input: func() *CircuitBreakerConfig {
				f := false
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 3,
					TimeWindow:             "6h",
					Cooldown:               "30m",
					Notify:                 &f,
				}
			}(),
			expected: func() *CircuitBreakerConfig {
				f := false
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 3,
					TimeWindow:             "6h",
					Cooldown:               "30m",
					Notify:                 &f,
				}
			}(),
		},
		{
			name: "zero max-consecutive-failures gets default",
			input: &CircuitBreakerConfig{
				MaxConsecutiveFailures: 0,
				TimeWindow:             "6h",
			},
			expected: func() *CircuitBreakerConfig {
				tr := true
				return &CircuitBreakerConfig{
					MaxConsecutiveFailures: 5,
					TimeWindow:             "6h",
					Cooldown:               "1h",
					Notify:                 &tr,
				}
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyCircuitBreakerDefaults(tt.input)
			assert.Equal(t, tt.expected.MaxConsecutiveFailures, tt.input.MaxConsecutiveFailures,
				"MaxConsecutiveFailures should match")
			assert.Equal(t, tt.expected.TimeWindow, tt.input.TimeWindow,
				"TimeWindow should match")
			assert.Equal(t, tt.expected.Cooldown, tt.input.Cooldown,
				"Cooldown should match")
			require.NotNil(t, tt.input.Notify, "Notify should not be nil after defaults")
			assert.Equal(t, *tt.expected.Notify, *tt.input.Notify,
				"Notify should match")
		})
	}
}

// TestCircuitBreakerDurationToMinutes tests duration string parsing.
func TestCircuitBreakerDurationToMinutes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int
		expectError bool
	}{
		{name: "1 hour", input: "1h", expected: 60},
		{name: "24 hours", input: "24h", expected: 1440},
		{name: "30 minutes", input: "30m", expected: 30},
		{name: "90 minutes", input: "1h30m", expected: 90},
		{name: "invalid duration", input: "invalid", expectError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := circuitBreakerDurationToMinutes(tt.input)
			if tt.expectError {
				assert.Error(t, err, "should return error for invalid duration")
				return
			}
			require.NoError(t, err, "should not return error for valid duration")
			assert.Equal(t, tt.expected, got, "duration in minutes should match")
		})
	}
}

// TestGenerateCircuitBreakerCheckSteps tests that the generated YAML steps are well-formed.
func TestGenerateCircuitBreakerCheckSteps(t *testing.T) {
	tr := true
	data := &WorkflowData{
		Name: "My Workflow",
		CircuitBreaker: &CircuitBreakerConfig{
			MaxConsecutiveFailures: 5,
			TimeWindow:             "24h",
			Cooldown:               "1h",
			Notify:                 &tr,
		},
	}

	compiler := &Compiler{}
	var steps []string
	steps = compiler.generateCircuitBreakerCheckSteps(data, steps)

	require.NotEmpty(t, steps, "should generate steps")

	combined := strings.Join(steps, "")

	// Step 1: find artifact
	assert.Contains(t, combined, "find_circuit_breaker_artifact", "find-artifact step ID should be present")
	assert.Contains(t, combined, "find_circuit_breaker_artifact.cjs", "find-artifact script should be referenced")

	// Step 2: download artifact
	assert.Contains(t, combined, "Download previous circuit breaker state", "download step name should be present")
	assert.Contains(t, combined, "circuit-breaker-state", "artifact name should be present")
	assert.Contains(t, combined, "previous_run_id", "run-id reference should be present")

	// Step 3: evaluate state
	assert.Contains(t, combined, "check_circuit_breaker", "check step ID should be present")
	assert.Contains(t, combined, "GH_AW_CB_MAX_FAILURES", "max failures env var should be present")
	assert.Contains(t, combined, "GH_AW_CB_TIME_WINDOW_MINUTES", "time window env var should be present")
	assert.Contains(t, combined, "GH_AW_CB_COOLDOWN_MINUTES", "cooldown env var should be present")
	assert.Contains(t, combined, "GH_AW_CB_NOTIFY", "notify env var should be present")
	assert.Contains(t, combined, "check_circuit_breaker.cjs", "check script should be referenced")
}

// TestGenerateCircuitBreakerCheckSteps_NilConfig ensures no steps are generated when disabled.
func TestGenerateCircuitBreakerCheckSteps_NilConfig(t *testing.T) {
	data := &WorkflowData{
		Name:           "My Workflow",
		CircuitBreaker: nil,
	}

	compiler := &Compiler{}
	var steps []string
	steps = compiler.generateCircuitBreakerCheckSteps(data, steps)

	assert.Empty(t, steps, "should generate no steps when circuit breaker is disabled")
}

// TestGenerateCircuitBreakerUpdateSteps tests that update steps are generated correctly.
func TestGenerateCircuitBreakerUpdateSteps(t *testing.T) {
	tr := true
	data := &WorkflowData{
		Name: "My Workflow",
		CircuitBreaker: &CircuitBreakerConfig{
			MaxConsecutiveFailures: 5,
			TimeWindow:             "24h",
			Cooldown:               "1h",
			Notify:                 &tr,
		},
	}

	compiler := &Compiler{}
	var yaml strings.Builder
	compiler.generateCircuitBreakerUpdateSteps(&yaml, data)

	output := yaml.String()
	assert.Contains(t, output, "Update circuit breaker state", "update step name should be present")
	assert.Contains(t, output, "if: always()", "update step should run always")
	assert.Contains(t, output, "update_circuit_breaker.cjs", "update script should be referenced")
	assert.Contains(t, output, "Upload circuit breaker state", "upload step should be present")
	assert.Contains(t, output, "circuit-breaker-state", "artifact name should be present")
	assert.Contains(t, output, "GH_AW_CB_JOB_STATUS", "job status env var should be present")
}

// TestGenerateCircuitBreakerUpdateSteps_NilConfig ensures no steps are generated when disabled.
func TestGenerateCircuitBreakerUpdateSteps_NilConfig(t *testing.T) {
	data := &WorkflowData{
		Name:           "My Workflow",
		CircuitBreaker: nil,
	}

	compiler := &Compiler{}
	var yaml strings.Builder
	compiler.generateCircuitBreakerUpdateSteps(&yaml, data)

	assert.Empty(t, yaml.String(), "should generate no steps when circuit breaker is disabled")
}
