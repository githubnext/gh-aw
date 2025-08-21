package mapper

import (
	"testing"
)

// TestComplexYAMLStructures tests mapping with complex YAML structures
func TestComplexYAMLStructures(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		instancePath string
		meta         ErrorMeta
		expectSpans  int
		description  string
	}{
		{
			name:         "flow style mapping",
			yaml:         `config: {port: 8080, host: "localhost", debug: true}`,
			instancePath: "/config/port",
			meta:         ErrorMeta{Kind: "type"},
			expectSpans:  1,
			description:  "should handle flow style mappings",
		},
		{
			name:         "flow style sequence",
			yaml:         `items: [first, second, third]`,
			instancePath: "/items/1",
			meta:         ErrorMeta{Kind: "type"},
			expectSpans:  1,
			description:  "should handle flow style sequences",
		},
		{
			name: "mixed nested structures",
			yaml: `
workflow:
  name: "test"
  on: [push, pull_request]
  jobs:
    build:
      runs-on: ubuntu-latest
      steps:
        - name: "checkout"
          uses: "actions/checkout@v2"
        - {name: "build", run: "make build"}`,
			instancePath: "/workflow/jobs/build/steps/1/run",
			meta:         ErrorMeta{Kind: "type"},
			expectSpans:  1,
			description:  "should handle mixed flow and block styles",
		},
		{
			name: "deeply nested path",
			yaml: `
a:
  b:
    c:
      d:
        e:
          f: "deep value"`,
			instancePath: "/a/b/c/d/e/f",
			meta:         ErrorMeta{Kind: "type"},
			expectSpans:  1,
			description:  "should handle deeply nested paths",
		},
		{
			name: "array with objects",
			yaml: `
users:
  - name: "alice"
    age: 30
  - name: "bob"
    age: 25`,
			instancePath: "/users/0/age",
			meta:         ErrorMeta{Kind: "type"},
			expectSpans:  1,
			description:  "should handle arrays containing objects",
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
				span := spans[0]
				if span.StartLine < 1 || span.StartCol < 1 {
					t.Errorf("Invalid span position: line %d, col %d", span.StartLine, span.StartCol)
				}
			}

			t.Logf("%s: %v", tt.description, spans)
		})
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		instancePath string
		meta         ErrorMeta
		description  string
	}{
		{
			name: "invalid yaml",
			yaml: `invalid: yaml: content
  with: bad indentation`,
			instancePath: "/invalid",
			meta:         ErrorMeta{Kind: "type"},
			description:  "should handle invalid YAML gracefully",
		},
		{
			name: "empty string keys",
			yaml: `"": "empty key"
normal: "value"`,
			instancePath: "/",
			meta:         ErrorMeta{Kind: "type"},
			description:  "should handle empty string keys",
		},
		{
			name: "numeric keys",
			yaml: `123: "numeric key"
456: "another numeric key"`,
			instancePath: "/123",
			meta:         ErrorMeta{Kind: "type"},
			description:  "should handle numeric keys",
		},
		{
			name: "keys with special characters",
			yaml: `"key-with-dashes": "value1"
"key.with.dots": "value2"
"key with spaces": "value3"`,
			instancePath: "/key-with-dashes",
			meta:         ErrorMeta{Kind: "type"},
			description:  "should handle keys with special characters",
		},
		{
			name:         "very large array index",
			yaml:         `items: [a, b, c]`,
			instancePath: "/items/999999",
			meta:         ErrorMeta{Kind: "type"},
			description:  "should handle out-of-bounds array access gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := MapErrorToSpans([]byte(tt.yaml), tt.instancePath, tt.meta)

			// For invalid YAML, we expect an error
			if tt.name == "invalid yaml" && err == nil {
				t.Error("Expected error for invalid YAML")
				return
			}

			// For other cases, we should get some result
			if tt.name != "invalid yaml" {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}

				if len(spans) == 0 {
					t.Error("Expected at least one span")
				}
			}

			t.Logf("%s: spans=%v, err=%v", tt.description, spans, err)
		})
	}
}

// TestAllErrorKinds tests all supported error kinds
func TestAllErrorKinds(t *testing.T) {
	yaml := `
config:
  port: 8080
  host: "localhost"
  extra: "not allowed"
items:
  - name: "first"
  - name: "second"
`

	tests := []struct {
		name          string
		instancePath  string
		meta          ErrorMeta
		minConfidence float64
	}{
		{
			name:          "type error",
			instancePath:  "/config/port",
			meta:          ErrorMeta{Kind: "type"},
			minConfidence: 0.8,
		},
		{
			name:          "required error",
			instancePath:  "/config/missing",
			meta:          ErrorMeta{Kind: "required", Property: "missing"},
			minConfidence: 0.5,
		},
		{
			name:          "additionalProperties error",
			instancePath:  "/config/extra",
			meta:          ErrorMeta{Kind: "additionalProperties", Property: "extra"},
			minConfidence: 0.6,
		},
		{
			name:          "oneOf error",
			instancePath:  "/config",
			meta:          ErrorMeta{Kind: "oneOf"},
			minConfidence: 0.5,
		},
		{
			name:          "anyOf error",
			instancePath:  "/items/0",
			meta:          ErrorMeta{Kind: "anyOf"},
			minConfidence: 0.5,
		},
		{
			name:          "unknown error kind",
			instancePath:  "/config/host",
			meta:          ErrorMeta{Kind: "customError"},
			minConfidence: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := MapErrorToSpans([]byte(yaml), tt.instancePath, tt.meta)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(spans) == 0 {
				t.Fatal("Expected at least one span")
			}

			span := spans[0]
			if span.Confidence < tt.minConfidence {
				t.Errorf("Expected confidence >= %f, got %f for %s",
					tt.minConfidence, span.Confidence, tt.name)
			}

			t.Logf("%s: line %d, col %d, confidence %f, reason: %s",
				tt.name, span.StartLine, span.StartCol, span.Confidence, span.Reason)
		})
	}
}

// TestConfidenceScoring tests confidence scoring behavior
func TestConfidenceScoring(t *testing.T) {
	yaml := `
config:
  port: 8080
  host: "localhost"
`

	// Test exact matches should have high confidence
	spans, err := MapErrorToSpans([]byte(yaml), "/config/port", ErrorMeta{Kind: "type"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	if spans[0].Confidence < 0.9 {
		t.Errorf("Expected high confidence for exact match, got %f", spans[0].Confidence)
	}

	// Test missing properties should have lower confidence
	spans, err = MapErrorToSpans([]byte(yaml), "/config/missing",
		ErrorMeta{Kind: "required", Property: "missing"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	if spans[0].Confidence > 0.8 {
		t.Errorf("Expected lower confidence for missing property, got %f", spans[0].Confidence)
	}

	// Test fallback cases should have lowest confidence
	spans, err = MapErrorToSpans([]byte(""), "/any/path", ErrorMeta{Kind: "type"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(spans) == 0 {
		t.Fatal("Expected at least one span")
	}

	if spans[0].Confidence > 0.3 {
		t.Errorf("Expected low confidence for fallback, got %f", spans[0].Confidence)
	}
}

// TestSpanPositions tests that span positions are accurate
func TestSpanPositions(t *testing.T) {
	yaml := `name: "test"
version: "1.0"
config:
  port: 8080
  host: "localhost"`

	tests := []struct {
		name         string
		instancePath string
		meta         ErrorMeta
		expectedLine int
	}{
		{
			name:         "first property",
			instancePath: "/name",
			meta:         ErrorMeta{Kind: "type"},
			expectedLine: 1,
		},
		{
			name:         "second property",
			instancePath: "/version",
			meta:         ErrorMeta{Kind: "type"},
			expectedLine: 2,
		},
		{
			name:         "nested property",
			instancePath: "/config/port",
			meta:         ErrorMeta{Kind: "type"},
			expectedLine: 4,
		},
		{
			name:         "last property",
			instancePath: "/config/host",
			meta:         ErrorMeta{Kind: "type"},
			expectedLine: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spans, err := MapErrorToSpans([]byte(yaml), tt.instancePath, tt.meta)

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(spans) == 0 {
				t.Fatal("Expected at least one span")
			}

			span := spans[0]
			if span.StartLine != tt.expectedLine {
				t.Errorf("Expected line %d, got %d", tt.expectedLine, span.StartLine)
			}

			// Validate basic span constraints
			if span.StartCol < 1 {
				t.Errorf("Invalid start column: %d", span.StartCol)
			}
			if span.EndCol < span.StartCol {
				t.Errorf("End column (%d) before start column (%d)", span.EndCol, span.StartCol)
			}
			if span.EndLine < span.StartLine {
				t.Errorf("End line (%d) before start line (%d)", span.EndLine, span.StartLine)
			}
		})
	}
}
