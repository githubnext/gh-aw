package mapper

import (
	"testing"
)

func TestDecodeJSONPointer(t *testing.T) {
	tests := []struct {
		name     string
		pointer  string
		expected []string
		hasError bool
	}{
		{
			name:     "empty pointer",
			pointer:  "",
			expected: []string{},
			hasError: false,
		},
		{
			name:     "root pointer",
			pointer:  "/",
			expected: []string{},
			hasError: false,
		},
		{
			name:     "simple path",
			pointer:  "/jobs/build/steps/0/uses",
			expected: []string{"jobs", "build", "steps", "0", "uses"},
			hasError: false,
		},
		{
			name:     "path with escapes",
			pointer:  "/path~0with~1slash",
			expected: []string{"path~with/slash"},
			hasError: false,
		},
		{
			name:     "invalid pointer without leading slash",
			pointer:  "jobs/build",
			expected: nil,
			hasError: true,
		},
		{
			name:     "path with empty segments",
			pointer:  "/jobs//steps",
			expected: []string{"jobs", "", "steps"},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts, err := decodeJSONPointer(tt.pointer)

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(parts) != len(tt.expected) {
				t.Fatalf("Expected %d parts, got %d: %v", len(tt.expected), len(parts), parts)
			}

			for i, expected := range tt.expected {
				if parts[i] != expected {
					t.Errorf("Part %d: expected %q, got %q", i, expected, parts[i])
				}
			}
		})
	}
}

func TestIsIndex(t *testing.T) {
	tests := []struct {
		name     string
		segment  string
		expected bool
	}{
		{"zero", "0", true},
		{"positive integer", "123", true},
		{"negative integer", "-1", false}, // JSON pointers don't support negative indices
		{"string", "name", false},
		{"empty", "", false},
		{"float", "1.5", false},
		{"mixed", "1a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isIndex(tt.segment)
			if result != tt.expected {
				t.Errorf("isIndex(%q) = %v, expected %v", tt.segment, result, tt.expected)
			}
		})
	}
}

// Integration tests for MapErrorToSpans
func TestMapErrorToSpans(t *testing.T) {
	tests := []struct {
		name          string
		yaml          string
		instancePath  string
		meta          ErrorMeta
		expectSpans   int
		minConfidence float64
		shouldContain string // substring that should be in the reason
	}{
		{
			name: "type mismatch on simple value",
			yaml: `name: "test"
version: "1.0"
number: "should be int"`,
			instancePath: "/number",
			meta: ErrorMeta{
				Kind:     "type",
				Property: "number",
			},
			expectSpans:   1,
			minConfidence: 0.8,
			shouldContain: "type mismatch",
		},
		{
			name: "missing required property",
			yaml: `name: "test"
version: "1.0"`,
			instancePath: "/required_field",
			meta: ErrorMeta{
				Kind:     "required",
				Property: "required_field",
			},
			expectSpans:   1,
			minConfidence: 0.5,
			shouldContain: "insertion anchor",
		},
		{
			name: "additional property",
			yaml: `name: "test"
version: "1.0"
extra_field: "not allowed"`,
			instancePath: "/extra_field",
			meta: ErrorMeta{
				Kind:     "additionalProperties",
				Property: "extra_field",
			},
			expectSpans:   1,
			minConfidence: 0.6,
			shouldContain: "property",
		},
		{
			name: "array index access",
			yaml: `items:
  - name: "first"
  - name: "second"
  - name: "third"`,
			instancePath: "/items/1/name",
			meta: ErrorMeta{
				Kind: "type",
			},
			expectSpans:   1,
			minConfidence: 0.8,
			shouldContain: "type mismatch",
		},
		{
			name: "array index out of range",
			yaml: `items:
  - name: "first"
  - name: "second"`,
			instancePath: "/items/5/name",
			meta: ErrorMeta{
				Kind: "type",
			},
			expectSpans:   1,
			minConfidence: 0.2,
			shouldContain: "",
		},
		{
			name: "nested object traversal",
			yaml: `workflow:
  jobs:
    build:
      steps:
        - name: "checkout"
          uses: "actions/checkout@v2"`,
			instancePath: "/workflow/jobs/build/steps/0/uses",
			meta: ErrorMeta{
				Kind: "type",
			},
			expectSpans:   1,
			minConfidence: 0.8,
			shouldContain: "type mismatch",
		},
		{
			name:         "empty document",
			yaml:         "",
			instancePath: "/any/path",
			meta: ErrorMeta{
				Kind: "required",
			},
			expectSpans:   1,
			minConfidence: 0.1,
			shouldContain: "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := MapErrorToSpans([]byte(tt.yaml), tt.instancePath, tt.meta)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(spans) < tt.expectSpans {
				t.Errorf("Expected at least %d spans, got %d", tt.expectSpans, len(spans))
			}

			if len(spans) > 0 {
				firstSpan := spans[0]
				if firstSpan.Confidence < tt.minConfidence {
					t.Errorf("Expected confidence >= %f, got %f", tt.minConfidence, firstSpan.Confidence)
				}

				if tt.shouldContain != "" && !containsSubstring(firstSpan.Reason, tt.shouldContain) {
					t.Errorf("Expected reason to contain %q, got %q", tt.shouldContain, firstSpan.Reason)
				}

				// Validate span positions are reasonable
				if firstSpan.StartLine < 1 {
					t.Errorf("Invalid start line: %d", firstSpan.StartLine)
				}
				if firstSpan.StartCol < 1 {
					t.Errorf("Invalid start column: %d", firstSpan.StartCol)
				}
				if firstSpan.EndLine < firstSpan.StartLine {
					t.Errorf("End line (%d) before start line (%d)", firstSpan.EndLine, firstSpan.StartLine)
				}
			}
		})
	}
}

// Test specific error kinds
func TestTypeErrorMapping(t *testing.T) {
	yaml := `config:
  port: "8080"
  debug: "true"`

	spans, err := MapErrorToSpans([]byte(yaml), "/config/port", ErrorMeta{Kind: "type"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	span := spans[0]
	if span.Confidence < 0.8 {
		t.Errorf("Expected high confidence for exact match, got %f", span.Confidence)
	}

	// Should point to the value "8080"
	if span.StartLine != 2 { // port is on line 2
		t.Errorf("Expected line 2, got %d", span.StartLine)
	}
}

func TestRequiredErrorMapping(t *testing.T) {
	yaml := `config:
  port: 8080`

	spans, err := MapErrorToSpans([]byte(yaml), "/config/required_field", ErrorMeta{
		Kind:     "required",
		Property: "required_field",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	span := spans[0]
	if span.Confidence < 0.5 {
		t.Errorf("Expected reasonable confidence for missing property, got %f", span.Confidence)
	}

	if !containsSubstring(span.Reason, "insertion") {
		t.Errorf("Expected insertion anchor reason, got %q", span.Reason)
	}
}

func TestAdditionalPropertiesErrorMapping(t *testing.T) {
	yaml := `config:
  port: 8080
  extra_setting: "not allowed"`

	spans, err := MapErrorToSpans([]byte(yaml), "/config/extra_setting", ErrorMeta{
		Kind:     "additionalProperties",
		Property: "extra_setting",
	})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	span := spans[0]
	if span.Confidence < 0.8 {
		t.Errorf("Expected high confidence for exact key match, got %f", span.Confidence)
	}

	// Should point to the key "extra_setting"
	if span.StartLine != 3 { // extra_setting is on line 3
		t.Errorf("Expected line 3, got %d", span.StartLine)
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return len(haystack) >= len(needle) &&
		indexOfSubstring(haystack, needle) != -1
}

// Helper function to find index of substring
func indexOfSubstring(haystack, needle string) int {
	if len(needle) == 0 {
		return 0
	}
	if len(haystack) < len(needle) {
		return -1
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
