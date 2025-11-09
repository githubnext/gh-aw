package workflow

import (
	"strings"
	"testing"
)

func TestValidateBashToolValue(t *testing.T) {
	tests := []struct {
		name        string
		bashValue   any
		shouldError bool
		errorMsg    string
		description string
	}{
		{
			name:        "nil value allows all bash commands",
			bashValue:   nil,
			shouldError: false,
			description: "nil should be valid and allow all bash commands",
		},
		{
			name:        "true allows all bash commands",
			bashValue:   true,
			shouldError: false,
			description: "boolean true should be valid and allow all bash commands",
		},
		{
			name:        "false disables bash",
			bashValue:   false,
			shouldError: false,
			description: "boolean false should be valid and disable bash",
		},
		{
			name:        "empty array is valid",
			bashValue:   []any{},
			shouldError: false,
			description: "empty array should be valid (no commands allowed)",
		},
		{
			name:        "array with valid string commands",
			bashValue:   []any{"git:*", "npm:*", "make:*"},
			shouldError: false,
			description: "array of string commands should be valid",
		},
		{
			name:        "array with single command",
			bashValue:   []any{"*"},
			shouldError: false,
			description: "array with wildcard should be valid",
		},
		{
			name:        "array with non-string element",
			bashValue:   []any{"git:*", 123, "npm:*"},
			shouldError: true,
			errorMsg:    "not a string",
			description: "array with non-string element should fail",
		},
		{
			name:        "array with boolean element",
			bashValue:   []any{"git:*", true},
			shouldError: true,
			errorMsg:    "not a string",
			description: "array with boolean element should fail",
		},
		{
			name:        "string value is invalid",
			bashValue:   "git:*",
			shouldError: true,
			errorMsg:    "must be null, boolean, or array",
			description: "string value should fail with helpful message",
		},
		{
			name:        "number value is invalid",
			bashValue:   123,
			shouldError: true,
			errorMsg:    "must be null, boolean, or array",
			description: "number value should fail with helpful message",
		},
		{
			name:        "object value is invalid",
			bashValue:   map[string]any{"commands": []string{"git:*"}},
			shouldError: true,
			errorMsg:    "must be null, boolean, or array",
			description: "object value should fail with helpful message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateBashToolValue(tt.bashValue)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for %s but got none", tt.description)
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Error message should contain '%s', got: %s", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for %s: %v", tt.description, err)
					return
				}
				// Verify result type for valid inputs
				if tt.bashValue == nil || tt.bashValue == true {
					if result != nil {
						t.Errorf("Expected nil result for nil/true input, got: %v", result)
					}
				} else if tt.bashValue == false {
					if arr, ok := result.([]any); !ok || len(arr) != 0 {
						t.Errorf("Expected empty array for false input, got: %v", result)
					}
				} else if bashArray, ok := tt.bashValue.([]any); ok {
					resultArray, ok := result.([]any)
					if !ok {
						t.Errorf("Expected array result for array input, got: %T", result)
					} else if len(resultArray) != len(bashArray) {
						t.Errorf("Expected array length %d, got %d", len(bashArray), len(resultArray))
					}
				}
			}
		})
	}
}

func TestValidateBashToolValueErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		bashValue     any
		expectedParts []string
	}{
		{
			name:      "invalid type shows examples",
			bashValue: "git:*",
			expectedParts: []string{
				"must be null, boolean, or array",
				"bash: true",
				"bash: [\"git:*\", \"npm:*\"]",
				"bash: false",
			},
		},
		{
			name:      "non-string array element shows index",
			bashValue: []any{"git:*", 123},
			expectedParts: []string{
				"index 1",
				"not a string",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateBashToolValue(tt.bashValue)
			if err == nil {
				t.Fatalf("Expected error but got none")
			}

			errMsg := err.Error()
			for _, part := range tt.expectedParts {
				if !strings.Contains(errMsg, part) {
					t.Errorf("Error message should contain '%s'\nGot: %s", part, errMsg)
				}
			}
		})
	}
}
