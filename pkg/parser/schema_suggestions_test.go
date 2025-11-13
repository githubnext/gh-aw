package parser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSchemaBasedSuggestions(t *testing.T) {
	// Sample schema JSON for testing
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"email": {"type": "string"},
			"permissions": {
				"type": "object",
				"properties": {
					"contents": {"type": "string"},
					"issues": {"type": "string"},
					"pull-requests": {"type": "string"}
				},
				"additionalProperties": false
			}
		},
		"additionalProperties": false
	}`

	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "additional property error at root level",
			errorMessage: "additional property 'nam' not allowed", // typo of 'name'
			jsonPath:     "",
			wantContains: []string{"Did you mean:", "Valid fields:", "name"},
		},
		{
			name:         "additional property error in nested object",
			errorMessage: "additional property 'content' not allowed", // close to 'contents'
			jsonPath:     "/permissions",
			wantContains: []string{"Did you mean:", "Valid fields:", "contents"},
		},
		{
			name:         "type error with integer expected",
			errorMessage: "got string, want integer",
			jsonPath:     "/age",
			wantContains: []string{"Expected format:", "42"},
		},
		{
			name:         "multiple additional properties",
			errorMessage: "additional properties 'nme', 'ag' not allowed", // typos of 'name' and 'age'
			jsonPath:     "",
			wantContains: []string{"Did you mean one of:", "Valid fields:", "name", "age"},
		},
		{
			name:         "non-validation error",
			errorMessage: "some other error",
			jsonPath:     "",
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaBasedSuggestions(schemaJSON, tt.errorMessage, tt.jsonPath)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain '%s', got: %s", want, result)
				}
			}
		})
	}
}

func TestExtractAcceptedFieldsFromSchema(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"permissions": {
				"type": "object",
				"properties": {
					"contents": {"type": "string"},
					"issues": {"type": "string"}
				}
			}
		}
	}`

	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	tests := []struct {
		name     string
		jsonPath string
		want     []string
	}{
		{
			name:     "root level fields",
			jsonPath: "",
			want:     []string{"age", "name", "permissions"}, // sorted
		},
		{
			name:     "nested object fields",
			jsonPath: "/permissions",
			want:     []string{"contents", "issues"}, // sorted
		},
		{
			name:     "non-existent path",
			jsonPath: "/nonexistent",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAcceptedFieldsFromSchema(schemaDoc, tt.jsonPath)

			if len(result) != len(tt.want) {
				t.Errorf("Expected %d fields, got %d: %v", len(tt.want), len(result), result)
				return
			}

			for i, field := range tt.want {
				if i >= len(result) || result[i] != field {
					t.Errorf("Expected field[%d] = %s, got %v", i, field, result)
				}
			}
		})
	}
}

func TestGenerateFieldSuggestions(t *testing.T) {
	tests := []struct {
		name           string
		invalidProps   []string
		acceptedFields []string
		wantContains   []string
	}{
		{
			name:           "single invalid property with close match",
			invalidProps:   []string{"contnt"},
			acceptedFields: []string{"content", "contents", "name"},
			wantContains:   []string{"Did you mean:", "Valid fields:", "content"},
		},
		{
			name:           "multiple invalid properties",
			invalidProps:   []string{"prop1", "prop2"},
			acceptedFields: []string{"name", "age", "email"},
			wantContains:   []string{"Valid fields:", "name", "age", "email"},
		},
		{
			name:           "no accepted fields",
			invalidProps:   []string{"invalid"},
			acceptedFields: []string{},
			wantContains:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateFieldSuggestions(tt.invalidProps, tt.acceptedFields)

			if len(tt.wantContains) == 0 {
				if result != "" {
					t.Errorf("Expected empty result, got: %s", result)
				}
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain '%s', got: %s", want, result)
				}
			}
		})
	}
}

func TestFindClosestMatches(t *testing.T) {
	candidates := []string{"content", "contents", "name", "age", "permissions", "timeout"}

	tests := []struct {
		name       string
		target     string
		maxResults int
		wantFirst  string // First result should be this
	}{
		{
			name:       "exact substring match",
			target:     "content",
			maxResults: 3,
			wantFirst:  "content",
		},
		{
			name:       "partial match",
			target:     "contnt",
			maxResults: 2,
			wantFirst:  "content",
		},
		{
			name:       "prefix match",
			target:     "time",
			maxResults: 1,
			wantFirst:  "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findClosestMatches(tt.target, candidates, tt.maxResults)

			if len(result) == 0 {
				t.Errorf("Expected at least one match, got none")
				return
			}

			if len(result) > tt.maxResults {
				t.Errorf("Expected at most %d results, got %d", tt.maxResults, len(result))
			}

			if result[0] != tt.wantFirst {
				t.Errorf("Expected first result to be '%s', got '%s'", tt.wantFirst, result[0])
			}
		})
	}
}

func TestGenerateExampleJSONForPath(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"timeout_minutes": {"type": "integer"},
			"name": {"type": "string"},
			"active": {"type": "boolean"},
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			},
			"config": {
				"type": "object",
				"properties": {
					"enabled": {"type": "boolean"},
					"count": {"type": "integer"}
				}
			}
		}
	}`

	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaJSON), &schemaDoc); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	tests := []struct {
		name         string
		jsonPath     string
		wantContains []string
	}{
		{
			name:         "integer field",
			jsonPath:     "/timeout_minutes",
			wantContains: []string{"42"},
		},
		{
			name:         "string field",
			jsonPath:     "/name",
			wantContains: []string{`"string"`},
		},
		{
			name:         "boolean field",
			jsonPath:     "/active",
			wantContains: []string{"true"},
		},
		{
			name:         "array field",
			jsonPath:     "/tags",
			wantContains: []string{"[", `"string"`, "]"},
		},
		{
			name:         "object field",
			jsonPath:     "/config",
			wantContains: []string{"{", "}", "enabled", "count"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleJSONForPath(schemaDoc, tt.jsonPath)

			if result == "" {
				t.Errorf("Expected non-empty result for path %s", tt.jsonPath)
				return
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain '%s', got: %s", want, result)
				}
			}
		})
	}
}
