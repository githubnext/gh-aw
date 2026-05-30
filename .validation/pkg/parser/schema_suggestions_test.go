//go:build !integration

package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{
			// Full end-to-end: path is the oneOf container, message contains nested path,
			// frontmatter has a permission level typo.
			name:               "nested oneOf enum violation extracts sub-path and suggests Did you mean",
			errorMessage:       "'oneOf' failed, none matched\n  - at '/permissions': got object, want string\n  - at '/permissions/contents': value must be one of 'read', 'write', 'none'",
			jsonPath:           "/permissions",
			frontmatterContent: "permissions:\n  contents: raed\n",
			wantContains:       []string{"Did you mean", "read"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaBasedSuggestions(schemaJSON, tt.errorMessage, tt.jsonPath, tt.frontmatterContent)

			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty suggestion result")
				return
			}

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Suggestion result should contain %q", want)
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
	require.NoError(t, json.Unmarshal([]byte(schemaJSON), &schemaDoc), "Failed to unmarshal schema")

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

			assert.Equal(t, tt.want, result, "Accepted fields should match expected list for path %q", tt.jsonPath)
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
				assert.Empty(t, result, "Expected empty suggestion result when no accepted fields")
				return
			}

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Field suggestion result should contain %q", want)
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

			require.NotEmpty(t, result, "FindClosestMatches should return at least one match for %q", tt.target)
			assert.LessOrEqual(t, len(result), tt.maxResults, "FindClosestMatches should return at most %d results", tt.maxResults)
			assert.Equal(t, tt.wantFirst, result[0], "First closest match for %q should be %q", tt.target, tt.wantFirst)
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
	require.NoError(t, json.Unmarshal([]byte(schemaJSON), &schemaDoc), "Failed to unmarshal schema")

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

			require.NotEmpty(t, result, "Expected non-empty example JSON for path %q", tt.jsonPath)

			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Example JSON for path %q should contain %q", tt.jsonPath, want)
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
			assert.Equal(t, tt.want, result, "Extracted enum values should match expected for input %q", tt.input)
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
			name:      "nested path - child not in yaml returns empty",
			yaml:      "engine: copilot\n",
			path:      "/permissions/issues",
			wantValue: "",
		},
		{
			name:      "nested path - extracts value under parent key",
			yaml:      "permissions:\n  contents: raed\n  issues: write\n",
			path:      "/permissions/contents",
			wantValue: "raed",
		},
		{
			name:      "nested path - second child key",
			yaml:      "permissions:\n  contents: read\n  issues: neno\n",
			path:      "/permissions/issues",
			wantValue: "neno",
		},
		{
			name:      "nested path - single-quoted value",
			yaml:      "permissions:\n  contents: 'raed'\n",
			path:      "/permissions/contents",
			wantValue: "raed",
		},
		{
			name:      "nested path - double-quoted value",
			yaml:      "permissions:\n  contents: \"raed\"\n",
			path:      "/permissions/contents",
			wantValue: "raed",
		},
		{
			name:      "three-level path returns empty",
			yaml:      "a:\n  b:\n    c: value\n",
			path:      "/a/b/c",
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
		{
			name:      "top-level key with only nested block returns empty - no inline scalar value",
			yaml:      "permissions:\n  contents: raed\n",
			path:      "/permissions",
			wantValue: "",
		},
		{
			// Ensures column-0 anchoring: a nested key with the same name must not
			// satisfy a top-level path request.
			name:      "indented key with same name does not match top-level path",
			yaml:      "parent:\n  engine: nested-value\nengine: top-value\n",
			path:      "/engine",
			wantValue: "top-value",
		},
		{
			// Grandchild key must not be returned for a direct-child path.
			name:      "nested path - grandchild key not returned for child path",
			yaml:      "permissions:\n  nested:\n    contents: grandchild\n  contents: direct\n",
			path:      "/permissions/contents",
			wantValue: "direct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractYAMLValueAtPath(tt.yaml, tt.path)
			assert.Equal(t, tt.wantValue, result, "extractYAMLValueAtPath(%q, %q)", tt.yaml, tt.path)
		})
	}
}

// TestExtractEnumConstraintPath tests that the correct JSON path is extracted from
// enum constraint messages, including nested paths embedded in oneOf error messages.
func TestExtractEnumConstraintPath(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		fallbackPath string
		wantPath     string
	}{
		{
			name:         "simple enum error uses fallback path",
			errorMessage: "value must be one of 'claude', 'copilot'",
			fallbackPath: "/engine",
			wantPath:     "/engine",
		},
		{
			name:         "nested enum constraint extracted from oneOf message",
			errorMessage: "'oneOf' failed, none matched\n  - at '/permissions': got object, want string\n  - at '/permissions/contents': value must be one of 'read', 'write', 'none'",
			fallbackPath: "/permissions",
			wantPath:     "/permissions/contents",
		},
		{
			name:         "nested path with issues scope",
			errorMessage: "  - at '/permissions/issues': value must be one of 'read', 'write', 'none'",
			fallbackPath: "/permissions",
			wantPath:     "/permissions/issues",
		},
		{
			name:         "no enum path pattern uses fallback",
			errorMessage: "got object, want string",
			fallbackPath: "/engine",
			wantPath:     "/engine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractEnumConstraintPath(tt.errorMessage, tt.fallbackPath)
			assert.Equal(t, tt.wantPath, result, "extractEnumConstraintPath() should return expected path")
		})
	}
}

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
			require.NotNil(t, result, "generateExampleFromSchema should return non-nil result")
			exampleJSON, err := json.Marshal(result)
			require.NoError(t, err, "Failed to marshal example result to JSON")
			assert.Contains(t, string(exampleJSON), tt.wantContains, "Example JSON should contain %q", tt.wantContains)
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
	require.NoError(t, json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc), "Failed to unmarshal schema")

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
			require.True(t, ok, "Field %q should be found in collected schema property paths", tt.fieldName)
			assert.Contains(t, paths, tt.wantParentPath, "Field %q should have parent path %q", tt.fieldName, tt.wantParentPath)
		})
	}
}

func TestFindFieldLocationsInSchema(t *testing.T) {
	var schemaDoc any
	require.NoError(t, json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc), "Failed to unmarshal schema")

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
			require.Len(t, results, tt.wantLen, "findFieldLocationsInSchema(%q, %q) should return %d results", tt.targetField, tt.currentPath, tt.wantLen)
			if tt.wantLen == 0 {
				return
			}
			assert.Equal(t, tt.wantPath, results[0].SchemaPath, "First result SchemaPath should match expected")
			assert.Equal(t, tt.wantField, results[0].FieldName, "First result FieldName should match expected")
			assert.Equal(t, tt.wantDist, results[0].Distance, "First result Distance should match expected")
		})
	}
}

func TestGeneratePathLocationSuggestion(t *testing.T) {
	var schemaDoc any
	require.NoError(t, json.Unmarshal([]byte(schemaWithNestedFields), &schemaDoc), "Failed to unmarshal schema")

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
				assert.Empty(t, result, "Expected empty path location suggestion")
				return
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Path location suggestion should contain %q", want)
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
				assert.Empty(t, result, "Expected empty schema suggestion result")
				return
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, result, want, "Schema suggestion result should contain %q", want)
			}
		})
	}
}

// TestExtractSchemaExamples tests that examples are extracted from the schema for a given JSON path.
func TestExtractSchemaExamples(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"timeout-minutes": {
				"type": "integer",
				"minimum": 1,
				"examples": [5, 10, 30]
			},
			"name": {
				"type": "string",
				"examples": ["My Workflow"]
			},
			"no-examples": {
				"type": "integer"
			}
		}
	}`
	var schemaDoc any
	require.NoError(t, json.Unmarshal([]byte(schemaJSON), &schemaDoc), "Failed to parse schema JSON")

	tests := []struct {
		name     string
		jsonPath string
		want     []string
	}{
		{
			name:     "integer field with examples",
			jsonPath: "/timeout-minutes",
			want:     []string{"5", "10", "30"},
		},
		{
			name:     "string field with examples",
			jsonPath: "/name",
			want:     []string{"My Workflow"},
		},
		{
			name:     "field without examples",
			jsonPath: "/no-examples",
			want:     nil,
		},
		{
			name:     "non-existent path",
			jsonPath: "/does-not-exist",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSchemaExamples(schemaDoc, tt.jsonPath)
			assert.Equal(t, tt.want, got, "extractSchemaExamples(%q) should return expected examples", tt.jsonPath)
		})
	}
}

// TestGenerateSchemaBasedSuggestions_RangeConstraintExamples tests that examples are surfaced
// for minimum/maximum constraint violations.
// pathInfo.Message from jsonschema has the form "at '/path': minimum: got X, want Y".
func TestGenerateSchemaBasedSuggestions_RangeConstraintExamples(t *testing.T) {
	schemaJSON := `{
		"type": "object",
		"properties": {
			"timeout-minutes": {
				"type": "integer",
				"minimum": 1,
				"examples": [5, 10, 30]
			},
			"retry-count": {
				"type": "integer",
				"minimum": 0,
				"maximum": 10
			}
		}
	}`

	tests := []struct {
		name         string
		errorMessage string
		jsonPath     string
		wantContains string
		wantEmpty    bool
	}{
		{
			name:         "minimum violation with examples (bare form)",
			errorMessage: "minimum: got -1, want 1",
			jsonPath:     "/timeout-minutes",
			wantContains: "Example values: 5, 10, 30",
		},
		{
			name:         "minimum violation with examples (at-path prefix form from jsonschema)",
			errorMessage: "at '/timeout-minutes': minimum: got -1, want 1",
			jsonPath:     "/timeout-minutes",
			wantContains: "Example values: 5, 10, 30",
		},
		{
			name:         "maximum violation with examples surfaced",
			errorMessage: "at '/timeout-minutes': maximum: got 15, want 10",
			jsonPath:     "/timeout-minutes",
			wantContains: "Example values: 5, 10, 30",
		},
		{
			name:         "minimum violation without examples — range handler skips, type handler may fire",
			errorMessage: "at '/retry-count': minimum: got -1, want 0",
			jsonPath:     "/retry-count",
			// The range-constraint handler finds no examples and skips.
			// The type-error handler may still produce a generic "Expected format: …" hint.
			// What must NOT happen is surfacing "Example values: …" (there are none).
			wantContains: "",
			wantEmpty:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateSchemaBasedSuggestions(schemaJSON, tt.errorMessage, tt.jsonPath, "")
			if tt.wantEmpty {
				assert.Empty(t, result, "Expected empty suggestion result")
				return
			}
			if tt.wantContains != "" {
				assert.Contains(t, result, tt.wantContains, "Suggestion result should contain %q", tt.wantContains)
			}
			// When wantContains is "", we only assert "Example values" is absent (no false examples)
			if tt.wantContains == "" {
				assert.NotContains(t, result, "Example values:", "Suggestion result should not contain spurious 'Example values:' section")
			}
		})
	}
}
