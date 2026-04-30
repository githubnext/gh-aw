//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateIntRange tests the validateIntRange helper function with boundary values
func TestValidateIntRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		fieldName string
		wantError bool
		errorText string
	}{
		{
			name:      "value at minimum",
			value:     1,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: false,
		},
		{
			name:      "value at maximum",
			value:     100,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: false,
		},
		{
			name:      "value in middle of range",
			value:     50,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: false,
		},
		{
			name:      "value below minimum",
			value:     0,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 1 and 100, got 0",
		},
		{
			name:      "value above maximum",
			value:     101,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 1 and 100, got 101",
		},
		{
			name:      "negative value below minimum",
			value:     -1,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 1 and 100, got -1",
		},
		{
			name:      "zero when minimum is zero",
			value:     0,
			min:       0,
			max:       100,
			fieldName: "test-field",
			wantError: false,
		},
		{
			name:      "large negative value",
			value:     -9999,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 1 and 100, got -9999",
		},
		{
			name:      "large positive value exceeding maximum",
			value:     999999,
			min:       1,
			max:       100,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 1 and 100, got 999999",
		},
		{
			name:      "single value range (min equals max)",
			value:     42,
			min:       42,
			max:       42,
			fieldName: "test-field",
			wantError: false,
		},
		{
			name:      "single value range - below",
			value:     41,
			min:       42,
			max:       42,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 42 and 42, got 41",
		},
		{
			name:      "single value range - above",
			value:     43,
			min:       42,
			max:       42,
			fieldName: "test-field",
			wantError: true,
			errorText: "test-field must be between 42 and 42, got 43",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntRange(tt.value, tt.min, tt.max, tt.fieldName)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorText, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestValidateIntRangeWithRealWorldValues tests validateIntRange with actual constraint values
func TestValidateIntRangeWithRealWorldValues(t *testing.T) {
	tests := []struct {
		name      string
		value     int
		min       int
		max       int
		fieldName string
		wantError bool
	}{
		// Port validation (1-65535)
		{
			name:      "port - minimum valid",
			value:     1,
			min:       1,
			max:       65535,
			fieldName: "port",
			wantError: false,
		},
		{
			name:      "port - maximum valid",
			value:     65535,
			min:       1,
			max:       65535,
			fieldName: "port",
			wantError: false,
		},
		{
			name:      "port - zero invalid",
			value:     0,
			min:       1,
			max:       65535,
			fieldName: "port",
			wantError: true,
		},
		{
			name:      "port - above maximum",
			value:     65536,
			min:       1,
			max:       65535,
			fieldName: "port",
			wantError: true,
		},

		// Max-file-size validation (1-104857600)
		{
			name:      "max-file-size - minimum valid",
			value:     1,
			min:       1,
			max:       104857600,
			fieldName: "max-file-size",
			wantError: false,
		},
		{
			name:      "max-file-size - maximum valid",
			value:     104857600,
			min:       1,
			max:       104857600,
			fieldName: "max-file-size",
			wantError: false,
		},
		{
			name:      "max-file-size - zero invalid",
			value:     0,
			min:       1,
			max:       104857600,
			fieldName: "max-file-size",
			wantError: true,
		},
		{
			name:      "max-file-size - above maximum",
			value:     104857601,
			min:       1,
			max:       104857600,
			fieldName: "max-file-size",
			wantError: true,
		},

		// Max-file-count validation (1-1000)
		{
			name:      "max-file-count - minimum valid",
			value:     1,
			min:       1,
			max:       1000,
			fieldName: "max-file-count",
			wantError: false,
		},
		{
			name:      "max-file-count - maximum valid",
			value:     1000,
			min:       1,
			max:       1000,
			fieldName: "max-file-count",
			wantError: false,
		},
		{
			name:      "max-file-count - zero invalid",
			value:     0,
			min:       1,
			max:       1000,
			fieldName: "max-file-count",
			wantError: true,
		},
		{
			name:      "max-file-count - above maximum",
			value:     1001,
			min:       1,
			max:       1000,
			fieldName: "max-file-count",
			wantError: true,
		},

		// Retention-days validation (1-90)
		{
			name:      "retention-days - minimum valid",
			value:     1,
			min:       1,
			max:       90,
			fieldName: "retention-days",
			wantError: false,
		},
		{
			name:      "retention-days - maximum valid",
			value:     90,
			min:       1,
			max:       90,
			fieldName: "retention-days",
			wantError: false,
		},
		{
			name:      "retention-days - zero invalid",
			value:     0,
			min:       1,
			max:       90,
			fieldName: "retention-days",
			wantError: true,
		},
		{
			name:      "retention-days - above maximum",
			value:     91,
			min:       1,
			max:       90,
			fieldName: "retention-days",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntRange(tt.value, tt.min, tt.max, tt.fieldName)

			if tt.wantError {
				if err == nil {
					t.Errorf("Expected error for %s=%d, got nil", tt.fieldName, tt.value)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for %s=%d, got: %v", tt.fieldName, tt.value, err)
				}
			}
		})
	}
}

// TestDirExists tests the fileutil.DirExists helper function
func TestDirExists(t *testing.T) {
	t.Run("empty path returns false", func(t *testing.T) {
		result := fileutil.DirExists("")
		assert.False(t, result, "empty path should return false")
	})

	t.Run("non-existent path returns false", func(t *testing.T) {
		result := fileutil.DirExists("/nonexistent/path/to/directory")
		assert.False(t, result, "non-existent path should return false")
	})

	t.Run("file path returns false", func(t *testing.T) {
		// validation_helpers.go should exist and be a file, not a directory
		result := fileutil.DirExists("validation_helpers.go")
		assert.False(t, result, "file path should return false")
	})

	t.Run("directory path returns true", func(t *testing.T) {
		// Current directory should exist
		result := fileutil.DirExists(".")
		assert.True(t, result, "current directory should return true")
	})

	t.Run("parent directory returns true", func(t *testing.T) {
		// Parent directory should exist
		result := fileutil.DirExists("..")
		assert.True(t, result, "parent directory should return true")
	})
}

// TestValidateMountStringFormat tests the shared mount format validation primitive.
func TestValidateMountStringFormat(t *testing.T) {
	tests := []struct {
		name     string
		mount    string
		wantErr  bool
		wantSrc  string
		wantDest string
		wantMode string
		allEmpty bool // true when format error (all three return values are empty)
	}{
		{
			name:     "valid ro mount",
			mount:    "/host/data:/data:ro",
			wantSrc:  "/host/data",
			wantDest: "/data",
			wantMode: "ro",
		},
		{
			name:     "valid rw mount",
			mount:    "/host/data:/data:rw",
			wantSrc:  "/host/data",
			wantDest: "/data",
			wantMode: "rw",
		},
		{
			name:     "too few parts — format error, all values empty",
			mount:    "/host/path:/container/path",
			wantErr:  true,
			allEmpty: true,
		},
		{
			name:     "too many parts — format error, all values empty",
			mount:    "/host/path:/container/path:ro:extra",
			wantErr:  true,
			allEmpty: true,
		},
		{
			name:     "invalid mode — source and dest returned, mode returned",
			mount:    "/host/path:/container/path:xyz",
			wantErr:  true,
			wantSrc:  "/host/path",
			wantDest: "/container/path",
			wantMode: "xyz",
		},
		{
			name:     "empty mode — source and dest returned, mode empty string",
			mount:    "/host/path:/container/path:",
			wantErr:  true,
			wantSrc:  "/host/path",
			wantDest: "/container/path",
			wantMode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dest, mode, err := validateMountStringFormat(tt.mount)

			if tt.wantErr {
				require.Error(t, err, "expected an error for mount %q", tt.mount)
				if tt.allEmpty {
					assert.Empty(t, src, "source should be empty on format error")
					assert.Empty(t, dest, "dest should be empty on format error")
					assert.Empty(t, mode, "mode should be empty on format error")
				} else {
					assert.Equal(t, tt.wantSrc, src, "source mismatch")
					assert.Equal(t, tt.wantDest, dest, "dest mismatch")
					assert.Equal(t, tt.wantMode, mode, "mode mismatch")
				}
			} else {
				require.NoError(t, err, "unexpected error for mount %q", tt.mount)
				assert.Equal(t, tt.wantSrc, src, "source mismatch")
				assert.Equal(t, tt.wantDest, dest, "dest mismatch")
				assert.Equal(t, tt.wantMode, mode, "mode mismatch")
			}
		})
	}
}

// TestContainsTrigger tests the shared trigger detection helper.
func TestContainsTrigger(t *testing.T) {
	tests := []struct {
		name        string
		onSection   any
		triggerName string
		expected    bool
	}{
		// string form
		{
			name:        "string: matching trigger",
			onSection:   "workflow_dispatch",
			triggerName: "workflow_dispatch",
			expected:    true,
		},
		{
			name:        "string: non-matching trigger",
			onSection:   "push",
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		{
			name:        "string: partial match does not count",
			onSection:   "workflow_dispatch_extra",
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		// []any form
		{
			name:        "slice: contains trigger",
			onSection:   []any{"push", "workflow_dispatch"},
			triggerName: "workflow_dispatch",
			expected:    true,
		},
		{
			name:        "slice: trigger absent",
			onSection:   []any{"push", "pull_request"},
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		{
			name:        "slice: single element matching",
			onSection:   []any{"workflow_call"},
			triggerName: "workflow_call",
			expected:    true,
		},
		{
			name:        "slice: non-string element skipped",
			onSection:   []any{42, "workflow_dispatch"},
			triggerName: "workflow_dispatch",
			expected:    true,
		},
		{
			name:        "slice: empty slice",
			onSection:   []any{},
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		// map form
		{
			name:        "map: trigger key present",
			onSection:   map[string]any{"workflow_dispatch": nil},
			triggerName: "workflow_dispatch",
			expected:    true,
		},
		{
			name:        "map: trigger key present with value",
			onSection:   map[string]any{"workflow_dispatch": map[string]any{"inputs": nil}},
			triggerName: "workflow_dispatch",
			expected:    true,
		},
		{
			name:        "map: trigger key absent",
			onSection:   map[string]any{"push": nil, "pull_request": nil},
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		{
			name:        "map: empty map",
			onSection:   map[string]any{},
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		// nil / unsupported types
		{
			name:        "nil onSection",
			onSection:   nil,
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		{
			name:        "unsupported type",
			onSection:   42,
			triggerName: "workflow_dispatch",
			expected:    false,
		},
		// workflow_call variant (ensures triggerName is respected)
		{
			name:        "string: workflow_call matching",
			onSection:   "workflow_call",
			triggerName: "workflow_call",
			expected:    true,
		},
		{
			name:        "string: workflow_dispatch does not match workflow_call",
			onSection:   "workflow_dispatch",
			triggerName: "workflow_call",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsTrigger(tt.onSection, tt.triggerName)
			if result != tt.expected {
				t.Errorf("containsTrigger(%v, %q) = %v, want %v", tt.onSection, tt.triggerName, result, tt.expected)
			}
		})
	}
}

// TestPreprocessProtectedFilesField tests the preprocessProtectedFilesField helper.
func TestPreprocessProtectedFilesField(t *testing.T) {
	tests := []struct {
		name          string
		configData    map[string]any
		wantExclude   []string
		wantPFAfter   any  // expected value of configData["protected-files"] after preprocessing
		wantPFPresent bool // whether configData["protected-files"] should exist after preprocessing
	}{
		{
			name:          "string form passes through unchanged",
			configData:    map[string]any{"protected-files": "blocked"},
			wantExclude:   nil,
			wantPFAfter:   "blocked",
			wantPFPresent: true,
		},
		{
			name:          "string form allowed passes through unchanged",
			configData:    map[string]any{"protected-files": "allowed"},
			wantExclude:   nil,
			wantPFAfter:   "allowed",
			wantPFPresent: true,
		},
		{
			name: "object form with policy and exclude",
			configData: map[string]any{
				"protected-files": map[string]any{
					"policy":  "fallback-to-issue",
					"exclude": []any{"AGENTS.md", "CLAUDE.md"},
				},
			},
			wantExclude:   []string{"AGENTS.md", "CLAUDE.md"},
			wantPFAfter:   "fallback-to-issue",
			wantPFPresent: true,
		},
		{
			name: "object form with exclude only (no policy)",
			configData: map[string]any{
				"protected-files": map[string]any{
					"exclude": []any{"AGENTS.md"},
				},
			},
			wantExclude:   []string{"AGENTS.md"},
			wantPFPresent: false, // key removed when no policy
		},
		{
			name: "object form with empty policy string",
			configData: map[string]any{
				"protected-files": map[string]any{
					"policy":  "",
					"exclude": []any{"AGENTS.md"},
				},
			},
			wantExclude:   []string{"AGENTS.md"},
			wantPFPresent: false, // empty policy treated as absent
		},
		{
			name:          "nil configData returns nil",
			configData:    nil,
			wantExclude:   nil,
			wantPFPresent: false,
		},
		{
			name:          "absent field returns nil",
			configData:    map[string]any{"other": "value"},
			wantExclude:   nil,
			wantPFPresent: false,
		},
		{
			name: "object form with empty exclude list",
			configData: map[string]any{
				"protected-files": map[string]any{
					"policy":  "blocked",
					"exclude": []any{},
				},
			},
			wantExclude:   nil,
			wantPFAfter:   "blocked",
			wantPFPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessProtectedFilesField(tt.configData, nil)
			if len(tt.wantExclude) == 0 {
				assert.Empty(t, got, "exclude list should be empty/nil")
			} else {
				assert.Equal(t, tt.wantExclude, got, "exclude list should match")
			}

			if tt.configData == nil {
				return
			}
			pfVal, pfPresent := tt.configData["protected-files"]
			assert.Equal(t, tt.wantPFPresent, pfPresent, "protected-files presence should match")
			if tt.wantPFPresent {
				assert.Equal(t, tt.wantPFAfter, pfVal, "protected-files value should match after preprocessing")
			}
		})
	}
}

// TestValidateStringEnumField tests the validateStringEnumField helper.
func TestValidateStringEnumField(t *testing.T) {
	allowed := []string{"am", "bundle"}
	tests := []struct {
		name        string
		configData  map[string]any
		wantPresent bool
		wantValue   any
	}{
		{
			name:        "valid enum value is kept",
			configData:  map[string]any{"patch-format": "am"},
			wantPresent: true,
			wantValue:   "am",
		},
		{
			name:        "another valid enum value is kept",
			configData:  map[string]any{"patch-format": "bundle"},
			wantPresent: true,
			wantValue:   "bundle",
		},
		{
			name:        "invalid literal string is removed",
			configData:  map[string]any{"patch-format": "invalid"},
			wantPresent: false,
		},
		{
			name:        "non-string value is removed",
			configData:  map[string]any{"patch-format": 42},
			wantPresent: false,
		},
		{
			name:        "absent field is a no-op",
			configData:  map[string]any{"other": "value"},
			wantPresent: false,
		},
		{
			name:        "nil configData is a no-op",
			configData:  nil,
			wantPresent: false,
		},
		{
			name:        "GitHub Actions expression is passed through",
			configData:  map[string]any{"patch-format": "${{ inputs.patch-format }}"},
			wantPresent: true,
			wantValue:   "${{ inputs.patch-format }}",
		},
		{
			name:        "expression with spaces is passed through",
			configData:  map[string]any{"patch-format": "${{ inputs.format || 'am' }}"},
			wantPresent: true,
			wantValue:   "${{ inputs.format || 'am' }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validateStringEnumField(tt.configData, "patch-format", allowed, nil)
			if tt.configData == nil {
				return
			}
			val, present := tt.configData["patch-format"]
			assert.Equal(t, tt.wantPresent, present, "field presence should match")
			if tt.wantPresent {
				assert.Equal(t, tt.wantValue, val, "field value should match")
			}
		})
	}
}

// TestPreprocessProtectedFilesFieldWithExpression tests that GitHub Actions expressions
// in the object-form policy field pass through the preprocessing step unchanged.
func TestPreprocessProtectedFilesFieldWithExpression(t *testing.T) {
	tests := []struct {
		name          string
		configData    map[string]any
		wantExclude   []string
		wantPFAfter   any
		wantPFPresent bool
	}{
		{
			name: "expression string form passes through unchanged",
			configData: map[string]any{
				"protected-files": "${{ inputs.protected-files-policy }}",
			},
			wantExclude:   nil,
			wantPFAfter:   "${{ inputs.protected-files-policy }}",
			wantPFPresent: true,
		},
		{
			name: "object form with expression policy",
			configData: map[string]any{
				"protected-files": map[string]any{
					"policy":  "${{ inputs.policy }}",
					"exclude": []any{"AGENTS.md"},
				},
			},
			wantExclude:   []string{"AGENTS.md"},
			wantPFAfter:   "${{ inputs.policy }}",
			wantPFPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessProtectedFilesField(tt.configData, nil)
			if len(tt.wantExclude) == 0 {
				assert.Empty(t, got, "exclude list should be empty/nil")
			} else {
				assert.Equal(t, tt.wantExclude, got, "exclude list should match")
			}

			pfVal, pfPresent := tt.configData["protected-files"]
			assert.Equal(t, tt.wantPFPresent, pfPresent, "protected-files presence should match")
			if tt.wantPFPresent {
				assert.Equal(t, tt.wantPFAfter, pfVal, "protected-files value should match after preprocessing")
			}
		})
	}
}
