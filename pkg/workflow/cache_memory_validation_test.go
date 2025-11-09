package workflow

import (
	"strings"
	"testing"
)

func TestValidateCacheMemoryToolValue(t *testing.T) {
	tests := []struct {
		name        string
		cacheValue  any
		shouldError bool
		errorMsg    string
		description string
	}{
		{
			name:        "nil value creates default cache",
			cacheValue:  nil,
			shouldError: false,
			description: "nil should create single default cache",
		},
		{
			name:        "true creates default cache",
			cacheValue:  true,
			shouldError: false,
			description: "boolean true should create single default cache",
		},
		{
			name:        "false disables cache-memory",
			cacheValue:  false,
			shouldError: false,
			description: "boolean false should disable cache-memory (no caches)",
		},
		{
			name: "valid object with all fields",
			cacheValue: map[string]any{
				"id":             "custom",
				"key":            "my-key",
				"description":    "Test cache",
				"retention-days": 7,
			},
			shouldError: false,
			description: "object with all valid fields should succeed",
		},
		{
			name:        "empty object is valid",
			cacheValue:  map[string]any{},
			shouldError: false,
			description: "empty object should use defaults",
		},
		{
			name: "object with unknown field",
			cacheValue: map[string]any{
				"id":      "test",
				"invalid": "field",
			},
			shouldError: true,
			errorMsg:    "unknown field 'invalid'",
			description: "object with unknown field should fail",
		},
		{
			name: "object with invalid id type",
			cacheValue: map[string]any{
				"id": 123,
			},
			shouldError: true,
			errorMsg:    "'id' field must be a string",
			description: "object with non-string id should fail",
		},
		{
			name: "object with invalid key type",
			cacheValue: map[string]any{
				"key": 123,
			},
			shouldError: true,
			errorMsg:    "'key' field must be a string",
			description: "object with non-string key should fail",
		},
		{
			name: "object with invalid description type",
			cacheValue: map[string]any{
				"description": 123,
			},
			shouldError: true,
			errorMsg:    "'description' field must be a string",
			description: "object with non-string description should fail",
		},
		{
			name: "object with invalid retention-days type",
			cacheValue: map[string]any{
				"retention-days": "7",
			},
			shouldError: true,
			errorMsg:    "'retention-days' must be a number",
			description: "object with non-number retention-days should fail",
		},
		{
			name: "object with retention-days below range",
			cacheValue: map[string]any{
				"retention-days": 0,
			},
			shouldError: true,
			errorMsg:    "must be between 1 and 90 days",
			description: "retention-days below 1 should fail",
		},
		{
			name: "object with retention-days above range",
			cacheValue: map[string]any{
				"retention-days": 91,
			},
			shouldError: true,
			errorMsg:    "must be between 1 and 90 days",
			description: "retention-days above 90 should fail",
		},
		{
			name: "object with valid retention-days at boundaries",
			cacheValue: map[string]any{
				"retention-days": 1,
			},
			shouldError: false,
			description: "retention-days of 1 should succeed",
		},
		{
			name: "object with valid retention-days float",
			cacheValue: map[string]any{
				"retention-days": 7.0,
			},
			shouldError: false,
			description: "retention-days as float should succeed",
		},
		{
			name: "valid array with single cache",
			cacheValue: []any{
				map[string]any{
					"id":  "default",
					"key": "custom-key",
				},
			},
			shouldError: false,
			description: "array with single valid cache should succeed",
		},
		{
			name: "valid array with multiple caches",
			cacheValue: []any{
				map[string]any{"id": "default"},
				map[string]any{"id": "session"},
			},
			shouldError: false,
			description: "array with multiple valid caches should succeed",
		},
		{
			name:        "empty array is invalid",
			cacheValue:  []any{},
			shouldError: true,
			errorMsg:    "cannot be empty",
			description: "empty array should fail",
		},
		{
			name:        "array with non-object element",
			cacheValue:  []any{"string"},
			shouldError: true,
			errorMsg:    "must be an object",
			description: "array with non-object element should fail",
		},
		{
			name: "array with duplicate IDs",
			cacheValue: []any{
				map[string]any{"id": "duplicate"},
				map[string]any{"id": "duplicate"},
			},
			shouldError: true,
			errorMsg:    "duplicate cache-memory ID 'duplicate'",
			description: "array with duplicate IDs should fail",
		},
		{
			name: "array element with invalid id type",
			cacheValue: []any{
				map[string]any{"id": 123},
			},
			shouldError: true,
			errorMsg:    "cache-memory[0].id must be a string",
			description: "array element with non-string id should fail",
		},
		{
			name: "array element with invalid key type",
			cacheValue: []any{
				map[string]any{"key": 123},
			},
			shouldError: true,
			errorMsg:    "cache-memory[0].key must be a string",
			description: "array element with non-string key should fail",
		},
		{
			name: "array element with retention-days out of range",
			cacheValue: []any{
				map[string]any{
					"id":             "test",
					"retention-days": 100,
				},
			},
			shouldError: true,
			errorMsg:    "cache-memory[0].retention-days must be between 1 and 90",
			description: "array element with out-of-range retention-days should fail",
		},
		{
			name: "array element with unknown field",
			cacheValue: []any{
				map[string]any{
					"id":      "test",
					"unknown": "field",
				},
			},
			shouldError: true,
			errorMsg:    "cache-memory[0]: unknown field 'unknown'",
			description: "array element with unknown field should fail",
		},
		{
			name:        "string value is invalid",
			cacheValue:  "invalid",
			shouldError: true,
			errorMsg:    "must be null, boolean, object, or array",
			description: "string value should fail",
		},
		{
			name:        "number value is invalid",
			cacheValue:  123,
			shouldError: true,
			errorMsg:    "must be null, boolean, object, or array",
			description: "number value should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := validateCacheMemoryToolValue(tt.cacheValue)

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
				if config == nil {
					t.Errorf("Expected non-nil config for valid input")
					return
				}

				// Verify result based on input type
				if tt.cacheValue == nil || tt.cacheValue == true {
					if len(config.Caches) != 1 {
						t.Errorf("Expected 1 cache for nil/true, got %d", len(config.Caches))
					}
					if len(config.Caches) > 0 && config.Caches[0].ID != "default" {
						t.Errorf("Expected default cache ID, got %s", config.Caches[0].ID)
					}
				} else if tt.cacheValue == false {
					if len(config.Caches) != 0 {
						t.Errorf("Expected 0 caches for false, got %d", len(config.Caches))
					}
				}
			}
		})
	}
}

func TestValidateCacheMemoryToolValueKeyAppending(t *testing.T) {
	tests := []struct {
		name        string
		cacheValue  any
		expectKey   string
		description string
	}{
		{
			name: "key without run_id gets it appended",
			cacheValue: map[string]any{
				"key": "my-key",
			},
			expectKey:   "my-key-${{ github.run_id }}",
			description: "should append run_id suffix",
		},
		{
			name: "key with run_id is not duplicated",
			cacheValue: map[string]any{
				"key": "my-key-${{ github.run_id }}",
			},
			expectKey:   "my-key-${{ github.run_id }}",
			description: "should not duplicate run_id suffix",
		},
		{
			name: "array element key without run_id",
			cacheValue: []any{
				map[string]any{
					"id":  "test",
					"key": "test-key",
				},
			},
			expectKey:   "test-key-${{ github.run_id }}",
			description: "array element should get run_id appended",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := validateCacheMemoryToolValue(tt.cacheValue)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(config.Caches) == 0 {
				t.Fatalf("Expected at least one cache")
			}

			if config.Caches[0].Key != tt.expectKey {
				t.Errorf("Expected key '%s', got '%s'", tt.expectKey, config.Caches[0].Key)
			}
		})
	}
}

func TestValidateCacheMemoryToolValueErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		cacheValue    any
		expectedParts []string
	}{
		{
			name:       "invalid type shows examples",
			cacheValue: "string",
			expectedParts: []string{
				"must be null, boolean, object, or array",
				"cache-memory: true",
				"cache-memory:",
				"id: default",
			},
		},
		{
			name:       "empty array shows helpful message",
			cacheValue: []any{},
			expectedParts: []string{
				"cannot be empty",
				"Use 'false' to disable",
			},
		},
		{
			name: "unknown field lists valid fields",
			cacheValue: map[string]any{
				"invalid": "field",
			},
			expectedParts: []string{
				"unknown field 'invalid'",
				"Valid fields",
				"id",
				"key",
				"description",
				"retention-days",
			},
		},
		{
			name: "retention-days out of range shows bounds",
			cacheValue: map[string]any{
				"retention-days": 100,
			},
			expectedParts: []string{
				"must be between 1 and 90 days",
				"got 100",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateCacheMemoryToolValue(tt.cacheValue)
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
