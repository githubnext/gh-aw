//go:build !integration

package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
)

// TestFormatYAMLError tests the new FormatYAMLError function that uses yaml.FormatError()
func TestFormatYAMLError(t *testing.T) {
	tests := []struct {
		name                  string
		yamlContent           string
		frontmatterLineOffset int
		expectedLineCol       string // Expected [line:col] format in output
		expectSourceContext   bool   // Should contain source code lines with | markers
		expectVisualPointer   bool   // Should contain visual ^ pointer
	}{
		{
			name:                  "invalid mapping with offset 1",
			yamlContent:           "invalid: yaml: syntax",
			frontmatterLineOffset: 1,
			expectedLineCol:       "[1:10]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "invalid mapping with offset 5",
			yamlContent:           "invalid: yaml: syntax",
			frontmatterLineOffset: 5,
			expectedLineCol:       "[5:10]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "indentation error",
			yamlContent:           "name: test\n  invalid_indentation: here",
			frontmatterLineOffset: 3,
			expectedLineCol:       "[3:",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
		{
			name:                  "duplicate key",
			yamlContent:           "name: test\nname: duplicate",
			frontmatterLineOffset: 2,
			expectedLineCol:       "[3:1]",
			expectSourceContext:   true,
			expectVisualPointer:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate an actual goccy/go-yaml error
			var result map[string]any
			err := yaml.Unmarshal([]byte(tt.yamlContent), &result)

			if err == nil {
				t.Errorf("Expected YAML parsing to fail for content: %q", tt.yamlContent)
				return
			}

			// Format the error with the new function
			formatted := FormatYAMLError(err, tt.frontmatterLineOffset, tt.yamlContent)

			// Check for expected [line:col] format
			if !strings.Contains(formatted, tt.expectedLineCol) {
				t.Errorf("Expected output to contain '%s', got:\n%s", tt.expectedLineCol, formatted)
			}

			// Check for source context (lines with | markers)
			if tt.expectSourceContext && !strings.Contains(formatted, "|") {
				t.Errorf("Expected output to contain source context with '|' markers, got:\n%s", formatted)
			}

			// Check for visual pointer
			if tt.expectVisualPointer && !strings.Contains(formatted, "^") {
				t.Errorf("Expected output to contain visual pointer '^', got:\n%s", formatted)
			}

			// Verify "already defined at" references also have adjusted line numbers
			if strings.Contains(formatted, "already defined at") {
				if tt.frontmatterLineOffset > 1 && strings.Contains(formatted, "already defined at [1:") {
					t.Errorf("Expected 'already defined at' line numbers to be adjusted, got:\n%s", formatted)
				}
			}

			t.Logf("Formatted error:\n%s", formatted)
		})
	}
}

// TestFormatYAMLErrorAdjustment specifically tests line number adjustment
func TestFormatYAMLErrorAdjustment(t *testing.T) {
	yamlContent := "name: test\nname: duplicate"

	tests := []struct {
		offset             int
		expectedFirstLine  string
		expectedSecondLine string
		expectedDefinedAt  string
	}{
		{
			offset:             1,
			expectedFirstLine:  "   1 |",
			expectedSecondLine: ">  2 |",
			expectedDefinedAt:  "already defined at [1:1]",
		},
		{
			offset:             5,
			expectedFirstLine:  "   5 |",
			expectedSecondLine: ">  6 |",
			expectedDefinedAt:  "already defined at [5:1]",
		},
		{
			offset:             10,
			expectedFirstLine:  "  10 |",
			expectedSecondLine: "> 11 |",
			expectedDefinedAt:  "already defined at [10:1]",
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("offset_%d", tt.offset), func(t *testing.T) {
			var result map[string]any
			err := yaml.Unmarshal([]byte(yamlContent), &result)

			if err == nil {
				t.Errorf("Expected YAML parsing to fail")
				return
			}

			formatted := FormatYAMLError(err, tt.offset, yamlContent)

			// Check first line number
			if !strings.Contains(formatted, tt.expectedFirstLine) {
				t.Errorf("Expected first line number format '%s', got:\n%s", tt.expectedFirstLine, formatted)
			}

			// Check second line number
			if !strings.Contains(formatted, tt.expectedSecondLine) {
				t.Errorf("Expected second line number format '%s', got:\n%s", tt.expectedSecondLine, formatted)
			}

			// Check "already defined at" reference
			if !strings.Contains(formatted, tt.expectedDefinedAt) {
				t.Errorf("Expected 'already defined at' reference '%s', got:\n%s", tt.expectedDefinedAt, formatted)
			}

			t.Logf("Formatted error (offset %d):\n%s", tt.offset, formatted)
		})
	}
}

// TestTranslateYAMLError tests that cryptic goccy/go-yaml parser messages are translated
// to user-friendly descriptions, and that source context lines are left untouched.
func TestTranslateYAMLError(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		contains           string // expected substring in translated output
		excludes           string // must NOT appear in the first (header) line
		sourceContextCheck string // if non-empty, must appear in the source context (after first line)
	}{
		{
			name:     "unexpected key name translated",
			input:    "[1:1] unexpected key name\n>  1 | engine claude\n       ^",
			contains: "missing ':' after key",
			excludes: "unexpected key name",
		},
		{
			name:     "mapping value not allowed translated",
			input:    "[1:6] mapping value is not allowed in this context\n>  1 | key: value: extra\n            ^",
			contains: "unexpected ':'",
			excludes: "mapping value is not allowed in this context",
		},
		{
			name:     "string used where mapping expected translated",
			input:    "[1:1] string was used where mapping is expected\n>  1 | name value\n       ^",
			contains: "expected a YAML mapping",
			excludes: "string was used where mapping is expected",
		},
		{
			name:     "tab character error translated",
			input:    "[1:7] tab character cannot use as a map key directly\n>  1 | \tengine: claude\n             ^",
			contains: "tab character in key",
			excludes: "tab character cannot use as a map key directly",
		},
		{
			name:     "unrecognized messages are returned unchanged",
			input:    "[2:1] mapping key \"name\" already defined at [1:1]\n>  2 | name: dup\n       ^",
			contains: "already defined",
		},
		{
			name:     "empty string is returned unchanged",
			input:    "",
			contains: "",
		},
		{
			name:  "pattern in source context line is not replaced",
			input: "[1:1] unexpected key name\n>  1 | unexpected key name: here\n       ^",
			// The header should be translated, but source context must remain untouched.
			contains:           "missing ':' after key",
			sourceContextCheck: "unexpected key name: here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := translateYAMLError(tt.input)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains,
					"translateYAMLError should contain %q\nResult: %s", tt.contains, result)
			}
			// Verify source context lines are preserved untouched when applicable.
			if tt.sourceContextCheck != "" {
				_, rest, _ := strings.Cut(result, "\n")
				assert.Contains(t, rest, tt.sourceContextCheck,
					"source context should be preserved unchanged\nContext: %s", rest)
			}
			// For cases with excludes, the pattern should not appear in the header line.
			if tt.excludes != "" {
				firstLine, _, _ := strings.Cut(result, "\n")
				assert.NotContains(t, firstLine, tt.excludes,
					"translateYAMLError header should not contain %q\nHeader: %s", tt.excludes, firstLine)
			}
		})
	}
}

// TestTranslateYAMLMessage tests the exported TranslateYAMLMessage function used
// by both the parser and workflow packages.
func TestTranslateYAMLMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
		excludes string
	}{
		{
			name:     "unexpected key name",
			input:    "unexpected key name",
			contains: "missing ':' after key",
			excludes: "unexpected key name",
		},
		{
			name:     "non-map value is specified",
			input:    "non-map value is specified",
			contains: "expected a YAML mapping",
			excludes: "non-map value is specified",
		},
		{
			name:     "found character that cannot start any token",
			input:    "found character that cannot start any token",
			contains: "invalid character",
			excludes: "found character that cannot start any token",
		},
		{
			name:     "unrecognized message is unchanged",
			input:    "some other error",
			contains: "some other error",
		},
		{
			name:     "empty string is unchanged",
			input:    "",
			contains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TranslateYAMLMessage(tt.input)
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains,
					"TranslateYAMLMessage should contain %q\nResult: %s", tt.contains, result)
			}
			if tt.excludes != "" {
				assert.NotContains(t, result, tt.excludes,
					"TranslateYAMLMessage should not contain %q\nResult: %s", tt.excludes, result)
			}
		})
	}
}

// TestFormatYAMLErrorTranslation verifies that FormatYAMLError applies translations
// to the underlying goccy/go-yaml error messages.
func TestFormatYAMLErrorTranslation(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		shouldContain string
		shouldExclude string
	}{
		{
			name:          "missing colon translated",
			yamlContent:   "engine claude\nmodel: gpt-4",
			shouldContain: "missing ':' after key",
			shouldExclude: "unexpected key name",
		},
		{
			name:          "extra colon translated",
			yamlContent:   "key: value: extra",
			shouldContain: "unexpected ':'",
			shouldExclude: "mapping value is not allowed in this context",
		},
		{
			name:          "plain string translated",
			yamlContent:   "not a mapping",
			shouldContain: "expected a YAML mapping",
			shouldExclude: "string was used where mapping is expected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result map[string]any
			err := yaml.Unmarshal([]byte(tt.yamlContent), &result)
			if err == nil {
				t.Skipf("Expected YAML parsing to fail for content: %q", tt.yamlContent)
				return
			}
			formatted := FormatYAMLError(err, 1, tt.yamlContent)
			assert.Contains(t, formatted, tt.shouldContain,
				"FormatYAMLError should contain %q\nResult: %s", tt.shouldContain, formatted)
			if tt.shouldExclude != "" {
				assert.NotContains(t, formatted, tt.shouldExclude,
					"FormatYAMLError should not contain %q\nResult: %s", tt.shouldExclude, formatted)
			}
		})
	}
}
