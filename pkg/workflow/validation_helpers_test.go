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
