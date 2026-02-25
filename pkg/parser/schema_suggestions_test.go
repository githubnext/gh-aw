//go:build !integration

package parser

import (
	"encoding/json"
	"slices"
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
		name               string
		errorMessage       string
		jsonPath           string
		frontmatterContent string
		wantContains       []string
		wantEmpty          bool
	}{
		{
			name:         "additional property error at root level",
			errorMessage: "additional property 'naem' not allowed", // Typo for "name" (distance 2)
			jsonPath:     "",
			wantContains: []string{"Did you mean", "name"}, // Close match found
		},
		{
			name:         "additional property error in nested object",
			errorMessage: "additional property 'contnt' not allowed", // Typo for "contents" (distance 1)
			jsonPath:     "/permissions",
			wantContains: []string{"Did you mean", "contents"}, // Close match found
		},
		{
			name:         "type error with integer expected",
			errorMessage: "got string, want integer",
			jsonPath:     "/age",
			wantContains: []string{"Expected format:", "42"},
		},
		{
			name:         "multiple additional properties",
			errorMessage: "additional properties 'prop1', 'prop2' not allowed", // Far from any valid field
			jsonPath:     "",
			wantContains: []string{"Valid fields are:", "name", "age"}, // No close matches, show valid fields
		},
		{
			name:         "non-validation error",
			errorMessage: "some other error",
			jsonPath:     "",
			wantEmpty:    true,
		},
		{
			name:               "enum violation with close match suggests Did you mean",
			errorMessage:       "value must be one of 'claude', 'codex', 'copilot', 'gemini'",
			jsonPath:           "/engine",
			frontmatterContent: "engine: coplit\n",
			wantContains:       []string{"Did you mean", "copilot"},
		},
		{
			name:               "enum violation with no user value returns empty",
			errorMessage:       "value must be one of 'claude', 'codex', 'copilot', 'gemini'",
			jsonPath:           "/engine",
			frontmatterContent: "",
			wantEmpty:          true,
		},
		{
			name:               "enum violation with no close match returns empty",
			errorMessage:       "value must be one of 'claude', 'codex', 'copilot', 'gemini'",
			jsonPath:           "/engine",
			frontmatterContent: "engine: xyz123\n",
			wantEmpty:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaBasedSuggestions(schemaJSON, tt.errorMessage, tt.jsonPath, tt.frontmatterContent)

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
			name:           "single invalid property with close matches",
			invalidProps:   []string{"contnt"},
			acceptedFields: []string{"content", "contents", "name"},
			wantContains:   []string{"Did you mean:", "content"}, // Returns suggestions including "content"
		},
		{
			name:           "multiple invalid properties",
			invalidProps:   []string{"prop1", "prop2"},
			acceptedFields: []string{"name", "age", "email"},
			wantContains:   []string{"Valid fields are:", "name", "age", "email"}, // No close matches, show all
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
			name:       "exact match skipped - returns next closest",
			target:     "content",
			maxResults: 3,
			wantFirst:  "contents", // Exact match is skipped, "contents" has distance 1
		},
		{
			name:       "partial match",
			target:     "contnt",
			maxResults: 2,
			wantFirst:  "content", // "content" has distance 1
		},
		{
			name:       "prefix match",
			target:     "time",
			maxResults: 1,
			wantFirst:  "name", // "name" has distance 2, closer than "timeout" (distance 3)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindClosestMatches(tt.target, candidates, tt.maxResults)

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

func TestExtractEnumValuesFromError(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "engine enum values",
			input: "value must be one of 'claude', 'codex', 'copilot', 'gemini'",
			want:  []string{"claude", "codex", "copilot", "gemini"},
		},
		{
			name:  "permissions enum values",
			input: "value must be one of 'read', 'write', 'none'",
			want:  []string{"read", "write", "none"},
		},
		{
			name:  "no single-quoted values",
			input: "some other error message",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEnumValuesFromError(tt.input)
			if len(result) != len(tt.want) {
				t.Errorf("Expected %v, got %v", tt.want, result)
				return
			}
			for i, v := range tt.want {
				if result[i] != v {
					t.Errorf("Expected result[%d]=%q, got %q", i, v, result[i])
				}
			}
		})
	}
}

func TestExtractYAMLValueAtPath(t *testing.T) {
	tests := []struct {
		name      string
		yaml      string
		path      string
		wantValue string
	}{
		{
			name:      "simple top-level field",
			yaml:      "engine: coplit\ntimeout-minutes: 30\n",
			path:      "/engine",
			wantValue: "coplit",
		},
		{
			name:      "field with double-quoted value",
			yaml:      `engine: "copilot"` + "\n",
			path:      "/engine",
			wantValue: "copilot",
		},
		{
			name:      "field with single-quoted value",
			yaml:      "engine: 'copilot'\n",
			path:      "/engine",
			wantValue: "copilot",
		},
		{
			name:      "nested path returns empty",
			yaml:      "engine: copilot\n",
			path:      "/permissions/issues",
			wantValue: "",
		},
		{
			name:      "empty yaml returns empty",
			yaml:      "",
			path:      "/engine",
			wantValue: "",
		},
		{
			name:      "field not present returns empty",
			yaml:      "engine: copilot\n",
			path:      "/timeout-minutes",
			wantValue: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractYAMLValueAtPath(tt.yaml, tt.path)
			if result != tt.wantValue {
				t.Errorf("extractYAMLValueAtPath(%q, %q) = %q, want %q", tt.yaml, tt.path, result, tt.wantValue)
			}
		})
	}
}

// TestGenerateExampleFromSchemaWithExamples tests that schema examples array is preferred over generic fallback
func TestGenerateExampleFromSchemaWithExamples(t *testing.T) {
	tests := []struct {
		name         string
		schema       map[string]any
		wantContains string
	}{
		{
			name: "integer with examples uses first example",
			schema: map[string]any{
				"type":     "integer",
				"minimum":  float64(1),
				"examples": []any{float64(5), float64(10), float64(30)},
			},
			wantContains: "5",
		},
		{
			name: "integer with default uses default",
			schema: map[string]any{
				"type":    "integer",
				"default": float64(20),
			},
			wantContains: "20",
		},
		{
			name: "integer without examples falls back to 42",
			schema: map[string]any{
				"type": "integer",
			},
			wantContains: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateExampleFromSchema(tt.schema)
			if result == nil {
				t.Errorf("Expected non-nil result")
				return
			}
			exampleJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("Failed to marshal result: %v", err)
			}
			if !strings.Contains(string(exampleJSON), tt.wantContains) {
				t.Errorf("Expected result to contain %q, got %s", tt.wantContains, string(exampleJSON))
			}
		})
	}
}

// schemaWithNestedFields is a test schema where "skip-if-match" lives under "on", not at the root or under "actions".
var schemaWithNestedFields = `{
	"type": "object",
	"properties": {
		"on": {
			"type": "object",
			"properties": {
				"skip-if-match": {"type": "string"},
				"skip-if-no-match": {"type": "string"}
			},
			"additionalProperties": false
		},
		"actions": {
			"type": "object",
			"properties": {
				"body": {"type": "string"},
				"branch": {"type": "string"}
			},
			"additionalProperties": false
		},
		"engine": {"type": "string"},
		"permissions": {
			"type": "object",
			"properties": {
				"contents": {"type": "string"},
				"issues": {"type": "string"}
			},
			"additionalProperties": false
		}
	},
	"additionalProperties": false
}`

func TestCollectSchemaPropertyPaths(t *testing.T) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	locations := collectSchemaPropertyPaths(schemaDoc, "", 0)

	// Build a map of field -> parent paths for easy assertions
	fieldPaths := make(map[string][]string)
	for _, loc := range locations {
		fieldPaths[loc.FieldName] = append(fieldPaths[loc.FieldName], loc.SchemaPath)
	}

	tests := []struct {
		fieldName      string
		wantParentPath string
	}{
		{"engine", ""},
		{"permissions", ""},
		{"on", ""},
		{"skip-if-match", "/on"},
		{"skip-if-no-match", "/on"},
		{"body", "/actions"},
		{"branch", "/actions"},
		{"contents", "/permissions"},
		{"issues", "/permissions"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			paths, ok := fieldPaths[tt.fieldName]
			if !ok {
				t.Errorf("Field %q not found in collected paths", tt.fieldName)
				return
			}
			if !slices.Contains(paths, tt.wantParentPath) {
				t.Errorf("Field %q: expected parent path %q, got paths: %v", tt.fieldName, tt.wantParentPath, paths)
			}
		})
	}
}

func TestFindFieldLocationsInSchema(t *testing.T) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	tests := []struct {
		name        string
		targetField string
		currentPath string
		wantLen     int
		wantPath    string // expected SchemaPath of first result (if any)
		wantField   string // expected FieldName (for fuzzy matches)
		wantDist    int    // expected Distance
	}{
		{
			name:        "exact match found at different path",
			targetField: "skip-if-match",
			currentPath: "/actions",
			wantLen:     1,
			wantPath:    "/on",
			wantField:   "skip-if-match",
			wantDist:    0,
		},
		{
			name:        "fuzzy match found at different path",
			targetField: "skip-if-mach",
			currentPath: "/actions",
			wantLen:     1,
			wantPath:    "/on",
			wantField:   "skip-if-match",
			wantDist:    1,
		},
		{
			name:        "same path excluded from results",
			targetField: "skip-if-match",
			currentPath: "/on",
			wantLen:     0,
		},
		{
			name:        "field not in schema returns empty",
			targetField: "totally-unknown-field",
			currentPath: "",
			wantLen:     0,
		},
		{
			name:        "field too far for fuzzy match",
			targetField: "abcdefghij",
			currentPath: "",
			wantLen:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := findFieldLocationsInSchema(schemaDoc, tt.targetField, tt.currentPath)
			if len(results) != tt.wantLen {
				t.Errorf("Expected %d results, got %d: %+v", tt.wantLen, len(results), results)
				return
			}
			if tt.wantLen == 0 {
				return
			}
			if results[0].SchemaPath != tt.wantPath {
				t.Errorf("Expected SchemaPath %q, got %q", tt.wantPath, results[0].SchemaPath)
			}
			if results[0].FieldName != tt.wantField {
				t.Errorf("Expected FieldName %q, got %q", tt.wantField, results[0].FieldName)
			}
			if results[0].Distance != tt.wantDist {
				t.Errorf("Expected Distance %d, got %d", tt.wantDist, results[0].Distance)
			}
		})
	}
}

func TestGeneratePathLocationSuggestion(t *testing.T) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc); err != nil {
		t.Fatalf("Failed to unmarshal schema: %v", err)
	}

	tests := []struct {
		name         string
		invalidProps []string
		currentPath  string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "exact match suggests correct parent path",
			invalidProps: []string{"skip-if-match"},
			currentPath:  "/actions",
			wantContains: []string{"skip-if-match", "belongs under", "on"},
		},
		{
			name:         "fuzzy match suggests field name and path",
			invalidProps: []string{"skip-if-mach"},
			currentPath:  "/actions",
			wantContains: []string{"Did you mean", "skip-if-match", "belongs under", "on"},
		},
		{
			name:         "field valid at current path excluded",
			invalidProps: []string{"skip-if-match"},
			currentPath:  "/on",
			wantEmpty:    true,
		},
		{
			name:         "field not in schema returns empty",
			invalidProps: []string{"completely-unknown"},
			currentPath:  "",
			wantEmpty:    true,
		},
		{
			name:         "empty invalid props returns empty",
			invalidProps: []string{},
			currentPath:  "",
			wantEmpty:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generatePathLocationSuggestion(tt.invalidProps, schemaDoc, tt.currentPath)
			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty result, got: %q", result)
				}
				return
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q, got: %q", want, result)
				}
			}
		})
	}
}

func TestGenerateSchemaBasedSuggestionsWithPathHeuristic(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:         "misplaced field suggests correct path",
			errorMessage: "additional property 'skip-if-match' not allowed",
			jsonPath:     "/actions",
			wantContains: []string{"skip-if-match", "belongs under", "on"},
		},
		{
			name:         "fuzzy misplaced field suggests field name and path",
			errorMessage: "additional property 'skip-if-mach' not allowed",
			jsonPath:     "/actions",
			wantContains: []string{"Did you mean", "skip-if-match", "on"},
		},
		{
			name:         "typo at correct location does not suggest path",
			errorMessage: "additional property 'isues' not allowed",
			jsonPath:     "/permissions",
			// Should suggest 'issues' (typo fix), not path suggestion for same location
			wantContains: []string{"Did you mean", "issues"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaBasedSuggestions(schemaWithNestedFields, tt.errorMessage, tt.jsonPath, "")
			if tt.wantEmpty {
				if result != "" {
					t.Errorf("Expected empty result, got: %q", result)
				}
				return
			}
			for _, want := range tt.wantContains {
				if !strings.Contains(result, want) {
					t.Errorf("Expected result to contain %q, got: %q", want, result)
				}
			}
		})
	}
}
