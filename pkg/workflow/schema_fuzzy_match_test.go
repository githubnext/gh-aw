package workflow

import (
	"errors"
	"strings"
	"testing"
)

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected int
	}{
		{
			name:     "identical strings",
			a:        "permissions",
			b:        "permissions",
			expected: 0,
		},
		{
			name:     "one character difference",
			a:        "permissions",
			b:        "permisions",
			expected: 1,
		},
		{
			name:     "engine typo",
			a:        "engine",
			b:        "engnie",
			expected: 2,
		},
		{
			name:     "tools typo",
			a:        "tools",
			b:        "toolz",
			expected: 1,
		},
		{
			name:     "timeout-minutes typo (underscore vs hyphen)",
			a:        "timeout-minutes",
			b:        "timeout_minutes",
			expected: 1,
		},
		{
			name:     "runs-on typo (underscore vs hyphen)",
			a:        "runs-on",
			b:        "runs_on",
			expected: 1,
		},
		{
			name:     "completely different strings",
			a:        "permissions",
			b:        "xyz",
			expected: 11,
		},
		{
			name:     "empty string a",
			a:        "",
			b:        "test",
			expected: 4,
		},
		{
			name:     "empty string b",
			a:        "test",
			b:        "",
			expected: 4,
		},
		{
			name:     "both empty",
			a:        "",
			b:        "",
			expected: 0,
		},
		{
			name:     "insertion needed",
			a:        "cat",
			b:        "cats",
			expected: 1,
		},
		{
			name:     "deletion needed",
			a:        "cats",
			b:        "cat",
			expected: 1,
		},
		{
			name:     "substitution needed",
			a:        "cat",
			b:        "bat",
			expected: 1,
		},
		{
			name:     "multiple operations",
			a:        "kitten",
			b:        "sitting",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := levenshteinDistance(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSuggestFieldName(t *testing.T) {
	validFields := []string{
		"on",
		"permissions",
		"engine",
		"tools",
		"timeout-minutes",
		"runs-on",
		"concurrency",
		"env",
		"steps",
		"if",
		"name",
	}

	tests := []struct {
		name         string
		invalidField string
		validFields  []string
		expected     []string
	}{
		{
			name:         "permisions - common typo",
			invalidField: "permisions",
			validFields:  validFields,
			expected:     []string{"permissions"}, // distance 1
		},
		{
			name:         "engnie - common typo",
			invalidField: "engnie",
			validFields:  validFields,
			expected:     []string{"engine"}, // distance 2
		},
		{
			name:         "toolz - common typo",
			invalidField: "toolz",
			validFields:  validFields,
			expected:     []string{"tools"}, // distance 1
		},
		{
			name:         "timeout_minutes - underscore instead of hyphen",
			invalidField: "timeout_minutes",
			validFields:  validFields,
			expected:     []string{"timeout-minutes"}, // distance 1
		},
		{
			name:         "runs_on - underscore instead of hyphen",
			invalidField: "runs_on",
			validFields:  validFields,
			expected:     []string{"runs-on"}, // distance 1
		},
		{
			name:         "exact match - no suggestions",
			invalidField: "permissions",
			validFields:  validFields,
			expected:     []string{}, // exact match, no suggestions
		},
		{
			name:         "very different string - some suggestions at distance 3",
			invalidField: "xyz",
			validFields:  validFields,
			expected:     []string{"env", "if", "on"}, // All have distance 3, sorted alphabetically
		},
		{
			name:         "multiple similar fields at distance 3",
			invalidField: "nam",
			validFields:  validFields,
			expected:     []string{"name", "env", "if", "on"}, // name is distance 1, others are distance 3
		},
		{
			name:         "distance 3 - should be included",
			invalidField: "permissio",
			validFields:  validFields,
			expected:     []string{"permissions"}, // distance 2 (missing 'ns')
		},
		{
			name:         "distance 4 - should not be included",
			invalidField: "permi",
			validFields:  validFields,
			expected:     []string{}, // distance 6, too far
		},
		{
			name:         "case insensitive matching",
			invalidField: "PERMISSIONS",
			validFields:  validFields,
			expected:     []string{}, // exact match when case-insensitive
		},
		{
			name:         "case insensitive typo",
			invalidField: "PERMISIONS",
			validFields:  validFields,
			expected:     []string{"permissions"}, // distance 1
		},
		{
			name:         "empty invalid field",
			invalidField: "",
			validFields:  validFields,
			expected:     []string{"if", "on", "env"}, // shortest fields: if, on (distance 2), env (distance 3)
		},
		{
			name:         "empty valid fields",
			invalidField: "test",
			validFields:  []string{},
			expected:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suggestFieldName(tt.invalidField, tt.validFields)
			
			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("suggestFieldName(%q) returned %d suggestions, want %d\nGot: %v\nWant: %v",
					tt.invalidField, len(result), len(tt.expected), result, tt.expected)
				return
			}

			// Check each suggestion
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("suggestFieldName(%q) suggestion[%d] = %q, want %q\nGot: %v\nWant: %v",
						tt.invalidField, i, result[i], tt.expected[i], result, tt.expected)
				}
			}
		})
	}
}

func TestSuggestFieldName_Sorting(t *testing.T) {
	validFields := []string{"alpha", "beta", "gamma"}

	// Test with a field that has equal distance to multiple candidates
	result := suggestFieldName("alpho", validFields)

	// "alpho" has distance 1 from "alpha"
	// Should return only "alpha"
	if len(result) != 1 {
		t.Errorf("Expected 1 suggestion, got %d: %v", len(result), result)
	}
	if len(result) > 0 && result[0] != "alpha" {
		t.Errorf("Expected 'alpha', got %q", result[0])
	}
}

func TestEnhanceSchemaValidationError(t *testing.T) {
	schema := map[string]interface{}{
		"properties": map[string]interface{}{
			"on":          map[string]interface{}{},
			"permissions": map[string]interface{}{},
			"engine":      map[string]interface{}{},
			"tools":       map[string]interface{}{},
		},
	}

	tests := []struct {
		name        string
		err         error
		schema      map[string]interface{}
		wantContain string
		wantNil     bool
	}{
		{
			name:        "nil error",
			err:         nil,
			schema:      schema,
			wantNil:     true,
			wantContain: "",
		},
		{
			name:        "non-schema error",
			err:         errors.New("some other error"),
			schema:      schema,
			wantNil:     false,
			wantContain: "some other error",
		},
		{
			name:        "unknown property with close match",
			err:         errors.New("Unknown property: permisions"),
			schema:      schema,
			wantNil:     false,
			wantContain: "Did you mean 'permissions'?",
		},
		{
			name:        "unknown property with multiple matches at distance 2",
			err:         errors.New("Unknown property: too"),
			schema:      schema,
			wantNil:     false,
			wantContain: "Did you mean one of:", // Two fields at distance 2
		},
		{
			name:        "unknown property with no close matches",
			err:         errors.New("Unknown property: xyz"),
			schema:      schema,
			wantNil:     false,
			wantContain: "Unknown property: xyz", // No enhancement if no matches
		},
		{
			name:        "unknown properties plural",
			err:         errors.New("Unknown properties: engnie, toolz"),
			schema:      schema,
			wantNil:     false,
			wantContain: "Did you mean 'engine'?", // Should suggest for first field
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := enhanceSchemaValidationError(tt.err, tt.schema)

			if tt.wantNil {
				if result != nil {
					t.Errorf("Expected nil error, got: %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("Expected non-nil error, got nil")
				return
			}

			errorMsg := result.Error()
			if tt.wantContain != "" && !strings.Contains(errorMsg, tt.wantContain) {
				t.Errorf("Expected error to contain %q, got: %q", tt.wantContain, errorMsg)
			}
		})
	}
}

func TestExtractInvalidFieldName(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg string
		expected string
	}{
		{
			name:     "unknown property singular",
			errorMsg: "Unknown property: permisions",
			expected: "permisions",
		},
		{
			name:     "unknown property with period",
			errorMsg: "Unknown property: engnie. Valid fields are: ...",
			expected: "engnie",
		},
		{
			name:     "unknown properties plural",
			errorMsg: "Unknown properties: toolz, xyz",
			expected: "toolz",
		},
		{
			name:     "no match",
			errorMsg: "Some other error message",
			expected: "",
		},
		{
			name:     "unknown property at end",
			errorMsg: "Unknown property: test",
			expected: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractInvalidFieldName(tt.errorMsg)
			if result != tt.expected {
				t.Errorf("extractInvalidFieldName(%q) = %q, want %q", tt.errorMsg, result, tt.expected)
			}
		})
	}
}

func TestExtractValidFieldsFromSchema(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected []string
	}{
		{
			name: "valid schema with properties",
			schema: map[string]interface{}{
				"properties": map[string]interface{}{
					"on":          map[string]interface{}{},
					"permissions": map[string]interface{}{},
					"engine":      map[string]interface{}{},
				},
			},
			expected: []string{"engine", "on", "permissions"}, // sorted
		},
		{
			name:     "nil schema",
			schema:   nil,
			expected: nil,
		},
		{
			name:     "empty schema",
			schema:   map[string]interface{}{},
			expected: nil,
		},
		{
			name: "schema without properties",
			schema: map[string]interface{}{
				"type": "object",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractValidFieldsFromSchema(tt.schema)

			if len(result) != len(tt.expected) {
				t.Errorf("extractValidFieldsFromSchema() returned %d fields, want %d\nGot: %v\nWant: %v",
					len(result), len(tt.expected), result, tt.expected)
				return
			}

			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("extractValidFieldsFromSchema() field[%d] = %q, want %q",
						i, result[i], tt.expected[i])
				}
			}
		})
	}
}
